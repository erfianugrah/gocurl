package metrics

import (
	"sort"
	"sync"
	"time"

	"github.com/erfi/gocurl/internal/client"
)

// Collector collects and aggregates metrics from multiple requests
type Collector struct {
	mu        sync.Mutex
	timings   []*client.TimingBreakdown
	startTime time.Time
	endTime   time.Time
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		timings:   make([]*client.TimingBreakdown, 0),
		startTime: time.Now(),
	}
}

// Record adds a timing measurement to the collector
func (c *Collector) Record(timing *client.TimingBreakdown) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timings = append(c.timings, timing)
}

// Finalize marks the end of data collection
func (c *Collector) Finalize() {
	c.endTime = time.Now()
}

// Calculate computes aggregated statistics from collected measurements
func (c *Collector) Calculate() *Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.timings) == 0 {
		return &Stats{}
	}

	stats := &Stats{
		TotalRequests: len(c.timings),
		StatusCodes:   make(map[int]int),
	}

	// Collect latencies and other metrics
	latencies := make([]time.Duration, 0, len(c.timings))
	var totalLatency time.Duration
	var totalBytes int64

	for _, t := range c.timings {
		latency := time.Duration(t.Total)
		latencies = append(latencies, latency)
		totalLatency += latency
		totalBytes += t.ResponseSize

		if t.Error == "" {
			stats.SuccessfulRequests++
			stats.StatusCodes[t.StatusCode]++
		} else {
			stats.FailedRequests++
		}
	}

	// Sort latencies for percentile calculation
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	// Calculate min, max, mean
	stats.MinLatency = Duration(latencies[0])
	stats.MaxLatency = Duration(latencies[len(latencies)-1])
	stats.MeanLatency = Duration(totalLatency / time.Duration(len(latencies)))

	// Calculate percentiles
	stats.P50 = Duration(percentile(latencies, 50))
	stats.P90 = Duration(percentile(latencies, 90))
	stats.P95 = Duration(percentile(latencies, 95))
	stats.P99 = Duration(percentile(latencies, 99))

	// Calculate extended percentiles if we have enough data
	if len(latencies) >= 1000 {
		stats.P999 = Duration(percentile(latencies, 99.9))
	}
	if len(latencies) >= 10000 {
		stats.P9999 = Duration(percentile(latencies, 99.99))
	}

	// Create histogram
	stats.Histogram = createHistogram(latencies)

	// Calculate throughput
	duration := c.endTime.Sub(c.startTime)
	stats.Duration = Duration(duration)
	stats.RequestsPerSecond = float64(stats.TotalRequests) / duration.Seconds()
	stats.ErrorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests)
	stats.TotalBytes = totalBytes
	stats.BytesPerSecond = float64(totalBytes) / duration.Seconds()

	return stats
}

// percentile calculates the nth percentile from a sorted slice of durations
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation between the two values
	weight := index - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// Reset clears all collected data
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timings = make([]*client.TimingBreakdown, 0)
	c.startTime = time.Now()
}

// createHistogram creates a histogram of latencies with 10ms buckets
func createHistogram(latencies []time.Duration) map[int]int {
	histogram := make(map[int]int)

	for _, latency := range latencies {
		ms := latency.Milliseconds()
		// Create buckets: 0-10ms, 10-20ms, 20-30ms, etc.
		bucket := int(ms / 10)
		histogram[bucket]++
	}

	return histogram
}
