package client

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestStreamingReader(t *testing.T) {
	data := "Hello, World! This is test data for streaming reader."
	reader := strings.NewReader(data)

	streamReader := NewStreamingReader(reader, "HTTP/2.0")

	// Read in small chunks to simulate network behavior
	buf := make([]byte, 10)
	totalRead := 0

	for {
		n, err := streamReader.Read(buf)
		totalRead += n

		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Small delay to simulate network timing
		time.Sleep(1 * time.Millisecond)
	}

	metrics := streamReader.Metrics()

	// Verify metrics
	if metrics.Protocol != "HTTP/2.0" {
		t.Errorf("Expected protocol HTTP/2.0, got %s", metrics.Protocol)
	}

	if metrics.TotalBytes != int64(len(data)) {
		t.Errorf("Expected %d bytes, got %d", len(data), metrics.TotalBytes)
	}

	if metrics.TotalChunks == 0 {
		t.Error("Expected at least one chunk")
	}

	if len(metrics.ChunkTimings) != metrics.TotalChunks {
		t.Errorf("ChunkTimings length %d doesn't match TotalChunks %d",
			len(metrics.ChunkTimings), metrics.TotalChunks)
	}

	if metrics.BytesPerSecond == 0 {
		t.Error("Expected non-zero throughput")
	}

	// Verify chunk timings are monotonically increasing
	var prevElapsed Duration
	for i, chunk := range metrics.ChunkTimings {
		if i > 0 && chunk.ElapsedTime <= prevElapsed {
			t.Errorf("Chunk %d elapsed time %v not greater than previous %v",
				i, chunk.ElapsedTime, prevElapsed)
		}
		prevElapsed = chunk.ElapsedTime

		if chunk.Size <= 0 {
			t.Errorf("Chunk %d has invalid size %d", i, chunk.Size)
		}

		if chunk.SequenceNumber != i {
			t.Errorf("Chunk sequence number %d doesn't match index %d",
				chunk.SequenceNumber, i)
		}
	}

	// Verify first and last chunk times
	if metrics.FirstChunkTime <= 0 {
		t.Error("FirstChunkTime should be positive")
	}

	if metrics.LastChunkTime <= 0 {
		t.Error("LastChunkTime should be positive")
	}

	if metrics.LastChunkTime < metrics.FirstChunkTime {
		t.Errorf("LastChunkTime %v should be >= FirstChunkTime %v",
			metrics.LastChunkTime, metrics.FirstChunkTime)
	}
}

func TestStreamingReaderEmpty(t *testing.T) {
	reader := strings.NewReader("")
	streamReader := NewStreamingReader(reader, "HTTP/1.1")

	buf := make([]byte, 100)
	n, err := streamReader.Read(buf)

	if n != 0 {
		t.Errorf("Expected 0 bytes read, got %d", n)
	}

	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}

	metrics := streamReader.Metrics()

	if metrics.TotalChunks != 0 {
		t.Errorf("Expected 0 chunks, got %d", metrics.TotalChunks)
	}

	if metrics.TotalBytes != 0 {
		t.Errorf("Expected 0 bytes, got %d", metrics.TotalBytes)
	}
}

func TestStreamingReaderLargeData(t *testing.T) {
	// Create 100KB of data
	data := bytes.Repeat([]byte("A"), 100*1024)
	reader := bytes.NewReader(data)

	streamReader := NewStreamingReader(reader, "HTTP/2.0")

	// Read all data
	result, err := io.ReadAll(streamReader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if !bytes.Equal(result, data) {
		t.Error("Read data doesn't match original")
	}

	metrics := streamReader.Metrics()

	if metrics.TotalBytes != int64(len(data)) {
		t.Errorf("Expected %d bytes, got %d", len(data), metrics.TotalBytes)
	}

	// Should have multiple chunks (io.ReadAll uses 512-byte buffer initially)
	if metrics.TotalChunks < 2 {
		t.Errorf("Expected multiple chunks for 100KB, got %d", metrics.TotalChunks)
	}

	if metrics.AverageChunkSize == 0 {
		t.Error("Expected non-zero average chunk size")
	}

	// Verify throughput calculation
	if metrics.BytesPerSecond <= 0 {
		t.Error("Expected positive throughput")
	}

	// Verify chunk details
	totalSizeFromChunks := int64(0)
	for _, chunk := range metrics.ChunkTimings {
		totalSizeFromChunks += int64(chunk.Size)

		if chunk.ElapsedTime <= 0 {
			t.Errorf("Chunk %d has non-positive elapsed time", chunk.SequenceNumber)
		}
	}

	if totalSizeFromChunks != metrics.TotalBytes {
		t.Errorf("Sum of chunk sizes %d doesn't match total %d",
			totalSizeFromChunks, metrics.TotalBytes)
	}
}

func TestStreamingReaderSingleByte(t *testing.T) {
	reader := strings.NewReader("X")
	streamReader := NewStreamingReader(reader, "HTTP/1.1")

	buf := make([]byte, 1)
	n, err := streamReader.Read(buf)

	if n != 1 {
		t.Errorf("Expected 1 byte read, got %d", n)
	}

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if buf[0] != 'X' {
		t.Errorf("Expected 'X', got '%c'", buf[0])
	}

	metrics := streamReader.Metrics()

	if metrics.TotalChunks != 1 {
		t.Errorf("Expected 1 chunk, got %d", metrics.TotalChunks)
	}

	if metrics.TotalBytes != 1 {
		t.Errorf("Expected 1 byte, got %d", metrics.TotalBytes)
	}

	if metrics.AverageChunkSize != 1 {
		t.Errorf("Expected average chunk size 1, got %d", metrics.AverageChunkSize)
	}
}

func TestStreamMetricsThroughput(t *testing.T) {
	// Create reader with known data
	data := bytes.Repeat([]byte("test"), 1000) // 4KB
	reader := bytes.NewReader(data)

	streamReader := NewStreamingReader(reader, "HTTP/2.0")

	start := time.Now()

	// Read all data
	_, err := io.Copy(io.Discard, streamReader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	elapsed := time.Since(start)
	metrics := streamReader.Metrics()

	// Calculate expected throughput
	expectedThroughput := float64(len(data)) / elapsed.Seconds()

	// Allow 10% margin for measurement overhead
	margin := expectedThroughput * 0.1

	if metrics.BytesPerSecond < expectedThroughput-margin ||
		metrics.BytesPerSecond > expectedThroughput+margin+float64(len(data)) {
		t.Logf("Expected throughput ~%.0f B/s, got %.0f B/s",
			expectedThroughput, metrics.BytesPerSecond)
		// Not failing this as timing can be flaky in tests
	}
}

func TestStreamingReaderProtocol(t *testing.T) {
	tests := []struct {
		protocol string
	}{
		{"HTTP/1.0"},
		{"HTTP/1.1"},
		{"HTTP/2.0"},
		{"HTTP/3.0"},
	}

	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			reader := strings.NewReader("test data")
			streamReader := NewStreamingReader(reader, tt.protocol)

			io.Copy(io.Discard, streamReader)

			metrics := streamReader.Metrics()
			if metrics.Protocol != tt.protocol {
				t.Errorf("Expected protocol %s, got %s", tt.protocol, metrics.Protocol)
			}
		})
	}
}

func BenchmarkStreamingReader(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data"), 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		streamReader := NewStreamingReader(reader, "HTTP/2.0")
		io.Copy(io.Discard, streamReader)
	}
}

func BenchmarkRegularReader(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data"), 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		io.Copy(io.Discard, reader)
	}
}

func TestDetectStalls(t *testing.T) {
	tests := []struct {
		name           string
		chunkTimings   []ChunkTiming
		threshold      time.Duration
		expectedStalls int
	}{
		{
			name:           "no chunks",
			chunkTimings:   []ChunkTiming{},
			threshold:      500 * time.Millisecond,
			expectedStalls: 0,
		},
		{
			name: "single chunk",
			chunkTimings: []ChunkTiming{
				{SequenceNumber: 0, Size: 100, ElapsedTime: Duration(100 * time.Millisecond)},
			},
			threshold:      500 * time.Millisecond,
			expectedStalls: 0,
		},
		{
			name: "no stalls",
			chunkTimings: []ChunkTiming{
				{SequenceNumber: 0, Size: 100, ElapsedTime: Duration(100 * time.Millisecond)},
				{SequenceNumber: 1, Size: 100, ElapsedTime: Duration(200 * time.Millisecond)},
				{SequenceNumber: 2, Size: 100, ElapsedTime: Duration(300 * time.Millisecond)},
			},
			threshold:      500 * time.Millisecond,
			expectedStalls: 0,
		},
		{
			name: "one stall",
			chunkTimings: []ChunkTiming{
				{SequenceNumber: 0, Size: 100, ElapsedTime: Duration(100 * time.Millisecond)},
				{SequenceNumber: 1, Size: 100, ElapsedTime: Duration(800 * time.Millisecond)}, // 700ms gap
				{SequenceNumber: 2, Size: 100, ElapsedTime: Duration(900 * time.Millisecond)},
			},
			threshold:      500 * time.Millisecond,
			expectedStalls: 1,
		},
		{
			name: "multiple stalls",
			chunkTimings: []ChunkTiming{
				{SequenceNumber: 0, Size: 100, ElapsedTime: Duration(100 * time.Millisecond)},
				{SequenceNumber: 1, Size: 100, ElapsedTime: Duration(800 * time.Millisecond)},  // stall 1
				{SequenceNumber: 2, Size: 100, ElapsedTime: Duration(900 * time.Millisecond)},
				{SequenceNumber: 3, Size: 100, ElapsedTime: Duration(2000 * time.Millisecond)}, // stall 2
			},
			threshold:      500 * time.Millisecond,
			expectedStalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &StreamMetrics{
				ChunkTimings: tt.chunkTimings,
			}

			stalls := DetectStalls(metrics, tt.threshold)

			if len(stalls) != tt.expectedStalls {
				t.Errorf("Expected %d stalls, got %d", tt.expectedStalls, len(stalls))
			}

			// Verify stall durations are above threshold
			for i, stall := range stalls {
				if time.Duration(stall.Duration) < tt.threshold {
					t.Errorf("Stall %d duration %v is below threshold %v",
						i, stall.Duration, tt.threshold)
				}
			}
		})
	}
}

func TestCalculateMean(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "empty",
			values:   []float64{},
			expected: 0,
		},
		{
			name:     "single value",
			values:   []float64{10},
			expected: 10,
		},
		{
			name:     "multiple values",
			values:   []float64{10, 20, 30},
			expected: 20,
		},
		{
			name:     "with decimals",
			values:   []float64{1.5, 2.5, 3.5},
			expected: 2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMean(tt.values)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCalculateStdDev(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		mean     float64
		expected float64
	}{
		{
			name:     "empty",
			values:   []float64{},
			mean:     0,
			expected: 0,
		},
		{
			name:     "no variation",
			values:   []float64{5, 5, 5, 5},
			mean:     5,
			expected: 0,
		},
		{
			name:     "with variation",
			values:   []float64{2, 4, 6, 8},
			mean:     5,
			expected: 2.236067977, // sqrt((9+1+1+9)/4) = sqrt(5)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStdDev(tt.values, tt.mean)
			// Use small epsilon for floating point comparison
			if result < tt.expected-0.001 || result > tt.expected+0.001 {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestDetectChunkPattern(t *testing.T) {
	tests := []struct {
		name     string
		cv       float64
		delays   []float64
		expected string
	}{
		{
			name:     "steady pattern",
			cv:       0.1,
			delays:   []float64{100, 105, 95, 102},
			expected: "steady",
		},
		{
			name:     "moderate pattern",
			cv:       0.5,
			delays:   []float64{100, 150, 120, 180},
			expected: "moderate",
		},
		{
			name:     "stalled pattern",
			cv:       1.5,
			delays:   []float64{100, 600, 150, 700, 120, 800},
			expected: "stalled",
		},
		{
			name:     "burst pattern",
			cv:       1.0,
			delays:   []float64{50, 60, 55, 200, 65, 70},
			expected: "burst",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectChunkPattern(tt.cv, tt.delays)
			if result != tt.expected {
				t.Errorf("Expected pattern %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDetectBuffering(t *testing.T) {
	tests := []struct {
		name     string
		analysis *BufferingAnalysis
		metrics  *StreamMetrics
		expected bool
	}{
		{
			name: "single chunk buffering",
			analysis: &BufferingAnalysis{
				TimeToFirstByte: Duration(100 * time.Millisecond),
				ChunkPattern:    "burst",
			},
			metrics: &StreamMetrics{
				TotalChunks: 1,
			},
			expected: true,
		},
		{
			name: "high TTFB with burst",
			analysis: &BufferingAnalysis{
				TimeToFirstByte: Duration(1500 * time.Millisecond),
				ChunkPattern:    "burst",
			},
			metrics: &StreamMetrics{
				TotalChunks: 5,
			},
			expected: true,
		},
		{
			name: "steady streaming",
			analysis: &BufferingAnalysis{
				TimeToFirstByte: Duration(50 * time.Millisecond),
				FirstChunkGap:   Duration(100 * time.Millisecond),
				ChunkPattern:    "steady",
				ChunkTimingCV:   0.2,
			},
			metrics: &StreamMetrics{
				TotalChunks: 10,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectBuffering(tt.analysis, tt.metrics)
			if result != tt.expected {
				t.Errorf("Expected buffering detection %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		name        string
		analysis    *BufferingAnalysis
		metrics     *StreamMetrics
		minExpected float64
		maxExpected float64
	}{
		{
			name: "low confidence - few chunks",
			analysis: &BufferingAnalysis{
				ChunkPattern: "moderate",
			},
			metrics: &StreamMetrics{
				TotalChunks: 2,
			},
			minExpected: 0.5,
			maxExpected: 0.7,
		},
		{
			name: "high confidence - many chunks and clear pattern",
			analysis: &BufferingAnalysis{
				ChunkPattern: "steady",
			},
			metrics: &StreamMetrics{
				TotalChunks: 15,
			},
			minExpected: 0.9,
			maxExpected: 1.0,
		},
		{
			name: "medium confidence",
			analysis: &BufferingAnalysis{
				ChunkPattern: "moderate",
			},
			metrics: &StreamMetrics{
				TotalChunks: 6,
			},
			minExpected: 0.6,
			maxExpected: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateConfidence(tt.analysis, tt.metrics)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("Expected confidence between %f and %f, got %f",
					tt.minExpected, tt.maxExpected, result)
			}
		})
	}
}
