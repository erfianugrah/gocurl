package output

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
	"github.com/fatih/color"
)

// GraphFormatter formats output with ASCII graphs
type GraphFormatter struct {
	verbose bool
}

// NewGraphFormatter creates a new graph formatter
func NewGraphFormatter(verbose bool) *GraphFormatter {
	return &GraphFormatter{verbose: verbose}
}

// Format formats timing with graphs
func (f *GraphFormatter) Format(timing *client.TimingBreakdown) (string, error) {
	return "", fmt.Errorf("graph format not supported for single requests")
}

// Write writes timing with graphs
func (f *GraphFormatter) Write(w io.Writer, timing *client.TimingBreakdown) error {
	return fmt.Errorf("graph format not supported for single requests")
}

// FormatMultiple formats multiple results with graphs
func (f *GraphFormatter) FormatMultiple(stats *metrics.Stats) (string, error) {
	var buf strings.Builder
	if err := f.WriteMultiple(&buf, stats); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteMultiple writes multiple results with graphs
func (f *GraphFormatter) WriteMultiple(w io.Writer, stats *metrics.Stats) error {
	// Summary header
	fmt.Fprintf(w, "%s\n", color.CyanString("=== Load Test Results ==="))
	fmt.Fprintf(w, "Total Requests: %d\n", stats.TotalRequests)
	fmt.Fprintf(w, "Successful: %s\n", color.GreenString("%d", stats.SuccessfulRequests))
	fmt.Fprintf(w, "Failed: %s\n", color.RedString("%d", stats.FailedRequests))
	fmt.Fprintf(w, "Duration: %s\n", formatDuration(stats.Duration))
	fmt.Fprintf(w, "Requests/sec: %.2f\n\n", stats.RequestsPerSecond)

	// Latency statistics
	fmt.Fprintf(w, "%s\n", color.YellowString("Latency Statistics:"))
	fmt.Fprintf(w, "  Min:         %s\n", formatDuration(stats.MinLatency))
	fmt.Fprintf(w, "  Max:         %s\n", formatDuration(stats.MaxLatency))
	fmt.Fprintf(w, "  Mean:        %s\n", formatDuration(stats.MeanLatency))
	fmt.Fprintf(w, "  Median (p50): %s\n", formatDuration(stats.P50))
	fmt.Fprintf(w, "  P95:         %s\n", formatDuration(stats.P95))
	fmt.Fprintf(w, "  P99:         %s\n", formatDuration(stats.P99))

	if stats.P999 > 0 {
		fmt.Fprintf(w, "  P99.9:       %s\n", formatDuration(stats.P999))
	}
	if stats.P9999 > 0 {
		fmt.Fprintf(w, "  P99.99:      %s\n", formatDuration(stats.P9999))
	}
	fmt.Fprintln(w)

	// Latency distribution histogram
	if stats.Histogram != nil && len(stats.Histogram) > 0 {
		fmt.Fprintf(w, "%s\n", color.YellowString("Latency Distribution:"))
		f.drawHistogram(w, stats.Histogram, stats.TotalRequests)
		fmt.Fprintln(w)
	}

	// Status code distribution
	if len(stats.StatusCodes) > 0 {
		fmt.Fprintf(w, "%s\n", color.YellowString("Status Code Distribution:"))
		for code, count := range stats.StatusCodes {
			pct := (float64(count) / float64(stats.TotalRequests)) * 100
			bar := f.createBar(int(pct), 50)
			statusColor := getStatusColor(code)
			fmt.Fprintf(w, "  %s %s %s (%.1f%%)\n",
				statusColor("%3d", code),
				bar,
				fmt.Sprintf("%5d", count),
				pct)
		}
		fmt.Fprintln(w)
	}

	return nil
}

// drawHistogram draws an ASCII histogram
func (f *GraphFormatter) drawHistogram(w io.Writer, histogram map[int]int, total int) {
	if len(histogram) == 0 {
		return
	}

	// Find the range and max count
	var minBucket, maxBucket, maxCount int
	first := true
	for bucket, count := range histogram {
		if first {
			minBucket = bucket
			maxBucket = bucket
			first = false
		}
		if bucket < minBucket {
			minBucket = bucket
		}
		if bucket > maxBucket {
			maxBucket = bucket
		}
		if count > maxCount {
			maxCount = count
		}
	}

	// Draw the histogram
	maxBarWidth := 50
	for bucket := minBucket; bucket <= maxBucket; bucket++ {
		count := histogram[bucket]
		if count == 0 {
			continue
		}

		// Calculate bar width
		barWidth := int(float64(count) / float64(maxCount) * float64(maxBarWidth))
		bar := strings.Repeat("█", barWidth)

		// Calculate percentage
		pct := float64(count) / float64(total) * 100

		// Format the bucket range
		rangeStr := f.formatBucketRange(bucket)

		fmt.Fprintf(w, "  %12s │%s %d (%.1f%%)\n", rangeStr, bar, count, pct)
	}
}

// formatBucketRange formats a histogram bucket as a time range
func (f *GraphFormatter) formatBucketRange(bucket int) string {
	// Each bucket represents a range of milliseconds
	// Bucket 0: 0-10ms, Bucket 1: 10-20ms, etc.
	start := bucket * 10
	end := start + 10

	if start < 1000 {
		return fmt.Sprintf("%d-%dms", start, end)
	}
	return fmt.Sprintf("%.1f-%.1fs", float64(start)/1000, float64(end)/1000)
}

// createBar creates a horizontal bar for visualization
func (f *GraphFormatter) createBar(value, maxWidth int) string {
	if value <= 0 {
		return ""
	}
	if value > maxWidth {
		value = maxWidth
	}
	return color.GreenString(strings.Repeat("█", value))
}

// drawLatencyGraph creates a simple line graph of latencies
func (f *GraphFormatter) drawLatencyGraph(w io.Writer, latencies []time.Duration) {
	if len(latencies) == 0 {
		return
	}

	// Sample data if too many points
	sampleSize := 50
	sampled := latencies
	if len(latencies) > sampleSize {
		step := len(latencies) / sampleSize
		sampled = make([]time.Duration, 0, sampleSize)
		for i := 0; i < len(latencies); i += step {
			sampled = append(sampled, latencies[i])
		}
	}

	// Convert to float64 milliseconds
	data := make([]float64, len(sampled))
	for i, d := range sampled {
		data[i] = float64(d.Milliseconds())
	}

	// Find min and max
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Simple ASCII plot (height = 10 lines)
	height := 10
	width := len(data)

	// Normalize data to height
	normalized := make([]int, len(data))
	for i, v := range data {
		if max > min {
			normalized[i] = int(float64(height-1) * (v - min) / (max - min))
		}
	}

	// Draw the graph
	for row := height - 1; row >= 0; row-- {
		line := ""
		for col := 0; col < width; col++ {
			if normalized[col] == row {
				line += "•"
			} else if normalized[col] > row {
				line += "│"
			} else {
				line += " "
			}
		}

		// Add y-axis label
		value := min + (max-min)*float64(row)/float64(height-1)
		fmt.Fprintf(w, "%8.1fms │%s\n", value, line)
	}

	// X-axis
	fmt.Fprintf(w, "          └%s\n", strings.Repeat("─", width))
	fmt.Fprintf(w, "           %s → %s\n", "Request 1", fmt.Sprintf("Request %d", len(latencies)))
}

// createHistogramBuckets creates histogram buckets from latency data
func createHistogramBuckets(latencies []time.Duration) map[int]int {
	histogram := make(map[int]int)

	for _, latency := range latencies {
		ms := latency.Milliseconds()
		// Use logarithmic buckets for better distribution
		bucket := int(math.Log10(float64(ms+1)) * 10)
		histogram[bucket]++
	}

	return histogram
}
