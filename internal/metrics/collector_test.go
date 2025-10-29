package metrics

import (
	"testing"
	"time"

	"github.com/erfi/gocurl/internal/client"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector()

	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if collector.timings == nil {
		t.Error("Timings slice should be initialized")
	}

	if collector.startTime.IsZero() {
		t.Error("Start time should be set")
	}
}

func TestCollectorRecord(t *testing.T) {
	collector := NewCollector()

	timing := &client.TimingBreakdown{
		Total: client.Duration(100 * time.Millisecond),
		StatusCode: 200,
	}

	collector.Record(timing)

	if len(collector.timings) != 1 {
		t.Errorf("Expected 1 timing, got %d", len(collector.timings))
	}

	if collector.timings[0] != timing {
		t.Error("Recorded timing does not match")
	}
}

func TestCollectorFinalize(t *testing.T) {
	collector := NewCollector()

	if !collector.endTime.IsZero() {
		t.Error("End time should not be set initially")
	}

	collector.Finalize()

	if collector.endTime.IsZero() {
		t.Error("End time should be set after finalize")
	}
}

func TestCollectorCalculateEmpty(t *testing.T) {
	collector := NewCollector()
	collector.Finalize()

	stats := collector.Calculate()

	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", stats.TotalRequests)
	}
}

func TestCollectorCalculateSingle(t *testing.T) {
	collector := NewCollector()

	timing := &client.TimingBreakdown{
		Total: client.Duration(100 * time.Millisecond),
		StatusCode:   200,
		ResponseSize: 1024,
	}

	collector.Record(timing)
	collector.Finalize()

	stats := collector.Calculate()

	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", stats.TotalRequests)
	}

	if stats.SuccessfulRequests != 1 {
		t.Errorf("Expected 1 successful request, got %d", stats.SuccessfulRequests)
	}

	if stats.FailedRequests != 0 {
		t.Errorf("Expected 0 failed requests, got %d", stats.FailedRequests)
	}

	if time.Duration(stats.MinLatency) != 100*time.Millisecond {
		t.Errorf("Expected min latency 100ms, got %v", stats.MinLatency)
	}

	if time.Duration(stats.MaxLatency) != 100*time.Millisecond {
		t.Errorf("Expected max latency 100ms, got %v", stats.MaxLatency)
	}

	if stats.TotalBytes != 1024 {
		t.Errorf("Expected 1024 total bytes, got %d", stats.TotalBytes)
	}
}

func TestCollectorCalculateMultiple(t *testing.T) {
	collector := NewCollector()

	timings := []*client.TimingBreakdown{
		{Total: client.Duration(50 * time.Millisecond), StatusCode: 200, ResponseSize: 512},
		{Total: client.Duration(100 * time.Millisecond), StatusCode: 200, ResponseSize: 1024},
		{Total: client.Duration(150 * time.Millisecond), StatusCode: 200, ResponseSize: 2048},
		{Total: client.Duration(200 * time.Millisecond), StatusCode: 201, ResponseSize: 256},
		{Total: client.Duration(250 * time.Millisecond), StatusCode: 500, ResponseSize: 128},
	}

	for _, timing := range timings {
		collector.Record(timing)
	}

	collector.Finalize()
	stats := collector.Calculate()

	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 total requests, got %d", stats.TotalRequests)
	}

	if stats.SuccessfulRequests != 5 {
		t.Errorf("Expected 5 successful requests, got %d", stats.SuccessfulRequests)
	}

	if time.Duration(stats.MinLatency) != 50*time.Millisecond {
		t.Errorf("Expected min latency 50ms, got %v", stats.MinLatency)
	}

	if time.Duration(stats.MaxLatency) != 250*time.Millisecond {
		t.Errorf("Expected max latency 250ms, got %v", stats.MaxLatency)
	}

	// Mean should be (50+100+150+200+250)/5 = 150ms
	if time.Duration(stats.MeanLatency) != 150*time.Millisecond {
		t.Errorf("Expected mean latency 150ms, got %v", stats.MeanLatency)
	}

	// Median (p50) should be 150ms
	if time.Duration(stats.P50) != 150*time.Millisecond {
		t.Errorf("Expected p50 150ms, got %v", stats.P50)
	}

	// Check status codes
	if stats.StatusCodes[200] != 3 {
		t.Errorf("Expected 3 requests with status 200, got %d", stats.StatusCodes[200])
	}

	if stats.StatusCodes[201] != 1 {
		t.Errorf("Expected 1 request with status 201, got %d", stats.StatusCodes[201])
	}

	if stats.StatusCodes[500] != 1 {
		t.Errorf("Expected 1 request with status 500, got %d", stats.StatusCodes[500])
	}
}

func TestCollectorCalculateWithErrors(t *testing.T) {
	collector := NewCollector()

	timings := []*client.TimingBreakdown{
		{Total: client.Duration(100 * time.Millisecond), StatusCode: 200},
		{Total: client.Duration(150 * time.Millisecond), StatusCode: 0, Error: "connection refused"},
		{Total: client.Duration(200 * time.Millisecond), StatusCode: 200},
	}

	for _, timing := range timings {
		collector.Record(timing)
	}

	collector.Finalize()
	stats := collector.Calculate()

	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}

	if stats.SuccessfulRequests != 2 {
		t.Errorf("Expected 2 successful requests, got %d", stats.SuccessfulRequests)
	}

	if stats.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", stats.FailedRequests)
	}

	expectedErrorRate := 1.0 / 3.0
	if stats.ErrorRate != expectedErrorRate {
		t.Errorf("Expected error rate %.3f, got %.3f", expectedErrorRate, stats.ErrorRate)
	}
}

func TestPercentile(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	tests := []struct {
		percentile float64
		expected   time.Duration
		tolerance  time.Duration
	}{
		{0, 10 * time.Millisecond, 1 * time.Millisecond},
		{50, 55 * time.Millisecond, 5 * time.Millisecond},
		{90, 91 * time.Millisecond, 5 * time.Millisecond},
		{100, 100 * time.Millisecond, 1 * time.Millisecond},
	}

	for _, tt := range tests {
		result := percentile(durations, tt.percentile)
		diff := result - tt.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tt.tolerance {
			t.Errorf("percentile(%.0f): expected ~%v, got %v (diff: %v)",
				tt.percentile, tt.expected, result, diff)
		}
	}
}

func TestPercentileEmpty(t *testing.T) {
	durations := []time.Duration{}
	result := percentile(durations, 50)

	if result != 0 {
		t.Errorf("Expected 0 for empty slice, got %v", result)
	}
}

func TestPercentileSingle(t *testing.T) {
	durations := []time.Duration{100 * time.Millisecond}

	tests := []float64{0, 25, 50, 75, 100}
	for _, p := range tests {
		result := percentile(durations, p)
		if result != 100*time.Millisecond {
			t.Errorf("percentile(%.0f) for single element: expected 100ms, got %v", p, result)
		}
	}
}

func TestCollectorReset(t *testing.T) {
	collector := NewCollector()

	timing := &client.TimingBreakdown{
		Total: client.Duration(100 * time.Millisecond),
		StatusCode: 200,
	}

	collector.Record(timing)

	if len(collector.timings) != 1 {
		t.Errorf("Expected 1 timing before reset, got %d", len(collector.timings))
	}

	collector.Reset()

	if len(collector.timings) != 0 {
		t.Errorf("Expected 0 timings after reset, got %d", len(collector.timings))
	}
}

func TestCollectorHistogram(t *testing.T) {
	collector := NewCollector()

	timings := []*client.TimingBreakdown{
		{Total: client.Duration(5 * time.Millisecond), StatusCode: 200},   // bucket 0
		{Total: client.Duration(15 * time.Millisecond), StatusCode: 200},  // bucket 1
		{Total: client.Duration(25 * time.Millisecond), StatusCode: 200},  // bucket 2
		{Total: client.Duration(95 * time.Millisecond), StatusCode: 200},  // bucket 9
		{Total: client.Duration(105 * time.Millisecond), StatusCode: 200}, // bucket 10
	}

	for _, timing := range timings {
		collector.Record(timing)
	}

	collector.Finalize()
	stats := collector.Calculate()

	if stats.Histogram == nil {
		t.Fatal("Histogram should not be nil")
	}

	// Check specific buckets
	if stats.Histogram[0] != 1 {
		t.Errorf("Expected 1 request in bucket 0, got %d", stats.Histogram[0])
	}

	if stats.Histogram[1] != 1 {
		t.Errorf("Expected 1 request in bucket 1, got %d", stats.Histogram[1])
	}

	if stats.Histogram[2] != 1 {
		t.Errorf("Expected 1 request in bucket 2, got %d", stats.Histogram[2])
	}

	if stats.Histogram[9] != 1 {
		t.Errorf("Expected 1 request in bucket 9, got %d", stats.Histogram[9])
	}

	if stats.Histogram[10] != 1 {
		t.Errorf("Expected 1 request in bucket 10, got %d", stats.Histogram[10])
	}
}

func TestCollectorExtendedPercentiles(t *testing.T) {
	// Test with 1000 requests for p99.9
	collector := NewCollector()

	for i := 0; i < 1000; i++ {
		timing := &client.TimingBreakdown{
			Total:      client.Duration(time.Duration(i+1) * time.Millisecond),
			StatusCode: 200,
		}
		collector.Record(timing)
	}

	collector.Finalize()
	stats := collector.Calculate()

	if time.Duration(stats.P999) == 0 {
		t.Error("P99.9 should be calculated for 1000+ requests")
	}

	// P99.9 should be around 999ms
	if time.Duration(stats.P999) < 990*time.Millisecond || time.Duration(stats.P999) > 1000*time.Millisecond {
		t.Errorf("P99.9 out of expected range: %v", stats.P999)
	}
}

func TestCollectorThroughput(t *testing.T) {
	collector := NewCollector()

	// Add some requests
	for i := 0; i < 10; i++ {
		timing := &client.TimingBreakdown{
			Total: client.Duration(100 * time.Millisecond),
			StatusCode:   200,
			ResponseSize: 1024,
		}
		collector.Record(timing)
	}

	// Wait a bit to have measurable duration
	time.Sleep(100 * time.Millisecond)
	collector.Finalize()

	stats := collector.Calculate()

	if stats.RequestsPerSecond <= 0 {
		t.Error("Requests per second should be positive")
	}

	if stats.BytesPerSecond <= 0 {
		t.Error("Bytes per second should be positive")
	}

	if stats.TotalBytes != 10240 {
		t.Errorf("Expected 10240 total bytes, got %d", stats.TotalBytes)
	}
}
