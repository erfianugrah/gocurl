package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/fatih/color"
)

// WriteStreamingMetrics outputs detailed streaming performance metrics
func WriteStreamingMetrics(w io.Writer, metrics *client.StreamMetrics, verbose bool) {
	if metrics == nil {
		return
	}

	fmt.Fprintln(w)

	// Streaming Info Section
	if metrics.StreamingInfo != nil {
		fmt.Fprintf(w, "%s\n", color.CyanString("Streaming Analysis:"))
		info := metrics.StreamingInfo

		if info.IsStreamingLikely {
			fmt.Fprintf(w, "  %s Streaming detected\n", color.GreenString("✓"))
		} else {
			fmt.Fprintf(w, "  %s Streaming not detected\n", color.YellowString("⚠"))
		}

		if info.IsChunked {
			fmt.Fprintf(w, "  • Transfer-Encoding: %s\n", color.GreenString("chunked"))
		}

		if info.ContentLength != nil {
			fmt.Fprintf(w, "  • Content-Length: %s\n", formatBytes(*info.ContentLength))
		} else {
			fmt.Fprintf(w, "  • Content-Length: %s\n", color.GreenString("not set (streaming)"))
		}

		if info.ContentType != "" {
			fmt.Fprintf(w, "  • Content-Type: %s\n", info.ContentType)
		}

		if info.XAccelBuffering != "" {
			fmt.Fprintf(w, "  • X-Accel-Buffering: %s\n", info.XAccelBuffering)
		}

		fmt.Fprintln(w)
	}

	// Buffering Analysis Section
	if metrics.BufferingAnalysis != nil {
		analysis := metrics.BufferingAnalysis
		fmt.Fprintf(w, "%s\n", color.CyanString("Delivery Characteristics:"))

		// Chunk delivery pattern
		fmt.Fprintf(w, "  Pattern: %s\n", analysis.ChunkPattern)

		// Buffering detection
		if analysis.BufferingDetected {
			fmt.Fprintf(w, "  Status: %s\n", color.RedString("✗ Buffering detected"))
		} else {
			fmt.Fprintf(w, "  Status: %s\n", color.GreenString("✓ Progressive delivery"))
		}

		// Statistical metrics
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", color.CyanString("Timing Statistics:"))
		fmt.Fprintf(w, "    Time to first byte: %s\n", formatDuration(analysis.TimeToFirstByte))
		if analysis.FirstChunkGap > 0 {
			fmt.Fprintf(w, "    First chunk gap: %s\n", formatDuration(analysis.FirstChunkGap))
		}

		// Inter-chunk timing variability
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", color.CyanString("Inter-Chunk Timing:"))
		fmt.Fprintf(w, "    Coefficient of Variation: %.2f\n", analysis.ChunkTimingCV)

		if analysis.MeanDelay > 0 {
			fmt.Fprintf(w, "    Mean delay: %.2fms\n", analysis.MeanDelay)
			fmt.Fprintf(w, "    Std deviation: %.2fms\n", analysis.StdDevDelay)
			if analysis.MinDelay >= 0 && analysis.MaxDelay >= 0 {
				fmt.Fprintf(w, "    Range: %.2fms - %.2fms\n", analysis.MinDelay, analysis.MaxDelay)
			}
		}

		// Confidence
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  Analysis confidence: %.0f%% (%d chunks)\n",
			analysis.Confidence*100, metrics.TotalChunks)

		fmt.Fprintln(w)
	}

	// Stalls Section
	if len(metrics.Stalls) > 0 {
		fmt.Fprintf(w, "%s\n", color.YellowString("Stalls Detected:"))
		fmt.Fprintf(w, "  Count: %d\n", len(metrics.Stalls))

		totalStallTime := time.Duration(0)
		for _, stall := range metrics.Stalls {
			totalStallTime += time.Duration(stall.Duration)
		}
		fmt.Fprintf(w, "  Total stall time: %s\n", formatDuration(client.Duration(totalStallTime)))

		if verbose {
			for i, stall := range metrics.Stalls {
				fmt.Fprintf(w, "  #%d: %s at %s (after %s)\n",
					i+1,
					formatDuration(stall.Duration),
					formatDuration(stall.StartTime),
					formatBytes(stall.Position))
			}
		}

		fmt.Fprintln(w)
	}

	// Performance metrics
	fmt.Fprintf(w, "%s\n", color.CyanString("Performance Metrics:"))
	fmt.Fprintf(w, "  Protocol: %s\n", metrics.Protocol)
	fmt.Fprintf(w, "  Total Chunks: %d\n", metrics.TotalChunks)
	fmt.Fprintf(w, "  Total Bytes: %s\n", formatBytes(metrics.TotalBytes))
	fmt.Fprintf(w, "  Average Chunk Size: %s\n", formatBytes(metrics.AverageChunkSize))
	fmt.Fprintf(w, "  Throughput: %s/s\n", formatBytes(int64(metrics.BytesPerSecond)))
	fmt.Fprintf(w, "  First Chunk: %s\n", formatDuration(metrics.FirstChunkTime))
	fmt.Fprintf(w, "  Last Chunk: %s\n", formatDuration(metrics.LastChunkTime))

	if verbose && len(metrics.ChunkTimings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s\n", color.CyanString("Chunk Timeline:"))
		drawChunkTimeline(w, metrics)

		// Show detailed chunk stats
		if len(metrics.ChunkTimings) <= 20 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "%s\n", color.CyanString("Chunk Details:"))
			for _, chunk := range metrics.ChunkTimings {
				fmt.Fprintf(w, "  #%-3d %6s at %7s (%6.2f Mbps)\n",
					chunk.SequenceNumber,
					formatBytes(int64(chunk.Size)),
					formatDuration(chunk.ElapsedTime),
					chunk.Throughput,
				)
			}
		}
	}
}

// drawChunkTimeline creates a visual timeline of data chunks
func drawChunkTimeline(w io.Writer, metrics *client.StreamMetrics) {
	if len(metrics.ChunkTimings) == 0 {
		return
	}

	maxWidth := 60
	totalTime := metrics.LastChunkTime

	if totalTime == 0 {
		return
	}

	// Group chunks into time buckets for visualization
	numBuckets := 20
	buckets := make([]int64, numBuckets)

	for _, chunk := range metrics.ChunkTimings {
		bucketIdx := int(float64(chunk.ElapsedTime) / float64(totalTime) * float64(numBuckets-1))
		if bucketIdx >= numBuckets {
			bucketIdx = numBuckets - 1
		}
		buckets[bucketIdx] += int64(chunk.Size)
	}

	// Find max bucket for scaling
	var maxBucket int64
	for _, size := range buckets {
		if size > maxBucket {
			maxBucket = size
		}
	}

	if maxBucket == 0 {
		return
	}

	// Draw histogram
	fmt.Fprintf(w, "  Data received over time:\n")
	for i, size := range buckets {
		barWidth := int(float64(size) / float64(maxBucket) * float64(maxWidth))
		if size > 0 && barWidth == 0 {
			barWidth = 1
		}

		timeMarker := float64(i) / float64(numBuckets) * float64(totalTime.Milliseconds())
		fmt.Fprintf(w, "  %5.0fms │%s%s │ %s\n",
			timeMarker,
			color.GreenString(strings.Repeat("█", barWidth)),
			strings.Repeat("░", maxWidth-barWidth),
			formatBytes(size),
		)
	}
}
