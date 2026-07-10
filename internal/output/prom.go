package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

// PromFormatter formats output in the Prometheus text exposition format.
//
// Durations are emitted in seconds (the Prometheus base-unit convention), unlike
// the table/json/csv formats which use milliseconds. The output is suitable for
// the node_exporter textfile collector or a Pushgateway.
type PromFormatter struct {
	verbose bool
}

// NewPromFormatter creates a new Prometheus formatter.
func NewPromFormatter(verbose bool) *PromFormatter {
	return &PromFormatter{verbose: verbose}
}

// promSeconds converts a Duration to fractional seconds.
func promSeconds(d client.Duration) float64 {
	return float64(d) / float64(time.Second)
}

// escapeLabelValue escapes a Prometheus label value per the exposition format
// (backslash, double-quote and newline must be escaped).
func escapeLabelValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// Format returns the single-request Prometheus output as a string.
func (f *PromFormatter) Format(timing *client.TimingBreakdown) (string, error) {
	var b strings.Builder
	if err := f.Write(&b, timing); err != nil {
		return "", err
	}
	return b.String(), nil
}

// FormatMultiple returns the aggregate Prometheus output as a string.
func (f *PromFormatter) FormatMultiple(stats *metrics.Stats) (string, error) {
	var b strings.Builder
	if err := f.WriteMultiple(&b, stats); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Write emits per-phase gauges for a single request.
func (f *PromFormatter) Write(w io.Writer, timing *client.TimingBreakdown) error {
	url := escapeLabelValue(timing.URL)

	fmt.Fprintln(w, "# HELP gocurl_request_phase_seconds Duration of each request phase in seconds.")
	fmt.Fprintln(w, "# TYPE gocurl_request_phase_seconds gauge")
	phases := []struct {
		name string
		val  client.Duration
	}{
		{"dns", timing.DNSLookup},
		{"tcp", timing.TCPConnection},
		{"tls", timing.TLSHandshake},
		{"server", timing.ServerProcessing},
		{"transfer", timing.ContentTransfer},
	}
	for _, p := range phases {
		fmt.Fprintf(w, "gocurl_request_phase_seconds{url=\"%s\",phase=%q} %g\n", url, p.name, promSeconds(p.val))
	}

	fmt.Fprintln(w, "# HELP gocurl_request_duration_seconds Total request duration in seconds.")
	fmt.Fprintln(w, "# TYPE gocurl_request_duration_seconds gauge")
	fmt.Fprintf(w, "gocurl_request_duration_seconds{url=\"%s\"} %g\n", url, promSeconds(timing.Total))

	fmt.Fprintln(w, "# HELP gocurl_response_status_code HTTP status code of the response (0 on transport error).")
	fmt.Fprintln(w, "# TYPE gocurl_response_status_code gauge")
	fmt.Fprintf(w, "gocurl_response_status_code{url=\"%s\"} %d\n", url, timing.StatusCode)

	fmt.Fprintln(w, "# HELP gocurl_response_size_bytes Response body size in bytes.")
	fmt.Fprintln(w, "# TYPE gocurl_response_size_bytes gauge")
	fmt.Fprintf(w, "gocurl_response_size_bytes{url=\"%s\"} %d\n", url, timing.ResponseSize)

	// A request is "successful" when it completed without a transport error and
	// returned a non-error HTTP status (< 400), matching the load-test semantics.
	success := 0
	if timing.Error == "" && timing.StatusCode > 0 && timing.StatusCode < 400 {
		success = 1
	}
	fmt.Fprintln(w, "# HELP gocurl_request_success Whether the request succeeded (1) or failed (0).")
	fmt.Fprintln(w, "# TYPE gocurl_request_success gauge")
	fmt.Fprintf(w, "gocurl_request_success{url=\"%s\"} %d\n", url, success)

	return nil
}

// WriteMultiple emits aggregate load-test metrics. Metrics are aggregate across
// all tested URLs, so no url label is applied.
func (f *PromFormatter) WriteMultiple(w io.Writer, stats *metrics.Stats) error {
	fmt.Fprintln(w, "# HELP gocurl_requests_total Total number of requests performed.")
	fmt.Fprintln(w, "# TYPE gocurl_requests_total counter")
	fmt.Fprintf(w, "gocurl_requests_total %d\n", stats.TotalRequests)

	fmt.Fprintln(w, "# HELP gocurl_requests_successful_total Requests that completed without error and returned status < 400.")
	fmt.Fprintln(w, "# TYPE gocurl_requests_successful_total counter")
	fmt.Fprintf(w, "gocurl_requests_successful_total %d\n", stats.SuccessfulRequests)

	fmt.Fprintln(w, "# HELP gocurl_requests_failed_total Requests that errored or returned status >= 400.")
	fmt.Fprintln(w, "# TYPE gocurl_requests_failed_total counter")
	fmt.Fprintf(w, "gocurl_requests_failed_total %d\n", stats.FailedRequests)

	fmt.Fprintln(w, "# HELP gocurl_error_rate Fraction of requests that failed (0..1).")
	fmt.Fprintln(w, "# TYPE gocurl_error_rate gauge")
	fmt.Fprintf(w, "gocurl_error_rate %g\n", stats.ErrorRate)

	fmt.Fprintln(w, "# HELP gocurl_requests_per_second Achieved request throughput over the measurement window.")
	fmt.Fprintln(w, "# TYPE gocurl_requests_per_second gauge")
	fmt.Fprintf(w, "gocurl_requests_per_second %g\n", stats.RequestsPerSecond)

	// Latency as a Prometheus summary. Quantiles come from the observed
	// distribution; _sum/_count make it a valid summary metric.
	fmt.Fprintln(w, "# HELP gocurl_request_duration_seconds Request latency distribution in seconds.")
	fmt.Fprintln(w, "# TYPE gocurl_request_duration_seconds summary")
	quantiles := []struct {
		q string
		v client.Duration
	}{
		{"0.5", stats.P50},
		{"0.9", stats.P90},
		{"0.95", stats.P95},
		{"0.99", stats.P99},
	}
	if stats.P999 > 0 {
		quantiles = append(quantiles, struct {
			q string
			v client.Duration
		}{"0.999", stats.P999})
	}
	if stats.P9999 > 0 {
		quantiles = append(quantiles, struct {
			q string
			v client.Duration
		}{"0.9999", stats.P9999})
	}
	for _, q := range quantiles {
		fmt.Fprintf(w, "gocurl_request_duration_seconds{quantile=%q} %g\n", q.q, promSeconds(q.v))
	}
	// _sum is derived from the mean (sum = mean * count); the collector does not
	// retain the raw sum, but mean*count is exact to the mean's precision.
	sum := promSeconds(stats.MeanLatency) * float64(stats.TotalRequests)
	fmt.Fprintf(w, "gocurl_request_duration_seconds_sum %g\n", sum)
	fmt.Fprintf(w, "gocurl_request_duration_seconds_count %d\n", stats.TotalRequests)

	fmt.Fprintln(w, "# HELP gocurl_request_duration_min_seconds Minimum observed latency in seconds.")
	fmt.Fprintln(w, "# TYPE gocurl_request_duration_min_seconds gauge")
	fmt.Fprintf(w, "gocurl_request_duration_min_seconds %g\n", promSeconds(stats.MinLatency))

	fmt.Fprintln(w, "# HELP gocurl_request_duration_max_seconds Maximum observed latency in seconds.")
	fmt.Fprintln(w, "# TYPE gocurl_request_duration_max_seconds gauge")
	fmt.Fprintf(w, "gocurl_request_duration_max_seconds %g\n", promSeconds(stats.MaxLatency))

	fmt.Fprintln(w, "# HELP gocurl_request_duration_mean_seconds Mean latency in seconds.")
	fmt.Fprintln(w, "# TYPE gocurl_request_duration_mean_seconds gauge")
	fmt.Fprintf(w, "gocurl_request_duration_mean_seconds %g\n", promSeconds(stats.MeanLatency))

	// Responses by status code (sorted for deterministic output).
	if len(stats.StatusCodes) > 0 {
		fmt.Fprintln(w, "# HELP gocurl_responses_total Number of responses by HTTP status code.")
		fmt.Fprintln(w, "# TYPE gocurl_responses_total counter")
		codes := make([]int, 0, len(stats.StatusCodes))
		for code := range stats.StatusCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for _, code := range codes {
			fmt.Fprintf(w, "gocurl_responses_total{status=\"%d\"} %d\n", code, stats.StatusCodes[code])
		}
	}

	fmt.Fprintln(w, "# HELP gocurl_response_bytes_total Total response bytes transferred.")
	fmt.Fprintln(w, "# TYPE gocurl_response_bytes_total counter")
	fmt.Fprintf(w, "gocurl_response_bytes_total %d\n", stats.TotalBytes)

	fmt.Fprintln(w, "# HELP gocurl_bytes_per_second Response throughput in bytes per second.")
	fmt.Fprintln(w, "# TYPE gocurl_bytes_per_second gauge")
	fmt.Fprintf(w, "gocurl_bytes_per_second %g\n", stats.BytesPerSecond)

	fmt.Fprintln(w, "# HELP gocurl_measurement_duration_seconds Wall-clock duration of the request window.")
	fmt.Fprintln(w, "# TYPE gocurl_measurement_duration_seconds gauge")
	fmt.Fprintf(w, "gocurl_measurement_duration_seconds %g\n", promSeconds(stats.Duration))

	return nil
}
