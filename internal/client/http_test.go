package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := &Config{
		Timeout:         30 * time.Second,
		Insecure:        false,
		MaxIdleConns:    100,
		MaxIdlePerHost:  10,
		DisableKeepAlive: false,
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.client == nil {
		t.Error("HTTP client should be initialized")
	}

	if client.config != config {
		t.Error("Config not properly set")
	}
}

func TestClientMeasureRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	config := &Config{
		Timeout:  5 * time.Second,
		Insecure: true,
	}

	client := NewClient(config)
	timing, err := client.MeasureRequest(server.URL, "GET", nil, nil)

	if err != nil {
		t.Fatalf("MeasureRequest failed: %v", err)
	}

	if timing == nil {
		t.Fatal("Timing should not be nil")
	}

	if timing.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", timing.StatusCode)
	}

	if timing.Total < Duration(10*time.Millisecond) {
		t.Errorf("Total time too short: %v", timing.Total)
	}

	if timing.ResponseSize != 13 {
		t.Errorf("Expected response size 13, got %d", timing.ResponseSize)
	}
}

func TestClientMeasureRequestWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Error("Custom header not received")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Timeout: 5 * time.Second,
	}

	headers := map[string]string{
		"X-Test-Header": "test-value",
	}

	client := NewClient(config)
	timing, err := client.MeasureRequest(server.URL, "GET", headers, nil)

	if err != nil {
		t.Fatalf("MeasureRequest failed: %v", err)
	}

	if timing.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", timing.StatusCode)
	}
}

func TestClientMeasureRequestWithBody(t *testing.T) {
	receivedBody := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &Config{
		Timeout: 5 * time.Second,
	}

	testBody := "test request body"
	body := strings.NewReader(testBody)

	client := NewClient(config)
	timing, err := client.MeasureRequest(server.URL, "POST", nil, body)

	if err != nil {
		t.Fatalf("MeasureRequest failed: %v", err)
	}

	if timing.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", timing.StatusCode)
	}

	if receivedBody != testBody {
		t.Errorf("Expected body %q, got %q", testBody, receivedBody)
	}
}

func TestClientMeasureRequestError(t *testing.T) {
	config := &Config{
		Timeout: 1 * time.Millisecond,
	}

	client := NewClient(config)
	timing, err := client.MeasureRequest("http://invalid-url-that-does-not-exist.local", "GET", nil, nil)

	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	if timing == nil {
		t.Fatal("Timing should be returned even on error")
	}

	if timing.Error == "" {
		t.Error("Error field should be populated")
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "empty",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:  "single header",
			input: []string{"Content-Type: application/json"},
			expected: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name: "multiple headers",
			input: []string{
				"Content-Type: application/json",
				"Authorization: Bearer token",
			},
			expected: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
		},
		{
			name:  "header with spaces",
			input: []string{"  X-Custom-Header  :  value with spaces  "},
			expected: map[string]string{
				"X-Custom-Header": "value with spaces",
			},
		},
		{
			name:     "invalid header",
			input:    []string{"Invalid Header Without Colon"},
			expected: map[string]string{},
		},
		{
			name:  "header with colon in value",
			input: []string{"X-URL: https://example.com"},
			expected: map[string]string{
				"X-URL": "https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseHeaders(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d headers, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Missing header %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("Header %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Timeout: 10 * time.Millisecond,
	}

	client := NewClient(config)
	_, err := client.MeasureRequest(server.URL, "GET", nil, nil)

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestClientUserAgent(t *testing.T) {
	receivedUA := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Timeout: 5 * time.Second,
	}

	client := NewClient(config)
	_, err := client.MeasureRequest(server.URL, "GET", nil, nil)

	if err != nil {
		t.Fatalf("MeasureRequest failed: %v", err)
	}

	if receivedUA != "gocurl/1.0" {
		t.Errorf("Expected User-Agent 'gocurl/1.0', got %q", receivedUA)
	}
}

func TestClientCustomUserAgent(t *testing.T) {
	receivedUA := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Timeout: 5 * time.Second,
	}

	headers := map[string]string{
		"User-Agent": "custom-agent/2.0",
	}

	client := NewClient(config)
	_, err := client.MeasureRequest(server.URL, "GET", headers, nil)

	if err != nil {
		t.Fatalf("MeasureRequest failed: %v", err)
	}

	if receivedUA != "custom-agent/2.0" {
		t.Errorf("Expected User-Agent 'custom-agent/2.0', got %q", receivedUA)
	}
}

func TestParseResolveHosts(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:     "empty",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:  "single valid",
			input: []string{"example.com:443:192.168.1.1"},
			expected: map[string]string{
				"example.com:443": "192.168.1.1",
			},
		},
		{
			name: "multiple valid",
			input: []string{
				"api.example.com:443:10.0.0.1",
				"cdn.example.com:80:10.0.0.2",
			},
			expected: map[string]string{
				"api.example.com:443": "10.0.0.1",
				"cdn.example.com:80":  "10.0.0.2",
			},
		},
		{
			name:  "with spaces",
			input: []string{"  example.com : 443 : 192.168.1.1  "},
			expected: map[string]string{
				"example.com:443": "192.168.1.1",
			},
		},
		{
			name:        "invalid format - missing port",
			input:       []string{"example.com:192.168.1.1"},
			expectError: true,
		},
		{
			name:  "with extra colons in addr",
			input: []string{"example.com:443:192.168.1.1:8080"},
			expected: map[string]string{
				"example.com:443": "192.168.1.1:8080",
			},
		},
		{
			name:        "invalid format - empty host",
			input:       []string{":443:192.168.1.1"},
			expectError: true,
		},
		{
			name:        "invalid format - empty port",
			input:       []string{"example.com::192.168.1.1"},
			expectError: true,
		},
		{
			name:        "invalid format - empty addr",
			input:       []string{"example.com:443:"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseResolveHosts(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d entries, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Missing entry %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("Entry %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestParseConnectToHosts(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:     "empty",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:  "single valid",
			input: []string{"example.com:443:backend.local:8443"},
			expected: map[string]string{
				"example.com:443": "backend.local:8443",
			},
		},
		{
			name: "multiple valid",
			input: []string{
				"api.example.com:443:backend1.local:443",
				"cdn.example.com:80:backend2.local:80",
			},
			expected: map[string]string{
				"api.example.com:443": "backend1.local:443",
				"cdn.example.com:80":  "backend2.local:80",
			},
		},
		{
			name:  "with spaces",
			input: []string{"  example.com : 443 : localhost : 8443  "},
			expected: map[string]string{
				"example.com:443": "localhost:8443",
			},
		},
		{
			name:        "invalid format - missing port2",
			input:       []string{"example.com:443:localhost"},
			expectError: true,
		},
		{
			name:  "with port in target",
			input: []string{"example.com:443:backend.internal:9443"},
			expected: map[string]string{
				"example.com:443": "backend.internal:9443",
			},
		},
		{
			name:        "invalid format - empty host1",
			input:       []string{":443:localhost:8443"},
			expectError: true,
		},
		{
			name:        "invalid format - empty port1",
			input:       []string{"example.com::localhost:8443"},
			expectError: true,
		},
		{
			name:        "invalid format - empty host2",
			input:       []string{"example.com:443::8443"},
			expectError: true,
		},
		{
			name:        "invalid format - empty port2",
			input:       []string{"example.com:443:localhost:"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseConnectToHosts(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d entries, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Missing entry %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("Entry %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}
