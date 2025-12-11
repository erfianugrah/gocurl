package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSelectiveHeaderCapture tests Feature 14: Selective Header Capture
func TestSelectiveHeaderCapture(t *testing.T) {
	// Create test server that returns specific headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("ETag", "abc123")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	tests := []struct {
		name           string
		captureHeaders []string
		includeHeaders bool
		checkPresent   []string
		checkAbsent    []string
	}{
		{
			name:           "capture specific headers",
			captureHeaders: []string{"Cache-Control", "ETag"},
			includeHeaders: false,
			checkPresent:   []string{"Cache-Control", "ETag"},
			checkAbsent:    []string{"Content-Type", "X-Custom-Header"},
		},
		{
			name:           "capture single header",
			captureHeaders: []string{"X-Custom-Header"},
			includeHeaders: false,
			checkPresent:   []string{"X-Custom-Header"},
			checkAbsent:    []string{"Cache-Control", "ETag"},
		},
		{
			name:           "capture all headers with -i flag",
			captureHeaders: []string{},
			includeHeaders: true,
			checkPresent:   []string{"Cache-Control", "Content-Type", "X-Custom-Header"},
			checkAbsent:    []string{},
		},
		{
			name:           "no header capture",
			captureHeaders: []string{},
			includeHeaders: false,
			checkPresent:   []string{},
			checkAbsent:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Timeout:        5 * time.Second,
				CaptureHeaders: tt.captureHeaders,
				IncludeHeaders: tt.includeHeaders,
			}

			client := NewClient(config)
			timing, err := client.MeasureRequest(server.URL, "GET", nil, nil)

			if err != nil {
				t.Fatalf("MeasureRequest failed: %v", err)
			}

			// Check that expected headers are present
			for _, header := range tt.checkPresent {
				if _, exists := timing.ResponseHeaders[header]; !exists {
					t.Errorf("Expected header %q not found in ResponseHeaders", header)
				}
			}

			// Check that unwanted headers are absent
			for _, header := range tt.checkAbsent {
				if _, exists := timing.ResponseHeaders[header]; exists {
					t.Errorf("Unexpected header %q found in ResponseHeaders", header)
				}
			}

			// Verify ResponseHeaders is nil when no capture is configured
			if len(tt.captureHeaders) == 0 && !tt.includeHeaders {
				if timing.ResponseHeaders != nil {
					t.Error("ResponseHeaders should be nil when no capture is configured")
				}
			}
		})
	}
}

// TestEffectiveURLTracking tests Feature 15: Effective URL Tracking
func TestEffectiveURLTracking(t *testing.T) {
	// Create test server with redirect chain
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			redirectCount++
			http.Redirect(w, r, "/middle", http.StatusFound)
		case "/middle":
			redirectCount++
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/final":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Final destination"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	tests := []struct {
		name          string
		startPath     string
		wantRedirects bool
	}{
		{
			name:          "multiple redirects",
			startPath:     "/start",
			wantRedirects: true,
		},
		{
			name:          "no redirects",
			startPath:     "/final",
			wantRedirects: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redirectCount = 0
			config := &Config{
				Timeout: 5 * time.Second,
			}

			client := NewClient(config)
			timing, err := client.MeasureRequest(server.URL+tt.startPath, "GET", nil, nil)

			if err != nil {
				t.Fatalf("MeasureRequest failed: %v", err)
			}

			if tt.wantRedirects {
				// Check that EffectiveURL is different from original URL
				if timing.EffectiveURL == "" {
					t.Error("EffectiveURL should be set after redirect")
				}

				expectedFinal := server.URL + "/final"
				if timing.EffectiveURL != expectedFinal {
					t.Errorf("EffectiveURL = %q, want %q", timing.EffectiveURL, expectedFinal)
				}

				if timing.RedirectCount == 0 {
					t.Error("RedirectCount should be > 0 after redirects")
				}
			} else {
				// No redirect - EffectiveURL should be empty
				if timing.EffectiveURL != "" {
					t.Errorf("EffectiveURL should be empty for non-redirected request, got %q", timing.EffectiveURL)
				}

				if timing.RedirectCount != 0 {
					t.Errorf("RedirectCount = %d, want 0", timing.RedirectCount)
				}
			}
		})
	}
}

// TestRangeRequestSupport tests Feature 16: Range Request Support
func TestRangeRequestSupport(t *testing.T) {
	// Create test server that supports range requests
	testContent := []byte("0123456789abcdefghijklmnopqrstuvwxyz")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		if rangeHeader == "" {
			// No range request - return full content
			w.WriteHeader(http.StatusOK)
			w.Write(testContent)
			return
		}

		// Simple range response (just verify header was sent)
		w.Header().Set("Content-Range", "bytes 0-9/36")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(testContent[0:10])
	}))
	defer server.Close()

	tests := []struct {
		name           string
		rangeHeader    string
		wantStatusCode int
		checkContent   bool
	}{
		{
			name:           "range request bytes 0-9",
			rangeHeader:    "bytes=0-9",
			wantStatusCode: http.StatusPartialContent,
			checkContent:   true,
		},
		{
			name:           "range request bytes 10-19",
			rangeHeader:    "bytes=10-19",
			wantStatusCode: http.StatusPartialContent,
			checkContent:   true,
		},
		{
			name:           "no range request",
			rangeHeader:    "",
			wantStatusCode: http.StatusOK,
			checkContent:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Timeout:     5 * time.Second,
				RangeHeader: tt.rangeHeader,
			}

			client := NewClient(config)
			timing, err := client.MeasureRequest(server.URL, "GET", nil, nil)

			if err != nil {
				t.Fatalf("MeasureRequest failed: %v", err)
			}

			if timing.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", timing.StatusCode, tt.wantStatusCode)
			}

			if tt.checkContent && tt.wantStatusCode == http.StatusPartialContent {
				if timing.ContentRange == "" {
					t.Error("ContentRange should be set for partial content response")
				}
			}
		})
	}
}

// TestRangeRequestWithCaptureHeaders tests combining Features 14 and 16
func TestRangeRequestWithCaptureHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			w.Header().Set("Content-Range", "bytes 0-999/10000")
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(make([]byte, 1000))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &Config{
		Timeout:        5 * time.Second,
		RangeHeader:    "bytes=0-999",
		CaptureHeaders: []string{"Content-Range", "Accept-Ranges"},
	}

	client := NewClient(config)
	timing, err := client.MeasureRequest(server.URL, "GET", nil, nil)

	if err != nil {
		t.Fatalf("MeasureRequest failed: %v", err)
	}

	// Verify range request worked
	if timing.StatusCode != http.StatusPartialContent {
		t.Errorf("StatusCode = %d, want %d", timing.StatusCode, http.StatusPartialContent)
	}

	// Verify headers were captured
	if _, exists := timing.ResponseHeaders["Content-Range"]; !exists {
		t.Error("Content-Range header should be captured")
	}

	if _, exists := timing.ResponseHeaders["Accept-Ranges"]; !exists {
		t.Error("Accept-Ranges header should be captured")
	}

	// Verify ContentRange field is set
	if timing.ContentRange == "" {
		t.Error("ContentRange field should be set")
	}
}
