package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
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
	warmupTotal := a.config.WarmupRequests * len(a.config.URLs)

	if !a.config.Quiet {
		if a.config.WarmupRequests > 0 {
			fmt.Printf("Running load test: %d URLs x %d requests = %d total requests with concurrency %d\n",
				len(a.config.URLs), a.config.Requests, totalRequests, a.config.Concurrency)
			fmt.Printf("Warmup: %d requests per URL (%d total warmup) - not included in metrics\n",
				a.config.WarmupRequests, warmupTotal)
		} else {
			fmt.Printf("Running load test: %d URLs x %d requests = %d total requests with concurrency %d\n",
				len(a.config.URLs), a.config.Requests, totalRequests, a.config.Concurrency)
		}
		if a.config.RPS > 0 {
			fmt.Printf("Rate limit: %d requests/second\n", a.config.RPS)
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
			fmt.Printf("Ramp-up: gradually increasing from 1 to %d workers over %s\n", a.config.Concurrency, rampUpDuration)
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

	// Start workers (with ramp-up if configured)
	if rampUpDuration > 0 && a.config.Concurrency > 1 {
		// Ramp-up mode: start workers gradually
		interval := rampUpDuration / time.Duration(a.config.Concurrency-1)
		for i := 0; i < a.config.Concurrency; i++ {
			if i > 0 {
				time.Sleep(interval)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					// Wait for rate limiter if enabled
					if rateLimiter != nil {
						<-rateLimiter
					}

					var body io.Reader
					if a.config.Data != "" {
						data := a.config.Data
						// Apply template substitution if needed
						if hasTemplates {
							ctx := NewTemplateContext(j.id+1, j.urlIndex) // Use 1-indexed seq
							data = SubstituteTemplate(data, ctx)
						}
						body = strings.NewReader(data)
					}

					timing, _ := a.client.MeasureRequest(
						j.url,
						a.config.Method,
						headers,
						body,
					)

					// Only record metrics if not a warmup request
					if timing != nil && !j.isWarmup {
						a.collector.Record(timing)
					}
				}
			}()
		}
	} else {
		// Normal mode: start all workers immediately
		for i := 0; i < a.config.Concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					// Wait for rate limiter if enabled
					if rateLimiter != nil {
						<-rateLimiter
					}

					var body io.Reader
					if a.config.Data != "" {
						data := a.config.Data
						// Apply template substitution if needed
						if hasTemplates {
							ctx := NewTemplateContext(j.id+1, j.urlIndex) // Use 1-indexed seq
							data = SubstituteTemplate(data, ctx)
						}
						body = strings.NewReader(data)
					}

					timing, _ := a.client.MeasureRequest(
						j.url,
						a.config.Method,
						headers,
						body,
					)

					// Only record metrics if not a warmup request
					if timing != nil && !j.isWarmup {
						a.collector.Record(timing)
					}
				}
			}()
		}
	}

	// Send jobs for each URL
	jobID := 0
	for urlIndex, url := range a.config.URLs {
		for i := 0; i < a.config.Requests; i++ {
			isWarmup := i < a.config.WarmupRequests
			jobs <- job{url: url, id: jobID, urlIndex: urlIndex, isWarmup: isWarmup}
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

	// Export to CSV if requested
	if a.config.ExportCSV != "" {
		timings := a.collector.GetTimings()
		if err := output.ExportCSV(a.config.ExportCSV, timings); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		if !a.config.Quiet {
			fmt.Printf("\nExported %d requests to %s\n", len(timings), a.config.ExportCSV)
		}
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
