package metrics

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDurationMarshalJSON(t *testing.T) {
	d := Duration(150 * time.Millisecond)

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Failed to marshal duration: %v", err)
	}

	expected := "150"
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestDurationMilliseconds(t *testing.T) {
	d := Duration(1500 * time.Millisecond)

	ms := d.Milliseconds()
	if ms != 1500 {
		t.Errorf("Expected 1500ms, got %d", ms)
	}
}

func TestDurationSeconds(t *testing.T) {
	d := Duration(1500 * time.Millisecond)

	seconds := d.Seconds()
	if seconds != 1.5 {
		t.Errorf("Expected 1.5s, got %f", seconds)
	}
}

func TestDurationString(t *testing.T) {
	d := Duration(1500 * time.Millisecond)

	str := d.String()
	expected := "1.5s"
	if str != expected {
		t.Errorf("Expected %s, got %s", expected, str)
	}
}

func TestStatsMarshalJSON(t *testing.T) {
	stats := &Stats{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		Duration:           Duration(10 * time.Second),
		RequestsPerSecond:  10.0,
		MinLatency:         Duration(50 * time.Millisecond),
		MaxLatency:         Duration(500 * time.Millisecond),
		MeanLatency:        Duration(150 * time.Millisecond),
		P50:                Duration(120 * time.Millisecond),
		P90:                Duration(300 * time.Millisecond),
		P95:                Duration(400 * time.Millisecond),
		P99:                Duration(480 * time.Millisecond),
		StatusCodes: map[int]int{
			200: 95,
			500: 5,
		},
		ErrorRate:      0.05,
		TotalBytes:     1024000,
		BytesPerSecond: 102400.0,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	// Unmarshal to verify structure
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stats: %v", err)
	}

	// Verify key fields
	if unmarshaled["total_requests"].(float64) != 100 {
		t.Error("total_requests not correctly marshaled")
	}

	if unmarshaled["duration"].(float64) != 10000 {
		t.Error("duration not correctly marshaled as milliseconds")
	}

	if unmarshaled["requests_per_second"].(float64) != 10.0 {
		t.Error("requests_per_second not correctly marshaled")
	}
}

func TestStatsWithExtendedPercentiles(t *testing.T) {
	stats := &Stats{
		TotalRequests: 1000,
		P99:           Duration(480 * time.Millisecond),
		P999:          Duration(495 * time.Millisecond),
		P9999:         Duration(499 * time.Millisecond),
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	// Verify the JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Check that values are marshaled as milliseconds (integers)
	if jsonMap["p99"].(float64) != 480 {
		t.Errorf("P99 not correctly marshaled as milliseconds: %v", jsonMap["p99"])
	}

	if jsonMap["p99_9"].(float64) != 495 {
		t.Errorf("P999 not correctly marshaled as milliseconds: %v", jsonMap["p99_9"])
	}

	if jsonMap["p99_99"].(float64) != 499 {
		t.Errorf("P9999 not correctly marshaled as milliseconds: %v", jsonMap["p99_99"])
	}
}

func TestStatsWithHistogram(t *testing.T) {
	stats := &Stats{
		TotalRequests: 100,
		Histogram: map[int]int{
			0:  10,
			1:  20,
			2:  30,
			5:  25,
			10: 15,
		},
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	var unmarshaled Stats
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stats: %v", err)
	}

	if len(unmarshaled.Histogram) != 5 {
		t.Errorf("Expected 5 histogram buckets, got %d", len(unmarshaled.Histogram))
	}

	if unmarshaled.Histogram[0] != 10 {
		t.Errorf("Expected 10 in bucket 0, got %d", unmarshaled.Histogram[0])
	}
}
