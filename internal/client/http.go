package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

// HTTPClient interface defines the contract for HTTP operations
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client wraps the standard HTTP client with performance measurement capabilities
type Client struct {
	client  *http.Client
	config  *Config
}

// Config contains configuration for the HTTP client
type Config struct {
	Timeout          time.Duration
	Insecure         bool
	MaxIdleConns     int
	MaxIdlePerHost   int
	DisableKeepAlive bool
	IncludeHeaders   bool
	ShowBody         bool
	ShowErrorBody    bool
	ResolveMap       map[string]string // "host:port" -> "ip"
	ConnectToMap     map[string]string // "host:port" -> "newhost:newport"
	StallThreshold   time.Duration     // Threshold for detecting stalls
}

// NewClient creates a new HTTP client with the specified configuration
func NewClient(config *Config) *Client {
	// Create default dialer
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdlePerHost,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   config.DisableKeepAlive,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.Insecure,
		},
	}

	// Set up custom DialContext if --resolve or --connect-to are used
	if len(config.ConnectToMap) > 0 || len(config.ResolveMap) > 0 {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Check --connect-to mappings first
			if newAddr, ok := config.ConnectToMap[addr]; ok {
				// Connect to different host:port
				return dialer.DialContext(ctx, network, newAddr)
			}

			// Check --resolve mappings
			if ip, ok := config.ResolveMap[addr]; ok {
				// Extract port from addr
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to parse address %s: %w", addr, err)
				}
				_ = host // host is replaced with ip from resolve map

				// Connect to resolved IP with original port
				resolvedAddr := net.JoinHostPort(ip, port)
				return dialer.DialContext(ctx, network, resolvedAddr)
			}

			// No mapping found, use default dialer
			return dialer.DialContext(ctx, network, addr)
		}
	}

	// Enable HTTP/2 support
	http2.ConfigureTransport(transport)

	return &Client{
		client: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		config: config,
	}
}

// Do executes an HTTP request with timing measurement
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// MeasureRequest executes a single HTTP request and captures detailed timing information
func (c *Client) MeasureRequest(url, method string, headers map[string]string, body io.Reader) (*TimingBreakdown, error) {
	tracer := NewTracer()

	// Create request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "gocurl/1.0")
	}

	// Attach the tracer to the request context
	ctx := httptrace.WithClientTrace(req.Context(), tracer.ClientTrace())
	req = req.WithContext(ctx)

	// Start timing and execute request
	tracer.Start()
	resp, err := c.client.Do(req)
	if err != nil {
		tracer.End()
		timing := tracer.Timing()
		timing.Error = err.Error()
		return timing, err
	}
	defer resp.Body.Close()

	// Capture response headers if requested
	if c.config.IncludeHeaders {
		timing := tracer.Timing()
		timing.ResponseHeaders = make(map[string]string)
		for key, values := range resp.Header {
			// Join multiple values with comma (per HTTP spec)
			timing.ResponseHeaders[key] = values[0]
			if len(values) > 1 {
				for i := 1; i < len(values); i++ {
					timing.ResponseHeaders[key] += ", " + values[i]
				}
			}
		}
	}

	// Read the response body
	var written int64
	var bodyBytes []byte

	shouldCaptureBody := c.config.ShowBody || (c.config.ShowErrorBody && resp.StatusCode >= 400)

	if shouldCaptureBody {
		// Read body into memory
		bodyBytes, err = io.ReadAll(resp.Body)
		written = int64(len(bodyBytes))
	} else {
		// Discard body
		written, err = io.Copy(io.Discard, resp.Body)
	}

	tracer.End()

	// Populate response information
	timing := tracer.Timing()
	timing.StatusCode = resp.StatusCode
	timing.ContentLength = resp.ContentLength
	timing.ResponseSize = written

	if shouldCaptureBody && len(bodyBytes) > 0 {
		timing.ResponseBody = string(bodyBytes)
	}

	if err != nil {
		timing.Error = err.Error()
	}

	return timing, nil
}

// ParseHeaders converts a slice of "key: value" strings into a map
func ParseHeaders(headerSlice []string) map[string]string {
	headers := make(map[string]string)
	for _, h := range headerSlice {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
		}
	}
	return headers
}

// ParseResolveHosts converts --resolve format (host:port:addr) into a map
func ParseResolveHosts(resolveSlice []string) (map[string]string, error) {
	resolveMap := make(map[string]string)
	for _, r := range resolveSlice {
		parts := strings.SplitN(r, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid --resolve format '%s': expected host:port:addr", r)
		}
		host := strings.TrimSpace(parts[0])
		port := strings.TrimSpace(parts[1])
		addr := strings.TrimSpace(parts[2])

		if host == "" || port == "" || addr == "" {
			return nil, fmt.Errorf("invalid --resolve format '%s': host, port, and addr cannot be empty", r)
		}

		key := host + ":" + port
		resolveMap[key] = addr
	}
	return resolveMap, nil
}

// ParseConnectToHosts converts --connect-to format (host1:port1:host2:port2) into a map
func ParseConnectToHosts(connectToSlice []string) (map[string]string, error) {
	connectMap := make(map[string]string)
	for _, c := range connectToSlice {
		parts := strings.SplitN(c, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid --connect-to format '%s': expected host1:port1:host2:port2", c)
		}
		host1 := strings.TrimSpace(parts[0])
		port1 := strings.TrimSpace(parts[1])
		host2 := strings.TrimSpace(parts[2])
		port2 := strings.TrimSpace(parts[3])

		if host1 == "" || port1 == "" || host2 == "" || port2 == "" {
			return nil, fmt.Errorf("invalid --connect-to format '%s': all fields must be non-empty", c)
		}

		key := host1 + ":" + port1
		value := host2 + ":" + port2
		connectMap[key] = value
	}
	return connectMap, nil
}
