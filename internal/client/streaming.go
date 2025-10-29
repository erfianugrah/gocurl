package client

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

// StreamMetrics captures progressive data delivery characteristics
type StreamMetrics struct {
	ChunkTimings     []ChunkTiming `json:"chunk_timings,omitempty"`
	BytesPerSecond   float64       `json:"bytes_per_second"`
	FirstChunkTime   Duration      `json:"first_chunk_time"`
	LastChunkTime    Duration      `json:"last_chunk_time"`
	TotalChunks      int           `json:"total_chunks"`
	TotalBytes       int64         `json:"total_bytes"`
	AverageChunkSize int64         `json:"average_chunk_size"`

	// HTTP/2 specific
	Protocol         string        `json:"protocol"` // "HTTP/2", "HTTP/1.1", etc.
	StreamID         uint32        `json:"stream_id,omitempty"`

	// Streaming analysis
	StreamingInfo    *StreamingInfo    `json:"streaming_info,omitempty"`
	BufferingAnalysis *BufferingAnalysis `json:"buffering_analysis,omitempty"`
	Stalls           []StallInfo        `json:"stalls,omitempty"`
}

// StreamingInfo contains HTTP response header analysis for streaming detection
type StreamingInfo struct {
	TransferEncoding  string `json:"transfer_encoding"`
	ContentLength     *int64 `json:"content_length"`     // nil = unknown length (streaming likely)
	ContentType       string `json:"content_type"`
	CacheControl      string `json:"cache_control"`
	XAccelBuffering   string `json:"x_accel_buffering"`  // nginx buffering control
	IsChunked         bool   `json:"is_chunked"`
	IsStreamingLikely bool   `json:"is_streaming_likely"` // heuristic
}

// BufferingAnalysis contains analysis of buffering behavior
type BufferingAnalysis struct {
	TimeToFirstByte   Duration `json:"time_to_first_byte"`
	FirstChunkGap     Duration `json:"first_chunk_gap"`      // Gap between first and second chunk
	ChunkPattern      string   `json:"chunk_pattern"`        // "steady", "burst", "stalled", "buffered"
	StallCount        int      `json:"stall_count"`
	TotalStallTime    Duration `json:"total_stall_time"`
	ChunkTimingCV     float64  `json:"chunk_timing_cv"`      // Coefficient of variation
	BufferingDetected bool     `json:"buffering_detected"`

	// Statistical metrics (objective)
	MeanDelay         float64  `json:"mean_delay_ms"`        // Mean inter-chunk delay in milliseconds
	StdDevDelay       float64  `json:"stddev_delay_ms"`      // Standard deviation in milliseconds
	MinDelay          float64  `json:"min_delay_ms"`         // Minimum delay in milliseconds
	MaxDelay          float64  `json:"max_delay_ms"`         // Maximum delay in milliseconds

	// Deprecated: Use objective metrics instead
	StreamingQuality  string   `json:"streaming_quality,omitempty"` // Deprecated: subjective assessment
	Confidence        float64  `json:"confidence"`           // 0-1 confidence score based on sample size
}

// StallInfo represents a pause in data delivery
type StallInfo struct {
	StartTime Duration `json:"start_time"`
	EndTime   Duration `json:"end_time"`
	Duration  Duration `json:"duration"`
	Position  int64    `json:"position"` // bytes received before stall
}

// ChunkTiming represents a single data chunk received
type ChunkTiming struct {
	SequenceNumber int       `json:"sequence"`
	Size           int       `json:"size"`
	ElapsedTime    Duration  `json:"elapsed_time"`
	Timestamp      time.Time `json:"timestamp"`
	Throughput     float64   `json:"throughput_mbps"` // Mbps for this chunk
}

// StreamingReader wraps an io.Reader to capture progressive delivery metrics
type StreamingReader struct {
	reader      io.Reader
	startTime   time.Time
	lastRead    time.Time
	metrics     *StreamMetrics
	chunkNumber int
	totalBytes  int64
}

// NewStreamingReader creates a reader that captures streaming metrics
func NewStreamingReader(reader io.Reader, protocol string) *StreamingReader {
	now := time.Now()
	return &StreamingReader{
		reader:    reader,
		startTime: now,
		lastRead:  now,
		metrics: &StreamMetrics{
			ChunkTimings: make([]ChunkTiming, 0),
			Protocol:     protocol,
		},
		chunkNumber: 0,
		totalBytes:  0,
	}
}

// Read implements io.Reader and captures timing for each read
func (sr *StreamingReader) Read(p []byte) (n int, err error) {
	n, err = sr.reader.Read(p)

	if n > 0 {
		now := time.Now()
		elapsed := now.Sub(sr.startTime)

		// Calculate throughput for this chunk (in Mbps)
		chunkDuration := now.Sub(sr.lastRead).Seconds()
		var throughput float64
		if chunkDuration > 0 {
			throughput = (float64(n) * 8) / (chunkDuration * 1_000_000) // Mbps
		}

		chunk := ChunkTiming{
			SequenceNumber: sr.chunkNumber,
			Size:           n,
			ElapsedTime:    Duration(elapsed),
			Timestamp:      now,
			Throughput:     throughput,
		}

		sr.metrics.ChunkTimings = append(sr.metrics.ChunkTimings, chunk)

		if sr.chunkNumber == 0 {
			sr.metrics.FirstChunkTime = Duration(elapsed)
		}

		sr.chunkNumber++
		sr.totalBytes += int64(n)
		sr.lastRead = now
		sr.metrics.LastChunkTime = Duration(elapsed)
	}

	return n, err
}

// Metrics returns the collected streaming metrics
func (sr *StreamingReader) Metrics() *StreamMetrics {
	sr.metrics.TotalChunks = sr.chunkNumber
	sr.metrics.TotalBytes = sr.totalBytes

	if sr.chunkNumber > 0 {
		sr.metrics.AverageChunkSize = sr.totalBytes / int64(sr.chunkNumber)
	}

	// Calculate overall throughput
	totalDuration := time.Since(sr.startTime).Seconds()
	if totalDuration > 0 {
		sr.metrics.BytesPerSecond = float64(sr.totalBytes) / totalDuration
	}

	return sr.metrics
}

// MeasureRequestWithStreaming executes a request and captures progressive delivery metrics
func (c *Client) MeasureRequestWithStreaming(ctx context.Context, url, method string, headers map[string]string, body io.Reader) (*TimingBreakdown, *StreamMetrics, error) {
	tracer := NewTracer()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, nil, err
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "gocurl/1.0")
	}

	// Attach tracer
	traceCtx := httptrace.WithClientTrace(ctx, tracer.ClientTrace())
	req = req.WithContext(traceCtx)

	// Execute request
	tracer.Start()
	resp, err := c.client.Do(req)
	if err != nil {
		tracer.End()
		timing := tracer.Timing()
		timing.Error = err.Error()
		return timing, nil, err
	}
	defer resp.Body.Close()

	// Analyze response headers for streaming indicators
	streamingInfo := AnalyzeStreamingHeaders(resp)

	// Capture protocol info
	protocol := resp.Proto // "HTTP/2.0", "HTTP/1.1", etc.

	// Wrap response body with streaming reader
	streamReader := NewStreamingReader(resp.Body, protocol)

	// Read body through streaming reader
	var bodyBytes []byte
	shouldCaptureBody := c.config.ShowBody || (c.config.ShowErrorBody && resp.StatusCode >= 400)

	if shouldCaptureBody {
		bodyBytes, err = io.ReadAll(streamReader)
	} else {
		_, err = io.Copy(io.Discard, streamReader)
	}

	tracer.End()

	// Get metrics
	streamMetrics := streamReader.Metrics()
	timing := tracer.Timing()
	timing.StatusCode = resp.StatusCode
	timing.ContentLength = resp.ContentLength
	timing.ResponseSize = streamMetrics.TotalBytes

	// Add streaming info and buffering analysis
	streamMetrics.StreamingInfo = streamingInfo
	if len(streamMetrics.ChunkTimings) > 0 {
		streamMetrics.BufferingAnalysis = AnalyzeBuffering(streamMetrics, timing)
		// Use configured stall threshold or default to 500ms
		threshold := c.config.StallThreshold
		if threshold == 0 {
			threshold = 500 * time.Millisecond
		}
		streamMetrics.Stalls = DetectStalls(streamMetrics, threshold)
	}

	if shouldCaptureBody && len(bodyBytes) > 0 {
		timing.ResponseBody = string(bodyBytes)
	}

	if err != nil {
		timing.Error = err.Error()
	}

	return timing, streamMetrics, nil
}

// AnalyzeStreamingHeaders examines HTTP response headers for streaming indicators
func AnalyzeStreamingHeaders(resp *http.Response) *StreamingInfo {
	info := &StreamingInfo{
		TransferEncoding: strings.Join(resp.TransferEncoding, ", "),
		ContentType:      resp.Header.Get("Content-Type"),
		CacheControl:     resp.Header.Get("Cache-Control"),
		XAccelBuffering:  resp.Header.Get("X-Accel-Buffering"),
	}

	// Check for chunked encoding
	for _, enc := range resp.TransferEncoding {
		if strings.ToLower(enc) == "chunked" {
			info.IsChunked = true
			break
		}
	}

	// Content-Length: if set, body size is known (less likely to be streaming)
	// If -1 or not set, body size is unknown (more likely streaming)
	if resp.ContentLength >= 0 {
		info.ContentLength = &resp.ContentLength
	}

	// Heuristic: streaming is likely if:
	// - Transfer-Encoding is chunked
	// - Content-Length is not set (unknown size)
	// - Content-Type suggests streaming (SSE, event-stream, etc.)
	// - X-Accel-Buffering is off
	info.IsStreamingLikely = info.IsChunked ||
		info.ContentLength == nil ||
		strings.Contains(strings.ToLower(info.ContentType), "stream") ||
		strings.Contains(strings.ToLower(info.ContentType), "event-stream") ||
		strings.ToLower(info.XAccelBuffering) == "no"

	return info
}

// AnalyzeBuffering performs statistical analysis to detect buffering behavior
func AnalyzeBuffering(metrics *StreamMetrics, timing *TimingBreakdown) *BufferingAnalysis {
	analysis := &BufferingAnalysis{
		TimeToFirstByte: metrics.FirstChunkTime,
		MinDelay:        -1, // Initialize to -1 to indicate no data
		MaxDelay:        -1,
	}

	if len(metrics.ChunkTimings) < 2 {
		// Not enough data for analysis
		analysis.ChunkPattern = "insufficient_data"
		return analysis
	}

	// Calculate gap between first and second chunk
	analysis.FirstChunkGap = Duration(time.Duration(metrics.ChunkTimings[1].ElapsedTime) - time.Duration(metrics.ChunkTimings[0].ElapsedTime))

	// Calculate inter-chunk timings
	interChunkDelays := make([]float64, len(metrics.ChunkTimings)-1)
	for i := 1; i < len(metrics.ChunkTimings); i++ {
		delay := time.Duration(metrics.ChunkTimings[i].ElapsedTime) - time.Duration(metrics.ChunkTimings[i-1].ElapsedTime)
		interChunkDelays[i-1] = float64(delay.Milliseconds())
	}

	// Calculate statistical metrics
	analysis.MeanDelay = calculateMean(interChunkDelays)
	analysis.StdDevDelay = calculateStdDev(interChunkDelays, analysis.MeanDelay)

	// Calculate min/max delays
	if len(interChunkDelays) > 0 {
		analysis.MinDelay = interChunkDelays[0]
		analysis.MaxDelay = interChunkDelays[0]
		for _, delay := range interChunkDelays {
			if delay < analysis.MinDelay {
				analysis.MinDelay = delay
			}
			if delay > analysis.MaxDelay {
				analysis.MaxDelay = delay
			}
		}
	}

	// Calculate coefficient of variation (CV = stddev / mean)
	if analysis.MeanDelay > 0 {
		analysis.ChunkTimingCV = analysis.StdDevDelay / analysis.MeanDelay
	}

	// Detect pattern based on CV and timing characteristics
	analysis.ChunkPattern = detectChunkPattern(analysis.ChunkTimingCV, interChunkDelays)

	// Detect buffering based on multiple signals
	analysis.BufferingDetected = detectBuffering(analysis, metrics)

	// Calculate confidence score
	analysis.Confidence = calculateConfidence(analysis, metrics)

	return analysis
}

// DetectStalls identifies pauses in data delivery
func DetectStalls(metrics *StreamMetrics, threshold time.Duration) []StallInfo {
	if len(metrics.ChunkTimings) < 2 {
		return nil
	}

	stalls := make([]StallInfo, 0)
	var totalBytes int64

	for i := 1; i < len(metrics.ChunkTimings); i++ {
		prev := metrics.ChunkTimings[i-1]
		curr := metrics.ChunkTimings[i]

		delay := time.Duration(curr.ElapsedTime) - time.Duration(prev.ElapsedTime)
		if delay > threshold {
			stall := StallInfo{
				StartTime: prev.ElapsedTime,
				EndTime:   curr.ElapsedTime,
				Duration:  Duration(delay),
				Position:  totalBytes,
			}
			stalls = append(stalls, stall)
		}

		totalBytes += int64(prev.Size)
	}

	return stalls
}

// Helper functions for statistical analysis
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(values)))
}

func detectChunkPattern(cv float64, delays []float64) string {
	// CV < 0.3: very steady (low variation)
	// CV 0.3-0.7: moderate variation
	// CV > 0.7: high variation (burst or stalled)

	if cv < 0.3 {
		return "steady"
	} else if cv < 0.7 {
		return "moderate"
	}

	// Check if it's burst (many small delays then big delays)
	// vs stalled (frequent long pauses)
	longDelays := 0
	for _, d := range delays {
		if d > 500 { // > 500ms
			longDelays++
		}
	}

	stallRatio := float64(longDelays) / float64(len(delays))
	if stallRatio > 0.3 {
		return "stalled"
	}

	return "burst"
}

func detectBuffering(analysis *BufferingAnalysis, metrics *StreamMetrics) bool {
	// Multiple signals indicate buffering:
	// 1. High TTFB with burst delivery
	// 2. All data arrives in single chunk
	// 3. Very high first chunk gap
	// 4. Low CV with high TTFB (data was buffered then sent)

	signals := 0

	// Signal 1: Single chunk delivery
	if metrics.TotalChunks == 1 {
		signals += 2 // Strong signal
	}

	// Signal 2: High TTFB (> 1s) with burst pattern
	if time.Duration(analysis.TimeToFirstByte) > time.Second && analysis.ChunkPattern == "burst" {
		signals++
	}

	// Signal 3: Very high first chunk gap (> 1s)
	if time.Duration(analysis.FirstChunkGap) > time.Second {
		signals++
	}

	// Signal 4: Low variation with high TTFB suggests buffering
	if analysis.ChunkTimingCV < 0.3 && time.Duration(analysis.TimeToFirstByte) > 500*time.Millisecond {
		signals++
	}

	return signals >= 2
}

func calculateConfidence(analysis *BufferingAnalysis, metrics *StreamMetrics) float64 {
	// Confidence increases with more data points and clearer patterns
	confidence := 0.5 // Base confidence

	// More chunks = higher confidence in statistical analysis
	if metrics.TotalChunks >= 10 {
		confidence += 0.3
	} else if metrics.TotalChunks >= 5 {
		confidence += 0.2
	} else if metrics.TotalChunks >= 2 {
		confidence += 0.1
	}

	// Clear patterns increase confidence
	// Steady pattern or buffering detection both indicate clear behavior
	if analysis.ChunkPattern == "steady" || analysis.BufferingDetected {
		confidence += 0.2
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}
