package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

func TestNewTableFormatter(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{"normal mode", false},
		{"verbose mode", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTableFormatter(tt.verbose)
			if formatter == nil {
				t.Error("NewTableFormatter returned nil")
			}

			if formatter.verbose != tt.verbose {
				t.Errorf("verbose = %v, want %v", formatter.verbose, tt.verbose)
			}
		})
	}
}

func TestTableFormatterWrite(t *testing.T) {
	tests := []struct {
		name      string
		timing    *client.TimingBreakdown
		verbose   bool
		wantError bool
		contains  []string
	}{
		{
			name: "successful request",
			timing: &client.TimingBreakdown{
				URL:              "https://example.com",
				DNSLookup:        client.Duration(10000000),  // 10ms in nanoseconds
				TCPConnection:    client.Duration(20000000),  // 20ms
				TLSHandshake:     client.Duration(30000000),  // 30ms
				ServerProcessing: client.Duration(40000000),  // 40ms
				ContentTransfer:  client.Duration(5000000),   // 5ms
				Total:            client.Duration(105000000), // 105ms
				StatusCode:       200,
				ResponseSize:     1024,
			},
			verbose:   false,
			wantError: false,
			contains: []string{
				"200",   // status code
				"105ms", // total time
				"10ms",  // DNS
				"20ms",  // TCP
				"30ms",  // TLS
				"40ms",  // Server
			},
		},
		{
			name: "request with error",
			timing: &client.TimingBreakdown{
				URL:        "https://example.com",
				StatusCode: 500,
				Total:      client.Duration(50000000),
				Error:      "connection timeout",
			},
			verbose:   false,
			wantError: false,
			contains: []string{
				"500",
			},
		},
		{
			name: "verbose output",
			timing: &client.TimingBreakdown{
				URL:              "https://example.com",
				StatusCode:       200,
				ResponseSize:     2048,
				TLSVersion:       "TLS 1.3",
				TLSCipherSuite:   "TLS_AES_128_GCM_SHA256",
				ConnectionReused: true,
				Total:            client.Duration(100000000),
			},
			verbose:   true,
			wantError: false,
			contains: []string{
				"TLS 1.3",
				"TLS_AES_128_GCM_SHA256",
				"2.0 KiB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTableFormatter(tt.verbose)
			var buf bytes.Buffer

			err := formatter.Write(&buf, tt.timing)

			if (err != nil) != tt.wantError {
				t.Errorf("Write() error = %v, wantError %v", err, tt.wantError)
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

func TestTableFormatterWriteMultiple(t *testing.T) {
	tests := []struct {
		name      string
		stats     *metrics.Stats
		wantError bool
		contains  []string
	}{
		{
			name: "load test results",
			stats: &metrics.Stats{
				TotalRequests:      100,
				SuccessfulRequests: 98,
				FailedRequests:     2,
				Duration:           metrics.Duration(5000000000), // 5s
				RequestsPerSecond:  19.6,
				MinLatency:         metrics.Duration(50000000),  // 50ms
				MaxLatency:         metrics.Duration(500000000), // 500ms
				MeanLatency:        metrics.Duration(120000000), // 120ms
				P50:                metrics.Duration(100000000), // 100ms
				P90:                metrics.Duration(200000000), // 200ms
				P95:                metrics.Duration(300000000), // 300ms
				P99:                metrics.Duration(450000000), // 450ms
			},
			wantError: false,
			contains: []string{
				"100",   // total requests
				"98",    // successful
				"2",     // failed
				"50ms",  // min
				"500ms", // max
				"100ms", // p50 (Median)
				"300ms", // p95
				"450ms", // p99
			},
		},
		{
			name: "all requests successful",
			stats: &metrics.Stats{
				TotalRequests:      50,
				SuccessfulRequests: 50,
				FailedRequests:     0,
				Duration:           metrics.Duration(2500000000), // 2.5s
				RequestsPerSecond:  20.0,
			},
			wantError: false,
			contains: []string{
				"50",
				"20.00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTableFormatter(false)
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

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		code     int
		wantFunc string // name of expected color function
	}{
		{200, "green"},
		{201, "green"},
		{301, "yellow"},
		{302, "yellow"},
		{400, "red"},
		{404, "red"},
		{500, "red"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.code)), func(t *testing.T) {
			colorFunc := getStatusColor(tt.code)
			if colorFunc == nil {
				t.Error("getStatusColor returned nil")
			}
			// Just verify it returns a function that works
			result := colorFunc("test")
			if result == "" {
				t.Error("color function returned empty string")
			}
		})
	}
}

func TestGetStatusText(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{304, "Not Modified"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
		{999, ""}, // unknown code
	}

	for _, tt := range tests {
		got := getStatusText(tt.code)
		if got != tt.want {
			t.Errorf("getStatusText(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration metrics.Duration
		want     string
	}{
		{metrics.Duration(500000), "0.50ms"},    // 0.5ms
		{metrics.Duration(10000000), "10ms"},    // 10ms
		{metrics.Duration(100000000), "100ms"},  // 100ms
		{metrics.Duration(999000000), "999ms"},  // 999ms
		{metrics.Duration(1000000000), "1.00s"}, // 1s
		{metrics.Duration(1500000000), "1.50s"}, // 1.5s
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{100, "100 B"},
		{1024, "1.0 KiB"},
		{2048, "2.0 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
