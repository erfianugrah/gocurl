package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

func TestPromFormatterSingle(t *testing.T) {
	f := NewPromFormatter(false)
	timing := &client.TimingBreakdown{
		URL:              "https://example.com/api",
		StatusCode:       200,
		DNSLookup:        client.Duration(12 * time.Millisecond),
		TCPConnection:    client.Duration(3 * time.Millisecond),
		TLSHandshake:     client.Duration(20 * time.Millisecond),
		ServerProcessing: client.Duration(30 * time.Millisecond),
		ContentTransfer:  client.Duration(1 * time.Millisecond),
		Total:            client.Duration(66 * time.Millisecond),
		ResponseSize:     559,
	}

	var buf bytes.Buffer
	if err := f.Write(&buf, timing); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	out := buf.String()

	wants := []string{
		"# TYPE gocurl_request_phase_seconds gauge",
		`gocurl_request_phase_seconds{url="https://example.com/api",phase="dns"} 0.012`,
		`gocurl_request_phase_seconds{url="https://example.com/api",phase="server"} 0.03`,
		`gocurl_request_duration_seconds{url="https://example.com/api"} 0.066`,
		`gocurl_response_status_code{url="https://example.com/api"} 200`,
		`gocurl_response_size_bytes{url="https://example.com/api"} 559`,
		`gocurl_request_success{url="https://example.com/api"} 1`,
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q\nGot:\n%s", w, out)
		}
	}
}

func TestPromFormatterSingleFailure(t *testing.T) {
	f := NewPromFormatter(false)

	// Transport error -> success 0.
	var buf bytes.Buffer
	if err := f.Write(&buf, &client.TimingBreakdown{URL: "https://x", StatusCode: 0, Error: "connection refused"}); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !strings.Contains(buf.String(), `gocurl_request_success{url="https://x"} 0`) {
		t.Errorf("transport error should be success 0\nGot:\n%s", buf.String())
	}

	// HTTP 500 -> success 0.
	buf.Reset()
	if err := f.Write(&buf, &client.TimingBreakdown{URL: "https://x", StatusCode: 500}); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !strings.Contains(buf.String(), `gocurl_request_success{url="https://x"} 0`) {
		t.Errorf("HTTP 500 should be success 0\nGot:\n%s", buf.String())
	}
}

func TestPromFormatterLabelEscaping(t *testing.T) {
	f := NewPromFormatter(false)
	var buf bytes.Buffer
	// URL containing a double-quote and backslash must be escaped.
	if err := f.Write(&buf, &client.TimingBreakdown{URL: `https://x/a"b\c`, StatusCode: 200}); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !strings.Contains(buf.String(), `url="https://x/a\"b\\c"`) {
		t.Errorf("label value not escaped\nGot:\n%s", buf.String())
	}
}

func TestPromFormatterMultiple(t *testing.T) {
	f := NewPromFormatter(false)
	stats := &metrics.Stats{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		Duration:           metrics.Duration(2 * time.Second),
		RequestsPerSecond:  50.0,
		MinLatency:         metrics.Duration(10 * time.Millisecond),
		MaxLatency:         metrics.Duration(500 * time.Millisecond),
		MeanLatency:        metrics.Duration(40 * time.Millisecond),
		P50:                metrics.Duration(30 * time.Millisecond),
		P90:                metrics.Duration(80 * time.Millisecond),
		P95:                metrics.Duration(120 * time.Millisecond),
		P99:                metrics.Duration(300 * time.Millisecond),
		ErrorRate:          0.05,
		TotalBytes:         10240,
		BytesPerSecond:     5120.0,
		StatusCodes:        map[int]int{200: 95, 500: 5},
	}

	var buf bytes.Buffer
	if err := f.WriteMultiple(&buf, stats); err != nil {
		t.Fatalf("WriteMultiple() error: %v", err)
	}
	out := buf.String()

	wants := []string{
		"# TYPE gocurl_requests_total counter",
		"gocurl_requests_total 100",
		"gocurl_requests_successful_total 95",
		"gocurl_requests_failed_total 5",
		"gocurl_error_rate 0.05",
		"gocurl_requests_per_second 50",
		"# TYPE gocurl_request_duration_seconds summary",
		`gocurl_request_duration_seconds{quantile="0.5"} 0.03`,
		`gocurl_request_duration_seconds{quantile="0.99"} 0.3`,
		"gocurl_request_duration_seconds_sum 4", // 0.04 * 100
		"gocurl_request_duration_seconds_count 100",
		"gocurl_request_duration_min_seconds 0.01",
		`gocurl_responses_total{status="200"} 95`,
		`gocurl_responses_total{status="500"} 5`,
		"gocurl_response_bytes_total 10240",
		"gocurl_measurement_duration_seconds 2",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q\nGot:\n%s", w, out)
		}
	}

	// P999/P9999 are unset here, so their quantile lines must be absent.
	if strings.Contains(out, `quantile="0.999"`) {
		t.Errorf("unexpected p99.9 quantile line when P999 is zero")
	}
}

func TestGetFormatterProm(t *testing.T) {
	for _, name := range []string{"prom", "prometheus"} {
		fmttr, err := GetFormatter(name, false, "v", "c", "d")
		if err != nil {
			t.Fatalf("GetFormatter(%q) error: %v", name, err)
		}
		if _, ok := fmttr.(*PromFormatter); !ok {
			t.Errorf("GetFormatter(%q) = %T, want *PromFormatter", name, fmttr)
		}
	}
}
