package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
	"github.com/erfi/gocurl/internal/output"
)

// Config contains application configuration
type Config struct {
	URLs            []string
	Method          string
	Headers         []string
	Data            string
	Requests        int
	Concurrency     int
	Duration        string
	Timeout         string
	Insecure        bool
	OutputFormat    string
	Verbose         bool
	Quiet           bool
	IncludeHeaders  bool
	ShowBody        bool
	ShowErrorBody   bool
	EnableStreaming bool
	ResolveHosts    []string
	ConnectToHosts  []string
	ExpectStreaming bool
	StallThreshold  string
}

// App represents the main application
type App struct {
	config    *Config
	client    *client.Client
	collector *metrics.Collector
	formatter output.Formatter
}

// New creates a new application instance
func New(config *Config) *App {
	// Parse timeout
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// Parse --resolve and --connect-to mappings
	resolveMap, err := client.ParseResolveHosts(config.ResolveHosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		resolveMap = make(map[string]string)
	}

	connectToMap, err := client.ParseConnectToHosts(config.ConnectToHosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		connectToMap = make(map[string]string)
	}

	// Parse stall threshold
	stallThreshold := 500 * time.Millisecond // default
	if config.StallThreshold != "" {
		parsed, err := time.ParseDuration(config.StallThreshold)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid stall threshold '%s', using default 500ms\n", config.StallThreshold)
		} else {
			stallThreshold = parsed
		}
	}

	// Configure HTTP client based on number of requests
	clientConfig := &client.Config{
		Timeout:        timeout,
		Insecure:       config.Insecure,
		IncludeHeaders: config.IncludeHeaders,
		ShowBody:       config.ShowBody,
		ShowErrorBody:  config.ShowErrorBody,
		ResolveMap:     resolveMap,
		ConnectToMap:   connectToMap,
		StallThreshold: stallThreshold,
	}

	if config.Requests == 1 {
		// Single request: disable keep-alives to measure connection establishment
		clientConfig.DisableKeepAlive = true
		clientConfig.MaxIdleConns = 1
		clientConfig.MaxIdlePerHost = 1
	} else {
		// Load testing: enable connection pooling
		clientConfig.DisableKeepAlive = false
		clientConfig.MaxIdleConns = config.Concurrency
		clientConfig.MaxIdlePerHost = config.Concurrency
	}

	httpClient := client.NewClient(clientConfig)
	collector := metrics.NewCollector()
	formatter, _ := output.GetFormatter(config.OutputFormat, config.Verbose)

	return &App{
		config:    config,
		client:    httpClient,
		collector: collector,
		formatter: formatter,
	}
}

// Run executes the application
func (a *App) Run() error {
	if a.config.Requests == 1 {
		return a.runSingle()
	}
	return a.runLoad()
}

// runSingle executes a single request
func (a *App) runSingle() error {
	if len(a.config.URLs) == 0 {
		return fmt.Errorf("no URLs provided")
	}

	url := a.config.URLs[0]
	headers := client.ParseHeaders(a.config.Headers)

	var body io.Reader
	if a.config.Data != "" {
		body = strings.NewReader(a.config.Data)
	}

	var timing *client.TimingBreakdown
	var streamMetrics *client.StreamMetrics
	var err error

	// Use streaming measurement if enabled
	if a.config.EnableStreaming {
		timing, streamMetrics, err = a.client.MeasureRequestWithStreaming(
			context.Background(),
			url,
			a.config.Method,
			headers,
			body,
		)
		// Attach streaming metrics to timing for JSON output
		if timing != nil && streamMetrics != nil {
			timing.Streaming = streamMetrics
		}
	} else {
		timing, err = a.client.MeasureRequest(
			url,
			a.config.Method,
			headers,
			body,
		)
	}

	if err != nil && timing == nil {
		return fmt.Errorf("request failed: %w", err)
	}

	// Output the timing result
	if err := a.formatter.Write(os.Stdout, timing); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// For table output, also write streaming metrics separately
	if streamMetrics != nil && a.config.OutputFormat == "table" {
		output.WriteStreamingMetrics(os.Stdout, streamMetrics, a.config.Verbose)
	}

	// Validate streaming expectation
	if a.config.ExpectStreaming && streamMetrics != nil {
		if err := a.validateStreaming(streamMetrics); err != nil {
			return err
		}
	}

	// Return error if request failed (but output was still produced)
	if timing.Error != "" {
		return fmt.Errorf("request error: %s", timing.Error)
	}

	return nil
}

// validateStreaming checks if streaming requirements are met
func (a *App) validateStreaming(metrics *client.StreamMetrics) error {
	// Check if streaming was detected
	if metrics.BufferingAnalysis == nil {
		return fmt.Errorf("streaming validation failed: no buffering analysis available")
	}

	if metrics.BufferingAnalysis.BufferingDetected {
		return fmt.Errorf("streaming validation failed: buffering detected (pattern: %s, CV: %.2f)",
			metrics.BufferingAnalysis.ChunkPattern,
			metrics.BufferingAnalysis.ChunkTimingCV)
	}

	if metrics.StreamingInfo != nil && !metrics.StreamingInfo.IsStreamingLikely {
		return fmt.Errorf("streaming validation failed: response headers do not indicate streaming")
	}

	// Success - streaming detected
	if !a.config.Quiet {
		fmt.Fprintf(os.Stdout, "\n✓ Streaming validation passed (pattern: %s, CV: %.2f, %d chunks)\n",
			metrics.BufferingAnalysis.ChunkPattern,
			metrics.BufferingAnalysis.ChunkTimingCV,
			metrics.TotalChunks)
	}

	return nil
}

// runLoad executes multiple concurrent requests
func (a *App) runLoad() error {
	if len(a.config.URLs) == 0 {
		return fmt.Errorf("no URLs provided")
	}

	totalRequests := a.config.Requests * len(a.config.URLs)

	if !a.config.Quiet {
		fmt.Printf("Running load test: %d URLs x %d requests = %d total requests with concurrency %d\n",
			len(a.config.URLs), a.config.Requests, totalRequests, a.config.Concurrency)
	}

	headers := client.ParseHeaders(a.config.Headers)

	// Create worker pool
	type job struct {
		url string
		id  int
	}

	jobs := make(chan job, totalRequests)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < a.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				var body io.Reader
				if a.config.Data != "" {
					body = strings.NewReader(a.config.Data)
				}

				timing, _ := a.client.MeasureRequest(
					j.url,
					a.config.Method,
					headers,
					body,
				)

				if timing != nil {
					a.collector.Record(timing)
				}
			}
		}()
	}

	// Send jobs for each URL
	jobID := 0
	for _, url := range a.config.URLs {
		for i := 0; i < a.config.Requests; i++ {
			jobs <- job{url: url, id: jobID}
			jobID++
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	a.collector.Finalize()

	// Calculate and display statistics
	stats := a.collector.Calculate()

	if err := a.formatter.WriteMultiple(os.Stdout, stats); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}
