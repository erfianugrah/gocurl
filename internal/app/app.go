package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
	"github.com/erfi/gocurl/internal/output"
	"github.com/erfi/gocurl/internal/scenario"
)

// Config contains application configuration
type Config struct {
	URLs            []string
	Method          string
	Headers         []string
	Data            string
	Requests        int
	Concurrency     int
	WarmupRequests  int
	RPS             int
	RampUp          string // Duration for ramping up concurrency (e.g., "30s")
	ExportCSV       string
	Timeout         string
	Insecure        bool
	OutputFormat    string
	Verbose         bool
	Quiet           bool
	IncludeHeaders  bool
	ShowBody        bool
	ShowErrorBody   bool
	CaptureHeaders  []string // Specific headers to capture
	RangeHeader     string   // Range header for partial content
	EnableStreaming bool
	ResolveHosts    []string
	ConnectToHosts  []string
	ExpectStreaming bool
	StallThreshold  string

	// Version information (for output metadata)
	Version   string
	Commit    string
	BuildDate string
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
		CaptureHeaders: config.CaptureHeaders,
		RangeHeader:    config.RangeHeader,
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
	formatter, _ := output.GetFormatter(config.OutputFormat, config.Verbose, config.Version, config.Commit, config.BuildDate)

	return &App{
		config:    config,
		client:    httpClient,
		collector: collector,
		formatter: formatter,
	}
}

// Run executes the application
func (a *App) Run() error {
	// Use runSingle only for a single URL with a single request
	// For multiple URLs, always use runLoad even with -n 1
	if a.config.Requests == 1 && len(a.config.URLs) == 1 {
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

	// Cancel the in-flight request on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var timing *client.TimingBreakdown
	var streamMetrics *client.StreamMetrics
	var err error

	// Use streaming measurement if enabled
	if a.config.EnableStreaming {
		timing, streamMetrics, err = a.client.MeasureRequestWithStreaming(
			ctx,
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
		timing, err = a.client.MeasureRequestContext(
			ctx,
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
		fmt.Fprintf(os.Stderr, "\n✓ Streaming validation passed (pattern: %s, CV: %.2f, %d chunks)\n",
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

	// Streaming analysis is a single-request deep inspection. Silently ignoring
	// it under load previously let --expect-streaming exit 0 (a false pass) in
	// CI. Reject the combination explicitly instead.
	if a.config.EnableStreaming {
		return fmt.Errorf("--streaming/--expect-streaming is only supported for a single request (one URL with -n 1); it is not available in load-test mode")
	}

	// Cancel in-flight work on SIGINT/SIGTERM and report partial results.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	totalRequests := a.config.Requests * len(a.config.URLs)
	warmupTotal := a.config.WarmupRequests * len(a.config.URLs)

	if !a.config.Quiet {
		if a.config.WarmupRequests > 0 {
			fmt.Fprintf(os.Stderr, "Running load test: %d URLs x %d requests = %d total requests with concurrency %d\n",
				len(a.config.URLs), a.config.Requests, totalRequests, a.config.Concurrency)
			fmt.Fprintf(os.Stderr, "Warmup: %d requests per URL (%d total warmup) - not included in metrics\n",
				a.config.WarmupRequests, warmupTotal)
		} else {
			fmt.Fprintf(os.Stderr, "Running load test: %d URLs x %d requests = %d total requests with concurrency %d\n",
				len(a.config.URLs), a.config.Requests, totalRequests, a.config.Concurrency)
		}
		if a.config.RPS > 0 {
			fmt.Fprintf(os.Stderr, "Rate limit: %d requests/second\n", a.config.RPS)
		}
	}

	headers := client.ParseHeaders(a.config.Headers)

	// Create worker pool
	type job struct {
		url      string
		id       int
		urlIndex int
		isWarmup bool
	}

	jobs := make(chan job, totalRequests)
	var wg sync.WaitGroup

	// Pre-fill the job queue and close it up front. The channel is buffered to
	// the full request count so these sends never block. Filling before the
	// workers start is what lets ramp-up actually stagger load: workers activate
	// over time and pull from an already-populated queue (previously the queue
	// was filled only after the ramp sleep, so no gradual load ever occurred).
	jobID := 0
	for urlIndex, url := range a.config.URLs {
		for i := 0; i < a.config.Requests; i++ {
			jobs <- job{url: url, id: jobID, urlIndex: urlIndex, isWarmup: i < a.config.WarmupRequests}
			jobID++
		}
	}
	close(jobs)

	// Check if data contains template variables
	hasTemplates := HasTemplateVariables(a.config.Data)

	// Parse ramp-up duration if specified
	var rampUpDuration time.Duration
	if a.config.RampUp != "" {
		var err error
		rampUpDuration, err = time.ParseDuration(a.config.RampUp)
		if err != nil {
			return fmt.Errorf("invalid ramp-up duration: %w", err)
		}
		if !a.config.Quiet {
			fmt.Fprintf(os.Stderr, "Ramp-up: gradually increasing from 1 to %d workers over %s\n", a.config.Concurrency, rampUpDuration)
		}
	}

	// Create rate limiter if RPS is specified
	var rateLimiter <-chan time.Time
	if a.config.RPS > 0 {
		// Create ticker for rate limiting
		interval := time.Second / time.Duration(a.config.RPS)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		rateLimiter = ticker.C
	}

	// Progress reporting
	var completed int64
	var progressDone chan struct{}
	if !a.config.Quiet {
		progressDone = make(chan struct{})
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					current := atomic.LoadInt64(&completed)
					pct := float64(current) / float64(totalRequests) * 100
					fmt.Fprintf(os.Stderr, "\rProgress: %d/%d requests (%.1f%%)", current, totalRequests, pct)
				case <-progressDone:
					// Final update reflects the actual count (may be < total if
					// the run was interrupted).
					final := atomic.LoadInt64(&completed)
					fmt.Fprintf(os.Stderr, "\rProgress: %d/%d requests (%.1f%%)\n", final, totalRequests, float64(final)/float64(totalRequests)*100)
					return
				}
			}
		}()
	}

	// Per-worker activation stagger for ramp-up: worker i becomes active at
	// t = stagger*i, so effective concurrency climbs linearly from 1 to
	// Concurrency over the ramp-up duration.
	var stagger time.Duration
	if rampUpDuration > 0 && a.config.Concurrency > 1 {
		stagger = rampUpDuration / time.Duration(a.config.Concurrency-1)
	}

	worker := func() {
		for j := range jobs {
			// Stop pulling new work once cancelled (e.g. SIGINT).
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Wait for the rate limiter if enabled, aborting on cancel.
			if rateLimiter != nil {
				select {
				case <-rateLimiter:
				case <-ctx.Done():
					return
				}
			}

			var body io.Reader
			if a.config.Data != "" {
				data := a.config.Data
				if hasTemplates {
					tctx := NewTemplateContext(j.id+1, j.urlIndex) // 1-indexed seq
					data = SubstituteTemplate(data, tctx)
				}
				body = strings.NewReader(data)
			}

			timing, _ := a.client.MeasureRequestContext(ctx, j.url, a.config.Method, headers, body)

			// Only record metrics for non-warmup requests that actually ran.
			if timing != nil && !j.isWarmup && ctx.Err() == nil {
				a.collector.Record(timing)
			}

			atomic.AddInt64(&completed, 1)
		}
	}

	for i := 0; i < a.config.Concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if stagger > 0 && idx > 0 {
				select {
				case <-time.After(stagger * time.Duration(idx)):
				case <-ctx.Done():
					return
				}
			}
			worker()
		}(i)
	}

	// Wait for all workers to complete (or exit early on cancellation).
	wg.Wait()

	// Stop progress reporting
	if !a.config.Quiet {
		close(progressDone)
		time.Sleep(100 * time.Millisecond) // Give progress goroutine time to print final message
	}

	interrupted := ctx.Err() != nil
	if interrupted && !a.config.Quiet {
		fmt.Fprintf(os.Stderr, "\nInterrupted - reporting statistics for completed requests only.\n")
	}

	a.collector.Finalize()

	// Calculate and display statistics
	stats := a.collector.Calculate()

	if err := a.formatter.WriteMultiple(os.Stdout, stats); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Export to CSV if requested
	if a.config.ExportCSV != "" {
		timings := a.collector.GetTimings()
		if err := output.ExportCSV(a.config.ExportCSV, timings); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		if !a.config.Quiet {
			fmt.Fprintf(os.Stderr, "\nExported %d requests to %s\n", len(timings), a.config.ExportCSV)
		}
	}

	if interrupted {
		return fmt.Errorf("interrupted")
	}

	return nil
}

// LoadAndExecuteScenario loads and executes a scenario from a YAML file
func LoadAndExecuteScenario(filename string, insecure bool, timeoutStr string) (*scenario.Scenario, error) {
	// Load scenario
	sc, err := scenario.LoadScenario(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load scenario: %w", err)
	}

	// Create execution context
	ctx, err := scenario.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	// Parse timeout
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 30 * time.Second
	}
	if sc.Config.Timeout != "" {
		parsed, err := time.ParseDuration(sc.Config.Timeout)
		if err == nil {
			timeout = parsed
		}
	}

	// Create HTTP client with cookie jar
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure || sc.Config.Insecure,
		},
	}

	ctx.Client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
		Jar:       ctx.CookieJar, // Enable automatic cookie handling
	}

	// Execute scenario
	fmt.Printf("=== Running Scenario: %s ===\n", sc.Name)
	if sc.Description != "" {
		fmt.Printf("Description: %s\n", sc.Description)
	}
	fmt.Printf("\n")

	if err := sc.Execute(ctx); err != nil {
		// Print results even on failure
		fmt.Fprintf(os.Stderr, "\n❌ Scenario failed: %v\n\n", err)
		printScenarioResults(ctx)
		return sc, err
	}

	// Print results
	printScenarioResults(ctx)

	return sc, nil
}

func printScenarioResults(ctx *scenario.Context) {
	fmt.Printf("=== Scenario Results ===\n\n")

	for i, result := range ctx.Results {
		status := "✓"
		if result.Error != "" {
			status = "✗"
		}

		fmt.Printf("%s Step %d: %s\n", status, i+1, result.StepName)
		fmt.Printf("  URL: %s\n", result.URL)
		fmt.Printf("  Method: %s\n", result.Method)
		fmt.Printf("  Status: %d\n", result.StatusCode)
		fmt.Printf("  Duration: %v\n", result.Duration)
		fmt.Printf("  Size: %d bytes\n", result.ResponseSize)

		if len(result.ExtractedVars) > 0 {
			fmt.Printf("  Extracted variables:\n")
			for key, value := range result.ExtractedVars {
				// Truncate long values
				displayValue := value
				if len(displayValue) > 50 {
					displayValue = displayValue[:47] + "..."
				}
				fmt.Printf("    %s = %s\n", key, displayValue)
			}
		}

		if result.Error != "" {
			fmt.Printf("  ❌ Error: %s\n", result.Error)
		}

		fmt.Printf("\n")
	}

	fmt.Printf("Total Steps: %d\n", len(ctx.Results))
	successCount := 0
	for _, r := range ctx.Results {
		if r.Error == "" {
			successCount++
		}
	}
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", len(ctx.Results)-successCount)
}
