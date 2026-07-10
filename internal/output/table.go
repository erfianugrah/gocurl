package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

// TableFormatter formats output as a table
type TableFormatter struct {
	verbose bool
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(verbose bool) *TableFormatter {
	return &TableFormatter{verbose: verbose}
}

// Format formats a single timing result as a table
func (f *TableFormatter) Format(timing *client.TimingBreakdown) (string, error) {
	var buf strings.Builder
	if err := f.Write(&buf, timing); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Write writes a single timing result as a table to the writer
func (f *TableFormatter) Write(w io.Writer, timing *client.TimingBreakdown) error {
	// Status line. Use a cross for transport failures / error statuses so a
	// failed request is not decorated with a success checkmark.
	if timing.Error != "" {
		fmt.Fprintf(w, "%s %s\n", color.RedString("✗ Error:"), color.RedString(timing.Error))
		if timing.StatusCode != 0 {
			fmt.Fprintf(w, "%s %s\n", color.RedString("✗ Status:"), color.RedString(fmt.Sprintf("%d %s", timing.StatusCode, getStatusText(timing.StatusCode))))
		}
		fmt.Fprintf(w, "%s %s\n", color.RedString("✗ Time:"), formatTimeDuration(time.Duration(timing.Total)))
	} else {
		statusColor := getStatusColor(timing.StatusCode)
		glyph := "✓"
		if timing.StatusCode >= 400 {
			glyph = "✗"
		}
		fmt.Fprintf(w, "%s %s\n", statusColor(glyph+" Status:"), statusColor(fmt.Sprintf("%d %s", timing.StatusCode, getStatusText(timing.StatusCode))))
		fmt.Fprintf(w, "%s %s\n", color.GreenString("✓ Time:"), formatTimeDuration(time.Duration(timing.Total)))
	}

	if timing.ConnectionReused {
		fmt.Fprintf(w, "%s %s\n", color.GreenString("✓ Connection:"), "Reused")
	}

	fmt.Fprintln(w)

	// Waterfall timeline visualization (like Chrome DevTools)
	drawWaterfall(w, timing)

	fmt.Fprintln(w)

	// Timing breakdown table (objective metrics only)
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Timing Breakdown")
	t.AppendHeader(table.Row{"Phase", "Duration", "% of Total"})

	total := timing.Total.Seconds()

	if timing.DNSLookup > 0 {
		pct := (timing.DNSLookup.Seconds() / total) * 100
		t.AppendRow(table.Row{
			"DNS Lookup",
			formatTimeDuration(time.Duration(timing.DNSLookup)),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	if timing.TCPConnection > 0 {
		pct := (timing.TCPConnection.Seconds() / total) * 100
		t.AppendRow(table.Row{
			"TCP Connection",
			formatTimeDuration(time.Duration(timing.TCPConnection)),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	if timing.TLSHandshake > 0 {
		pct := (timing.TLSHandshake.Seconds() / total) * 100
		t.AppendRow(table.Row{
			"TLS Handshake",
			formatTimeDuration(time.Duration(timing.TLSHandshake)),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	if timing.ServerProcessing > 0 {
		pct := (timing.ServerProcessing.Seconds() / total) * 100
		t.AppendRow(table.Row{
			"Server Processing",
			formatTimeDuration(time.Duration(timing.ServerProcessing)),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	if timing.ContentTransfer > 0 {
		pct := (timing.ContentTransfer.Seconds() / total) * 100
		t.AppendRow(table.Row{
			"Content Transfer",
			formatTimeDuration(time.Duration(timing.ContentTransfer)),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	// Account for time not attributed to a named phase (connection acquisition
	// from the pool, gaps between trace events) so the percentage column
	// reconciles to 100%.
	named := timing.DNSLookup + timing.TCPConnection + timing.TLSHandshake + timing.ServerProcessing + timing.ContentTransfer
	if other := time.Duration(timing.Total) - time.Duration(named); other > 0 && timing.Total > 0 {
		pct := (other.Seconds() / total) * 100
		t.AppendRow(table.Row{
			"Other/Setup",
			formatTimeDuration(other),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	t.AppendSeparator()
	t.AppendRow(table.Row{"Total", formatTimeDuration(time.Duration(timing.Total)), "100%"})

	t.SetStyle(table.StyleLight)
	t.Render()

	// Show response headers if captured (from -i flag)
	if len(timing.ResponseHeaders) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s\n", color.CyanString("Response Headers:"))
		for key, value := range timing.ResponseHeaders {
			fmt.Fprintf(w, "  %s: %s\n", key, value)
		}
	}

	// Show response body if captured
	if timing.ResponseBody != "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s\n", color.CyanString("Response Body:"))
		fmt.Fprintln(w, timing.ResponseBody)
	}

	// Additional details in verbose mode
	if f.verbose {
		fmt.Fprintln(w)
		if timing.ResponseSize > 0 {
			fmt.Fprintf(w, "Response Size: %s\n", formatBytes(timing.ResponseSize))
			if timing.ContentLength > 0 {
				fmt.Fprintf(w, "Content Length: %s\n", formatBytes(timing.ContentLength))
			}
		}

		// TLS information
		if timing.TLSVersion != "" {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "%s\n", color.CyanString("TLS Connection:"))
			fmt.Fprintf(w, "  Version: %s\n", timing.TLSVersion)
			fmt.Fprintf(w, "  Cipher: %s\n", timing.TLSCipherSuite)
			if timing.TLSServerName != "" {
				fmt.Fprintf(w, "  SNI: %s\n", timing.TLSServerName)
			}
		}

		// Connection info
		if timing.ConnectionReused {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "%s\n", color.GreenString("Connection was reused from pool"))
			if timing.IdleTime > 0 {
				fmt.Fprintf(w, "  Idle time: %s\n", formatTimeDuration(time.Duration(timing.IdleTime)))
			}
		}
	}

	return nil
}

// FormatMultiple formats multiple timing results as statistics
func (f *TableFormatter) FormatMultiple(stats *metrics.Stats) (string, error) {
	var buf strings.Builder
	if err := f.WriteMultiple(&buf, stats); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteMultiple writes multiple timing results as statistics to the writer
func (f *TableFormatter) WriteMultiple(w io.Writer, stats *metrics.Stats) error {
	// Summary
	fmt.Fprintf(w, "%s\n", color.CyanString("=== Load Test Results ==="))
	fmt.Fprintf(w, "Total Requests: %d\n", stats.TotalRequests)
	fmt.Fprintf(w, "Successful: %s\n", color.GreenString("%d", stats.SuccessfulRequests))
	fmt.Fprintf(w, "Failed: %s\n", color.RedString("%d", stats.FailedRequests))
	fmt.Fprintf(w, "Duration: %s\n", formatDuration(stats.Duration))
	fmt.Fprintf(w, "Requests/sec: %.2f\n\n", stats.RequestsPerSecond)

	// Latency statistics
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetTitle("Latency Statistics")
	t.AppendHeader(table.Row{"Metric", "Value"})
	t.AppendRow(table.Row{"Min", formatDuration(stats.MinLatency)})
	t.AppendRow(table.Row{"Max", formatDuration(stats.MaxLatency)})
	t.AppendRow(table.Row{"Mean", formatDuration(stats.MeanLatency)})
	t.AppendRow(table.Row{"Median (p50)", formatDuration(stats.P50)})
	t.AppendRow(table.Row{"P95", formatDuration(stats.P95)})
	t.AppendRow(table.Row{"P99", formatDuration(stats.P99)})
	t.SetStyle(table.StyleLight)
	t.Render()

	// Status code distribution
	if len(stats.StatusCodes) > 0 {
		fmt.Fprintln(w)
		st := table.NewWriter()
		st.SetOutputMirror(w)
		st.SetTitle("Status Code Distribution")
		st.AppendHeader(table.Row{"Status Code", "Count", "Percentage"})
		for code, count := range stats.StatusCodes {
			pct := (float64(count) / float64(stats.TotalRequests)) * 100
			st.AppendRow(table.Row{
				fmt.Sprintf("%d %s", code, getStatusText(code)),
				count,
				fmt.Sprintf("%.1f%%", pct),
			})
		}
		st.SetStyle(table.StyleLight)
		st.Render()
	}

	return nil
}

// Helper functions

func getStatusColor(code int) func(string, ...interface{}) string {
	switch {
	case code >= 200 && code < 300:
		return color.GreenString
	case code >= 300 && code < 400:
		return color.YellowString
	case code >= 400:
		return color.RedString
	default:
		return color.WhiteString
	}
}

func getStatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return ""
	}
}

func formatDuration(d metrics.Duration) string {
	ms := d.Milliseconds()
	if ms < 1 {
		return fmt.Sprintf("%.2fms", d.Seconds()*1000)
	} else if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Helper functions for time.Duration (for single requests)

func formatTimeDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1 {
		return fmt.Sprintf("%.2fms", d.Seconds()*1000)
	} else if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// drawWaterfall creates a horizontal timeline visualization of request phases
// Similar to Chrome DevTools Network tab waterfall view
func drawWaterfall(w io.Writer, timing *client.TimingBreakdown) {
	fmt.Fprintln(w, "Request Timeline:")

	totalMs := float64(timing.Total.Milliseconds())
	if totalMs == 0 {
		return
	}

	// Calculate width for each phase (max 60 chars wide)
	maxWidth := 60

	// Define colors for each phase (using 256-color ANSI)
	dnsColor := color.New(color.FgMagenta)
	tcpColor := color.New(color.FgYellow)
	tlsColor := color.New(color.FgCyan)
	serverColor := color.New(color.FgGreen)
	contentColor := color.New(color.FgBlue)

	// Calculate bar widths
	dnsWidth := int((float64(timing.DNSLookup.Milliseconds()) / totalMs) * float64(maxWidth))
	tcpWidth := int((float64(timing.TCPConnection.Milliseconds()) / totalMs) * float64(maxWidth))
	tlsWidth := int((float64(timing.TLSHandshake.Milliseconds()) / totalMs) * float64(maxWidth))
	serverWidth := int((float64(timing.ServerProcessing.Milliseconds()) / totalMs) * float64(maxWidth))
	contentWidth := int((float64(timing.ContentTransfer.Milliseconds()) / totalMs) * float64(maxWidth))

	// Ensure at least 1 char width for non-zero values
	if timing.DNSLookup > 0 && dnsWidth == 0 {
		dnsWidth = 1
	}
	if timing.TCPConnection > 0 && tcpWidth == 0 {
		tcpWidth = 1
	}
	if timing.TLSHandshake > 0 && tlsWidth == 0 {
		tlsWidth = 1
	}
	if timing.ServerProcessing > 0 && serverWidth == 0 {
		serverWidth = 1
	}
	if timing.ContentTransfer > 0 && contentWidth == 0 {
		contentWidth = 1
	}

	// Draw the waterfall bar
	fmt.Fprint(w, "  ")
	if dnsWidth > 0 {
		dnsColor.Fprint(w, strings.Repeat("█", dnsWidth))
	}
	if tcpWidth > 0 {
		tcpColor.Fprint(w, strings.Repeat("█", tcpWidth))
	}
	if tlsWidth > 0 {
		tlsColor.Fprint(w, strings.Repeat("█", tlsWidth))
	}
	if serverWidth > 0 {
		serverColor.Fprint(w, strings.Repeat("█", serverWidth))
	}
	if contentWidth > 0 {
		contentColor.Fprint(w, strings.Repeat("█", contentWidth))
	}
	fmt.Fprintf(w, " %s\n", formatTimeDuration(time.Duration(timing.Total)))

	// Draw legend
	fmt.Fprintln(w)
	fmt.Fprint(w, "  ")
	if timing.DNSLookup > 0 {
		dnsColor.Fprint(w, "■")
		fmt.Fprintf(w, " DNS (%s)  ", formatTimeDuration(time.Duration(timing.DNSLookup)))
	}
	if timing.TCPConnection > 0 {
		tcpColor.Fprint(w, "■")
		fmt.Fprintf(w, " TCP (%s)  ", formatTimeDuration(time.Duration(timing.TCPConnection)))
	}
	if timing.TLSHandshake > 0 {
		tlsColor.Fprint(w, "■")
		fmt.Fprintf(w, " TLS (%s)  ", formatTimeDuration(time.Duration(timing.TLSHandshake)))
	}
	if timing.ServerProcessing > 0 {
		serverColor.Fprint(w, "■")
		fmt.Fprintf(w, " Server (%s)  ", formatTimeDuration(time.Duration(timing.ServerProcessing)))
	}
	if timing.ContentTransfer > 0 {
		contentColor.Fprint(w, "■")
		fmt.Fprintf(w, " Content (%s)", formatTimeDuration(time.Duration(timing.ContentTransfer)))
	}
	fmt.Fprintln(w)
}
