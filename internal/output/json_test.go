package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

func TestNewJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter(false)

	if formatter == nil {
		t.Fatal("NewJSONFormatter returned nil")
	}

	if formatter.verbose {
		t.Error("verbose should be false")
	}
}

func TestJSONFormatterFormat(t *testing.T) {
	formatter := NewJSONFormatter(false)

	timing := &client.TimingBreakdown{
		DNSLookup: client.Duration(10 * time.Millisecond),
		TCPConnection: client.Duration(20 * time.Millisecond),
		TLSHandshake: client.Duration(30 * time.Millisecond),
		ServerProcessing: client.Duration(40 * time.Millisecond),
		ContentTransfer: client.Duration(50 * time.Millisecond),
		Total: client.Duration(150 * time.Millisecond),
		StatusCode:       200,
		ContentLength:    1024,
		ResponseSize:     1024,
	}

	output, err := formatter.Format(timing)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Parse the JSON to verify it's valid
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify key fields
	if parsed["status_code"].(float64) != 200 {
		t.Error("status_code not correctly formatted")
	}

	if parsed["response_size"].(float64) != 1024 {
		t.Error("response_size not correctly formatted")
	}
}

func TestJSONFormatterWrite(t *testing.T) {
	formatter := NewJSONFormatter(false)

	timing := &client.TimingBreakdown{
		Total: client.Duration(100 * time.Millisecond),
		StatusCode: 200,
	}

	var buf bytes.Buffer
	err := formatter.Write(&buf, timing)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Output should not be empty")
	}

	// Verify JSON is valid
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}
}

func TestJSONFormatterFormatMultiple(t *testing.T) {
	formatter := NewJSONFormatter(false)

	stats := &metrics.Stats{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		Duration:           metrics.Duration(10 * time.Second),
		RequestsPerSecond:  10.0,
		MinLatency:         metrics.Duration(50 * time.Millisecond),
		MaxLatency:         metrics.Duration(500 * time.Millisecond),
		MeanLatency:        metrics.Duration(150 * time.Millisecond),
		P50:                metrics.Duration(120 * time.Millisecond),
		P95:                metrics.Duration(400 * time.Millisecond),
		P99:                metrics.Duration(480 * time.Millisecond),
		StatusCodes: map[int]int{
			200: 95,
			500: 5,
		},
		ErrorRate:      0.05,
		TotalBytes:     1024000,
		BytesPerSecond: 102400.0,
	}

	output, err := formatter.FormatMultiple(stats)
	if err != nil {
		t.Fatalf("FormatMultiple failed: %v", err)
	}

	// Parse the JSON to verify it's valid
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify key fields
	if parsed["total_requests"].(float64) != 100 {
		t.Error("total_requests not correctly formatted")
	}

	if parsed["requests_per_second"].(float64) != 10.0 {
		t.Error("requests_per_second not correctly formatted")
	}
}

func TestJSONFormatterWriteMultiple(t *testing.T) {
	formatter := NewJSONFormatter(false)

	stats := &metrics.Stats{
		TotalRequests:      10,
		SuccessfulRequests: 10,
		RequestsPerSecond:  5.0,
	}

	var buf bytes.Buffer
	err := formatter.WriteMultiple(&buf, stats)
	if err != nil {
		t.Fatalf("WriteMultiple failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Output should not be empty")
	}

	// Verify JSON is valid
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}
}
