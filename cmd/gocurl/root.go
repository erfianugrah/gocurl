package main

import (
	"fmt"

	"github.com/erfi/gocurl/internal/app"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	outputFormat   string
	noColor        bool
	verbose        bool
	quiet          bool
	requests       int
	concurrency    int
	duration       string
	headers        []string
	method         string
	data           string
	timeout        string
	insecure       bool
	urlListFile    string
	useStdin       bool
	includeHeaders bool
	showBody       bool
	showErrorBody  bool
	headRequest    bool
	enableStreaming bool
	resolveHosts   []string
	connectToHosts []string
	expectStreaming bool
	stallThreshold  string
)

var rootCmd = &cobra.Command{
	Use:   "gocurl [flags] [url]",
	Short: "A Go-based performance measurement CLI tool that extends curl functionality",
	Long: `gocurl is a production-grade performance measurement tool that provides
rich metrics, multiple output formats, and load testing capabilities.

It measures detailed HTTP performance metrics including DNS lookup time,
TCP connection time, TLS handshake time, server processing time, and more.`,
	Example: `  gocurl https://api.example.com
  gocurl -n 100 -c 10 https://api.example.com
  gocurl -o json https://api.example.com
  gocurl -o graph -n 100 -c 10 https://api.example.com
  gocurl -H "Authorization: Bearer token" https://api.example.com
  gocurl -L urls.txt -n 10 -c 5
  cat urls.txt | gocurl -L - -n 10`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHTTPTest,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table|json|prom|graph")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output (errors only)")

	// HTTP flags
	rootCmd.Flags().IntVarP(&requests, "requests", "n", 1, "Number of requests per URL")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 1, "Concurrent workers")
	rootCmd.Flags().StringVarP(&duration, "duration", "d", "", "Test duration (e.g., 30s, 5m)")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom headers (repeatable)")
	rootCmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	rootCmd.Flags().StringVar(&data, "data", "", "Request body")
	rootCmd.Flags().StringVar(&timeout, "timeout", "30s", "Request timeout")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS verification")
	rootCmd.Flags().StringVarP(&urlListFile, "url-list", "L", "", "File containing URLs (one per line), use '-' for stdin")
	rootCmd.Flags().BoolVar(&useStdin, "stdin", false, "Read URLs from stdin")

	// Response display flags
	rootCmd.Flags().BoolVarP(&includeHeaders, "include", "i", false, "Include response headers in output")
	rootCmd.Flags().BoolVarP(&headRequest, "head", "I", false, "Make HEAD request (show headers only)")
	rootCmd.Flags().BoolVar(&showBody, "show-body", false, "Show response body in output")
	rootCmd.Flags().BoolVar(&showErrorBody, "show-error", false, "Show response body for error responses (4xx, 5xx)")

	// Performance analysis flags
	rootCmd.Flags().BoolVar(&enableStreaming, "streaming", false, "Enable detailed streaming metrics (chunk-level timing)")
	rootCmd.Flags().BoolVar(&expectStreaming, "expect-streaming", false, "Exit with error if streaming is not detected (implies --streaming)")
	rootCmd.Flags().StringVar(&stallThreshold, "stall-threshold", "500ms", "Duration threshold for detecting stalls in streaming")

	// Connection control flags
	rootCmd.Flags().StringArrayVar(&resolveHosts, "resolve", []string{}, "Resolve host:port to address (format: host:port:addr)")
	rootCmd.Flags().StringArrayVar(&connectToHosts, "connect-to", []string{}, "Connect to host:port instead (format: host1:port1:host2:port2)")
}

func runHTTPTest(cmd *cobra.Command, args []string) error {
	if noColor {
		color.NoColor = true
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

	config := &app.Config{
		URLs:            urls,
		Method:          method,
		Headers:         headers,
		Data:            data,
		Requests:        requests,
		Concurrency:     concurrency,
		Duration:        duration,
		Timeout:         timeout,
		Insecure:        insecure,
		OutputFormat:    outputFormat,
		Verbose:         verbose,
		Quiet:           quiet,
		IncludeHeaders:  includeHeaders,
		ShowBody:        showBody,
		ShowErrorBody:   showErrorBody,
		EnableStreaming: enableStreaming,
		ResolveHosts:    resolveHosts,
		ConnectToHosts:  connectToHosts,
		ExpectStreaming: expectStreaming,
		StallThreshold:  stallThreshold,
	}

	application := app.New(config)
	return application.Run()
}

func Execute() error {
	return rootCmd.Execute()
}
