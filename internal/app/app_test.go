package app

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/scenario"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		check  func(*testing.T, *App)
	}{
		{
			name: "basic config",
			config: &Config{
				URLs:        []string{"https://example.com"},
				Method:      "GET",
				Timeout:     "30s",
				Requests:    1,
				Concurrency: 1,
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
				if app.config == nil {
					t.Error("config is nil")
				}
				if app.client == nil {
					t.Error("client is nil")
				}
				if app.collector == nil {
					t.Error("collector is nil")
				}
				if app.formatter == nil {
					t.Error("formatter is nil")
				}
			},
		},
		{
			name: "with invalid timeout",
			config: &Config{
				URLs:    []string{"https://example.com"},
				Timeout: "invalid",
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
				// Should fall back to default timeout
			},
		},
		{
			name: "with capture headers",
			config: &Config{
				URLs:           []string{"https://example.com"},
				CaptureHeaders: []string{"Cache-Control", "ETag"},
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
			},
		},
		{
			name: "with range header",
			config: &Config{
				URLs:        []string{"https://example.com"},
				RangeHeader: "bytes=0-999",
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
			},
		},
		{
			name: "with verbose",
			config: &Config{
				URLs:    []string{"https://example.com"},
				Verbose: true,
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
			},
		},
		{
			name: "json output format",
			config: &Config{
				URLs:         []string{"https://example.com"},
				OutputFormat: "json",
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
				if app.formatter == nil {
					t.Error("formatter should not be nil for json format")
				}
			},
		},
		{
			name: "graph output format",
			config: &Config{
				URLs:         []string{"https://example.com"},
				OutputFormat: "graph",
				Requests:     10,
			},
			check: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("New() returned nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(tt.config)
			tt.check(t, app)
		})
	}
}

func TestValidateStreaming(t *testing.T) {
	tests := []struct {
		name      string
		metrics   *client.StreamMetrics
		quiet     bool
		wantError bool
	}{
		{
			name: "valid streaming",
			metrics: &client.StreamMetrics{
				TotalChunks: 10,
				BufferingAnalysis: &client.BufferingAnalysis{
					BufferingDetected: false,
					ChunkPattern:      "steady",
					ChunkTimingCV:     0.15,
				},
				StreamingInfo: &client.StreamingInfo{
					IsStreamingLikely: true,
				},
			},
			quiet:     true,
			wantError: false,
		},
		{
			name: "buffering detected",
			metrics: &client.StreamMetrics{
				TotalChunks: 5,
				BufferingAnalysis: &client.BufferingAnalysis{
					BufferingDetected: true,
					ChunkPattern:      "bursty",
					ChunkTimingCV:     0.8,
				},
			},
			quiet:     true,
			wantError: true,
		},
		{
			name: "no buffering analysis",
			metrics: &client.StreamMetrics{
				TotalChunks:       10,
				BufferingAnalysis: nil,
			},
			quiet:     true,
			wantError: true,
		},
		{
			name: "not streaming likely",
			metrics: &client.StreamMetrics{
				TotalChunks: 10,
				BufferingAnalysis: &client.BufferingAnalysis{
					BufferingDetected: false,
					ChunkPattern:      "steady",
					ChunkTimingCV:     0.2,
				},
				StreamingInfo: &client.StreamingInfo{
					IsStreamingLikely: false,
				},
			},
			quiet:     true,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				config: &Config{
					Quiet: tt.quiet,
				},
			}

			err := app.validateStreaming(tt.metrics)

			if (err != nil) != tt.wantError {
				t.Errorf("validateStreaming() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestLoadAndExecuteScenario(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","token":"test-token-123"}`))
	}))
	defer server.Close()

	tests := []struct {
		name         string
		scenarioYAML string
		insecure     bool
		timeout      string
		wantError    bool
	}{
		{
			name: "valid scenario",
			scenarioYAML: `name: "Test Scenario"
description: "Test description"
steps:
  - name: "Step 1"
    url: "` + server.URL + `"
    method: GET
    expect:
      status_code: 200
  - name: "Step 2"
    url: "` + server.URL + `"
    method: GET
    extract:
      authToken: "token"
`,
			insecure:  false,
			timeout:   "10s",
			wantError: false,
		},
		{
			name: "scenario with timeout in config",
			scenarioYAML: `name: "Test Scenario"
config:
  timeout: "5s"
steps:
  - name: "Step 1"
    url: "` + server.URL + `"
    method: GET
`,
			insecure:  false,
			timeout:   "30s",
			wantError: false,
		},
		{
			name:         "invalid file",
			scenarioYAML: "",
			insecure:     false,
			timeout:      "10s",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpfile *os.File
			var err error

			if tt.scenarioYAML != "" {
				// Create temp scenario file
				tmpfile, err = os.CreateTemp("", "scenario-*.yaml")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(tmpfile.Name())

				if _, err := tmpfile.Write([]byte(tt.scenarioYAML)); err != nil {
					t.Fatal(err)
				}
				tmpfile.Close()
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			var filename string
			if tmpfile != nil {
				filename = tmpfile.Name()
			} else {
				filename = "/nonexistent/file.yaml"
			}

			sc, err := LoadAndExecuteScenario(filename, tt.insecure, tt.timeout)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			io.Copy(io.Discard, r)

			if (err != nil) != tt.wantError {
				t.Errorf("LoadAndExecuteScenario() error = %v, wantError %v", err, tt.wantError)
			}

			if err == nil && sc == nil {
				t.Error("LoadAndExecuteScenario() returned nil scenario without error")
			}

			if err == nil && sc != nil {
				if sc.Name == "" {
					t.Error("Scenario name is empty")
				}
			}
		})
	}
}

func TestPrintScenarioResults(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *scenario.Context
		contains []string
	}{
		{
			name: "single successful step",
			ctx: &scenario.Context{
				Results: []scenario.StepResult{
					{
						StepName:     "Test Step",
						URL:          "https://example.com",
						Method:       "GET",
						StatusCode:   200,
						Duration:     100 * time.Millisecond,
						ResponseSize: 1024,
						Error:        "",
					},
				},
			},
			contains: []string{"Test Step", "200", "100ms", "1024 bytes"},
		},
		{
			name: "step with error",
			ctx: &scenario.Context{
				Results: []scenario.StepResult{
					{
						StepName:   "Failed Step",
						URL:        "https://example.com",
						Method:     "GET",
						StatusCode: 500,
						Duration:   50 * time.Millisecond,
						Error:      "connection timeout",
					},
				},
			},
			contains: []string{"Failed Step", "500", "connection timeout"},
		},
		{
			name: "step with extracted variables",
			ctx: &scenario.Context{
				Results: []scenario.StepResult{
					{
						StepName:   "Extract Step",
						URL:        "https://example.com",
						Method:     "GET",
						StatusCode: 200,
						Duration:   100 * time.Millisecond,
						ExtractedVars: map[string]string{
							"token": "abc123",
							"id":    "42",
						},
					},
				},
			},
			contains: []string{"Extract Step", "token", "abc123", "id", "42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printScenarioResults(tt.ctx)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check for expected strings
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestNewWithResolveAndConnectTo(t *testing.T) {
	tests := []struct {
		name         string
		resolveHosts []string
		connectTo    []string
		wantWarning  bool
	}{
		{
			name:         "valid resolve mapping",
			resolveHosts: []string{"example.com:443:93.184.216.34"},
			connectTo:    []string{},
			wantWarning:  false,
		},
		{
			name:         "invalid resolve format",
			resolveHosts: []string{"invalid-format"},
			connectTo:    []string{},
			wantWarning:  true,
		},
		{
			name:         "valid connect-to mapping",
			resolveHosts: []string{},
			connectTo:    []string{"example.com:443:localhost:8080"},
			wantWarning:  false,
		},
		{
			name:         "invalid connect-to format",
			resolveHosts: []string{},
			connectTo:    []string{"invalid"},
			wantWarning:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			config := &Config{
				URLs:           []string{"https://example.com"},
				ResolveHosts:   tt.resolveHosts,
				ConnectToHosts: tt.connectTo,
			}

			app := New(config)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			io.Copy(&buf, r)
			stderr := buf.String()

			if tt.wantWarning && !strings.Contains(stderr, "Warning") {
				t.Error("Expected warning in stderr but got none")
			}

			if app == nil {
				t.Fatal("New() returned nil")
			}
		})
	}
}

func TestNewWithStallThreshold(t *testing.T) {
	tests := []struct {
		name           string
		stallThreshold string
		wantDefault    bool
	}{
		{
			name:           "valid threshold",
			stallThreshold: "1s",
			wantDefault:    false,
		},
		{
			name:           "invalid threshold",
			stallThreshold: "invalid",
			wantDefault:    true,
		},
		{
			name:           "empty threshold",
			stallThreshold: "",
			wantDefault:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				URLs:           []string{"https://example.com"},
				StallThreshold: tt.stallThreshold,
			}

			app := New(config)

			if app == nil {
				t.Fatal("New() returned nil")
			}
		})
	}
}
