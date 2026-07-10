package main

import (
	"fmt"
	"os"

	"github.com/erfi/gocurl/internal/app"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	outputFormat    string
	noColor         bool
	verbose         bool
	quiet           bool
	requests        int
	concurrency     int
	headers         []string
	method          string
	data            string
	timeout         string
	insecure        bool
	urlListFile     string
	useStdin        bool
	includeHeaders  bool
	showBody        bool
	showErrorBody   bool
	headRequest     bool
	enableStreaming bool
	resolveHosts    []string
	connectToHosts  []string
	expectStreaming bool
	stallThreshold  string
	queryParamsFile string
	warmupRequests  int
	rps             int
	rampUp          string
	exportCSV       string
	captureHeaders  []string
	rangeHeader     string
	scenarioFile    string
)

var rootCmd = &cobra.Command{
	Use:     "gocurl [flags] [url]",
	Short:   "A Go-based performance measurement CLI tool that extends curl functionality",
	Version: version,
	Long: fmt.Sprintf(`%s

gocurl is a production-grade performance measurement tool that provides
rich metrics, multiple output formats, and load testing capabilities.

It measures detailed HTTP performance metrics including DNS lookup time,
TCP connection time, TLS handshake time, server processing time, and more.`, GetVersionInfo()),
	Example: `  gocurl https://api.example.com
  gocurl -n 100 -c 10 https://api.example.com
  gocurl -o json https://api.example.com
  gocurl -o graph -n 100 -c 10 https://api.example.com
  gocurl -H "Authorization: Bearer token" https://api.example.com
  gocurl -L urls.txt -n 10 -c 5
  gocurl -L urls.txt --query-params params.txt -n 10 -c 5
  cat urls.txt | gocurl -L - -n 10`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHTTPTest,
}

func init() {
	// Set custom version template for detailed output
	rootCmd.SetVersionTemplate(GetVersionDetails() + "\n")

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table|json|prom|graph")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output (errors only)")

	// HTTP flags
	rootCmd.Flags().IntVarP(&requests, "requests", "n", 1, "Number of requests per URL")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 1, "Concurrent workers")
	rootCmd.Flags().IntVar(&warmupRequests, "warmup", 0, "Number of warmup requests to skip from metrics (per URL)")
	rootCmd.Flags().IntVar(&rps, "rps", 0, "Rate limit: requests per second (0 = unlimited)")
	rootCmd.Flags().StringVar(&rampUp, "ramp-up", "", "Gradually increase concurrency over duration (e.g., '30s', '1m')")
	rootCmd.Flags().StringVar(&exportCSV, "export-csv", "", "Export individual request data to CSV file")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom headers (repeatable)")
	rootCmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	rootCmd.Flags().StringVar(&data, "data", "", "Request body")
	rootCmd.Flags().StringVar(&timeout, "timeout", "30s", "Request timeout")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS verification")
	rootCmd.Flags().StringVarP(&urlListFile, "url-list", "L", "", "File containing URLs (one per line), use '-' for stdin")
	rootCmd.Flags().BoolVar(&useStdin, "stdin", false, "Read URLs from stdin")
	rootCmd.Flags().StringVar(&queryParamsFile, "query-params", "", "File containing query parameters (one per line) to append to each URL")

	// Response display flags
	rootCmd.Flags().BoolVarP(&includeHeaders, "include", "i", false, "Include response headers in output")
	rootCmd.Flags().BoolVarP(&headRequest, "head", "I", false, "Make HEAD request (show headers only)")
	rootCmd.Flags().BoolVar(&showBody, "show-body", false, "Show response body in output")
	rootCmd.Flags().BoolVar(&showErrorBody, "show-error", false, "Show response body for error responses (4xx, 5xx)")
	rootCmd.Flags().StringArrayVar(&captureHeaders, "capture-header", []string{}, "Capture specific response headers (repeatable, e.g., --capture-header Cache-Control)")
	rootCmd.Flags().StringVar(&rangeHeader, "range", "", "Request partial content (e.g., 'bytes=0-1023' or 'bytes=1024-')")

	// Performance analysis flags
	rootCmd.Flags().BoolVar(&enableStreaming, "streaming", false, "Enable detailed streaming metrics (chunk-level timing)")
	rootCmd.Flags().BoolVar(&expectStreaming, "expect-streaming", false, "Exit with error if streaming is not detected (implies --streaming)")
	rootCmd.Flags().StringVar(&stallThreshold, "stall-threshold", "500ms", "Duration threshold for detecting stalls in streaming")

	// Connection control flags
	rootCmd.Flags().StringArrayVar(&resolveHosts, "resolve", []string{}, "Resolve host:port to address (format: host:port:addr)")
	rootCmd.Flags().StringArrayVar(&connectToHosts, "connect-to", []string{}, "Connect to host:port instead (format: host1:port1:host2:port2)")

	// Scenario testing flags
	rootCmd.Flags().StringVarP(&scenarioFile, "scenario", "s", "", "Execute a multi-step scenario from YAML file")
}

func runHTTPTest(cmd *cobra.Command, args []string) error {
	if noColor {
		color.NoColor = true
	}

	// Show version in verbose mode (but not quiet mode)
	if verbose && !quiet {
		fmt.Fprintf(os.Stderr, "%s\n\n", GetVersionInfo())
	}

	// Handle scenario mode
	if scenarioFile != "" {
		return runScenario(scenarioFile)
	}

	// Handle HEAD request flag
	if headRequest {
		method = "HEAD"
		includeHeaders = true // Always show headers for HEAD requests
	}

	// --expect-streaming implies --streaming
	if expectStreaming {
		enableStreaming = true
	}

	var urls []string

	// Handle URL input
	if urlListFile != "" || useStdin {
		// Read from file or stdin
		reader := &app.URLReader{}
		var err error

		if urlListFile == "-" || useStdin {
			err = reader.ReadFromStdin()
		} else if urlListFile != "" {
			err = reader.ReadFromFile(urlListFile)
		}

		if err != nil {
			return err
		}

		urls = reader.GetURLs()
		if len(urls) == 0 {
			return fmt.Errorf("no URLs provided")
		}
	} else if len(args) > 0 {
		// Single URL from argument
		urls = []string{args[0]}
	} else {
		return fmt.Errorf("no URL provided (use a URL argument or -L flag)")
	}

	// Expand URLs with query parameters if provided
	if queryParamsFile != "" {
		queryParams, err := app.ReadQueryParamsFromFile(queryParamsFile)
		if err != nil {
			return fmt.Errorf("failed to read query params: %w", err)
		}

		if len(queryParams) > 0 {
			urls = app.ExpandURLsWithQueryParams(urls, queryParams)
			if !quiet {
				// Notices go to stderr so stdout stays a clean data channel
				// (e.g. `-o json | jq`).
				fmt.Fprintf(os.Stderr, "Expanded %d base URL(s) x %d query param variant(s) = %d total URL(s)\n",
					len(urls)/len(queryParams), len(queryParams), len(urls))
			}
		}
	}

	// Validate warmup requests
	if warmupRequests >= requests && requests > 1 {
		return fmt.Errorf("warmup requests (%d) must be less than total requests (%d)", warmupRequests, requests)
	}

	config := &app.Config{
		URLs:            urls,
		Method:          method,
		Headers:         headers,
		Data:            data,
		Requests:        requests,
		Concurrency:     concurrency,
		WarmupRequests:  warmupRequests,
		RPS:             rps,
		RampUp:          rampUp,
		ExportCSV:       exportCSV,
		Timeout:         timeout,
		Insecure:        insecure,
		OutputFormat:    outputFormat,
		Verbose:         verbose,
		Quiet:           quiet,
		IncludeHeaders:  includeHeaders,
		ShowBody:        showBody,
		ShowErrorBody:   showErrorBody,
		CaptureHeaders:  captureHeaders,
		RangeHeader:     rangeHeader,
		EnableStreaming: enableStreaming,
		ResolveHosts:    resolveHosts,
		ConnectToHosts:  connectToHosts,
		ExpectStreaming: expectStreaming,
		StallThreshold:  stallThreshold,

		// Version information
		Version:   version,
		Commit:    commit,
		BuildDate: date,
	}

	application := app.New(config)
	return application.Run()
}

func Execute() error {
	return rootCmd.Execute()
}

func runScenario(filename string) error {
	scenario, err := app.LoadAndExecuteScenario(filename, insecure, timeout)
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Printf("✓ Scenario '%s' completed successfully\n", scenario.Name)
		fmt.Printf("  Description: %s\n", scenario.Description)
		fmt.Printf("  Steps executed: %d\n\n", len(scenario.Steps))
	}

	return nil
}
