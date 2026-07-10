package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

func TestNewGraphFormatter(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{"normal mode", false},
		{"verbose mode", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGraphFormatter(tt.verbose)
			if formatter == nil {
				t.Error("NewGraphFormatter returned nil")
			}

			if formatter.verbose != tt.verbose {
				t.Errorf("verbose = %v, want %v", formatter.verbose, tt.verbose)
			}
		})
	}
}

func TestGraphFormatterFormat(t *testing.T) {
	formatter := NewGraphFormatter(false)
	timing := &client.TimingBreakdown{
		URL:        "https://example.com",
		StatusCode: 200,
		Total:      client.Duration(100000000),
	}

	result, err := formatter.Format(timing)
	if err == nil {
		t.Error("Format() should return error for single requests")
	}
	if result != "" {
		t.Errorf("Format() should return empty string, got %q", result)
	}
}

func TestGraphFormatterWrite(t *testing.T) {
	formatter := NewGraphFormatter(false)
	var buf bytes.Buffer
	timing := &client.TimingBreakdown{
		URL:        "https://example.com",
		StatusCode: 200,
		Total:      client.Duration(100000000),
	}

	err := formatter.Write(&buf, timing)
	if err == nil {
		t.Error("Write() should return error for single requests")
	}
}

func TestGraphFormatterWriteMultiple(t *testing.T) {
	tests := []struct {
		name      string
		stats     *metrics.Stats
		wantError bool
		contains  []string
	}{
		{
			name: "load test with histogram",
			stats: &metrics.Stats{
				TotalRequests:      100,
				SuccessfulRequests: 95,
				FailedRequests:     5,
				Duration:           metrics.Duration(10000000000), // 10s
				RequestsPerSecond:  10.0,
				MinLatency:         metrics.Duration(50000000),  // 50ms
				MaxLatency:         metrics.Duration(500000000), // 500ms
				MeanLatency:        metrics.Duration(120000000), // 120ms
				P50:                metrics.Duration(100000000), // 100ms
				P95:                metrics.Duration(300000000), // 300ms
				P99:                metrics.Duration(450000000), // 450ms
				Histogram: map[int]int{
					1: 10,
					2: 30,
					3: 40,
					4: 15,
					5: 5,
				},
				StatusCodes: map[int]int{
					200: 95,
					500: 5,
				},
			},
			wantError: false,
			contains: []string{
				"Load Test Results",
				"100",   // total requests
				"95",    // successful
				"5",     // failed
				"10.00", // rps
				"50ms",  // min
				"500ms", // max
				"Latency Distribution",
				"Status Code Distribution",
			},
		},
		{
			name: "load test without histogram",
			stats: &metrics.Stats{
				TotalRequests:      50,
				SuccessfulRequests: 50,
				FailedRequests:     0,
				Duration:           metrics.Duration(5000000000), // 5s
				RequestsPerSecond:  10.0,
				MinLatency:         metrics.Duration(100000000), // 100ms
				MaxLatency:         metrics.Duration(200000000), // 200ms
				MeanLatency:        metrics.Duration(150000000), // 150ms
				P50:                metrics.Duration(150000000), // 150ms
				P95:                metrics.Duration(190000000), // 190ms
				P99:                metrics.Duration(195000000), // 195ms
			},
			wantError: false,
			contains: []string{
				"Load Test Results",
				"50",
				"10.00",
			},
		},
		{
			name: "load test with p999 and p9999",
			stats: &metrics.Stats{
				TotalRequests:      1000,
				SuccessfulRequests: 1000,
				FailedRequests:     0,
				Duration:           metrics.Duration(100000000000), // 100s
				RequestsPerSecond:  10.0,
				MinLatency:         metrics.Duration(50000000),   // 50ms
				MaxLatency:         metrics.Duration(1000000000), // 1s
				MeanLatency:        metrics.Duration(120000000),  // 120ms
				P50:                metrics.Duration(100000000),  // 100ms
				P95:                metrics.Duration(300000000),  // 300ms
				P99:                metrics.Duration(500000000),  // 500ms
				P999:               metrics.Duration(800000000),  // 800ms
				P9999:              metrics.Duration(950000000),  // 950ms
			},
			wantError: false,
			contains: []string{
				"P99.9",
				"P99.99",
				"800ms",
				"950ms",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGraphFormatter(false)
			var buf bytes.Buffer

			err := formatter.WriteMultiple(&buf, tt.stats)

			if (err != nil) != tt.wantError {
				t.Errorf("WriteMultiple() error = %v, wantError %v", err, tt.wantError)
				return
			}

			output := buf.String()

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestGraphFormatterFormatBucketRange(t *testing.T) {
	formatter := NewGraphFormatter(false)

	tests := []struct {
		bucket int
		want   string
	}{
		{0, "0-10ms"},
		{1, "10-20ms"},
		{5, "50-60ms"},
		{99, "990-1000ms"},
		{100, "1.0-1.0s"},
		{150, "1.5-1.5s"},
	}

	for _, tt := range tests {
		got := formatter.formatBucketRange(tt.bucket)
		if got != tt.want {
			t.Errorf("formatBucketRange(%d) = %q, want %q", tt.bucket, got, tt.want)
		}
	}
}

func TestGraphFormatterCreateBar(t *testing.T) {
	formatter := NewGraphFormatter(false)

	tests := []struct {
		name     string
		value    int
		maxWidth int
		wantLen  int // approximate length (considering color codes)
	}{
		{"empty bar", 0, 50, 0},
		{"partial bar", 25, 50, 25},
		{"full bar", 50, 50, 50},
		{"over max", 75, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatter.createBar(tt.value, tt.maxWidth)
			// For empty bars
			if tt.wantLen == 0 {
				if got != "" {
					t.Errorf("createBar(%d, %d) = %q, want empty string", tt.value, tt.maxWidth, got)
				}
				return
			}
			// For non-empty bars, just check it's not empty
			if got == "" {
				t.Errorf("createBar(%d, %d) returned empty string", tt.value, tt.maxWidth)
			}
		})
	}
}

func TestGraphFormatterDrawHistogram(t *testing.T) {
	formatter := NewGraphFormatter(false)
	var buf bytes.Buffer

	histogram := map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}
	total := 60

	formatter.drawHistogram(&buf, histogram, total)
	output := buf.String()

	// Check that output contains histogram bars
	if output == "" {
		t.Error("drawHistogram produced no output")
	}

	// Check for histogram formatting
	if !strings.Contains(output, "│") {
		t.Error("drawHistogram output missing histogram separator")
	}
}

func TestGraphFormatterDrawHistogramEmpty(t *testing.T) {
	formatter := NewGraphFormatter(false)
	var buf bytes.Buffer

	// Empty histogram
	histogram := map[int]int{}
	formatter.drawHistogram(&buf, histogram, 0)

	output := buf.String()
	if output != "" {
		t.Errorf("drawHistogram with empty histogram should produce no output, got: %q", output)
	}
}
