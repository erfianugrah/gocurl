package metrics

import (
	"github.com/erfi/gocurl/internal/client"
)

// Duration is an alias to client.Duration for backward compatibility
type Duration = client.Duration

// Stats contains aggregated statistics from multiple requests
type Stats struct {
	TotalRequests      int                `json:"total_requests"`
	SuccessfulRequests int                `json:"successful_requests"`
	FailedRequests     int                `json:"failed_requests"`
	Duration           Duration           `json:"duration"`
	RequestsPerSecond  float64            `json:"requests_per_second"`
	MinLatency         Duration           `json:"min_latency"`
	MaxLatency         Duration           `json:"max_latency"`
	MeanLatency        Duration           `json:"mean_latency"`
	P50                Duration           `json:"p50"`
	P90                Duration           `json:"p90"`
	P95                Duration           `json:"p95"`
	P99                Duration           `json:"p99"`
	P999               Duration           `json:"p99_9,omitempty"`
	P9999              Duration           `json:"p99_99,omitempty"`
	StatusCodes        map[int]int        `json:"status_codes"`
	ErrorRate          float64            `json:"error_rate"`
	TotalBytes         int64              `json:"total_bytes"`
	BytesPerSecond     float64            `json:"bytes_per_second"`
	Histogram          map[int]int        `json:"histogram,omitempty"`
}
