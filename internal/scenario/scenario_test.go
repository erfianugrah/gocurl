package scenario

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLoadScenario(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantError bool
		wantSteps int
	}{
		{
			name: "valid scenario",
			yaml: `name: "Test Scenario"
description: "Test description"
steps:
  - name: "Step 1"
    url: "https://example.com"
    method: GET
  - name: "Step 2"
    url: "https://example.com/api"
    method: POST
`,
			wantError: false,
			wantSteps: 2,
		},
		{
			name: "scenario with defaults",
			yaml: `name: "Test"
steps:
  - name: "Step 1"
    url: "https://example.com"
`,
			wantError: false,
			wantSteps: 1,
		},
		{
			name:      "invalid YAML",
			yaml:      `invalid: [yaml syntax`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp("", "scenario-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.yaml)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			// Load scenario
			scenario, err := LoadScenario(tmpfile.Name())

			if (err != nil) != tt.wantError {
				t.Errorf("LoadScenario() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				if len(scenario.Steps) != tt.wantSteps {
					t.Errorf("LoadScenario() got %d steps, want %d", len(scenario.Steps), tt.wantSteps)
				}

				// Check that default method is set
				for i, step := range scenario.Steps {
					if step.Method == "" {
						t.Errorf("Step %d method not set to default", i)
					}
				}
			}
		})
	}
}

func TestLoadScenarioInvalidFile(t *testing.T) {
	_, err := LoadScenario("/path/that/does/not/exist.yaml")
	if err == nil {
		t.Error("LoadScenario() with invalid path should return error")
	}
}

func TestNewContext(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}

	if ctx.Variables == nil {
		t.Error("NewContext() Variables map is nil")
	}

	if ctx.CookieJar == nil {
		t.Error("NewContext() CookieJar is nil")
	}

	if ctx.Results == nil {
		t.Error("NewContext() Results slice is nil")
	}
}

func TestContextSubstituteVariables(t *testing.T) {
	ctx := &Context{
		Variables: map[string]string{
			"host":  "example.com",
			"port":  "8080",
			"token": "abc123",
		},
	}

	tests := []struct {
		input string
		want  string
	}{
		{"https://${host}", "https://example.com"},
		{"https://${host}:${port}/api", "https://example.com:8080/api"},
		{"Bearer ${token}", "Bearer abc123"},
		{"no variables here", "no variables here"},
		{"${host} and ${port} and ${token}", "example.com and 8080 and abc123"},
		{"${unknown}", "${unknown}"}, // Unknown variables left as-is
	}

	for _, tt := range tests {
		got := ctx.substituteVariables(tt.input)
		if got != tt.want {
			t.Errorf("substituteVariables(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCheckExpectations(t *testing.T) {
	tests := []struct {
		name        string
		expectation *Expectation
		statusCode  int
		headers     map[string]string
		body        string
		wantError   bool
	}{
		{
			name: "status code match",
			expectation: &Expectation{
				StatusCode: 200,
			},
			statusCode: 200,
			wantError:  false,
		},
		{
			name: "status code mismatch",
			expectation: &Expectation{
				StatusCode: 200,
			},
			statusCode: 404,
			wantError:  true,
		},
		{
			name: "status codes any match",
			expectation: &Expectation{
				StatusCodes: []int{200, 201, 202},
			},
			statusCode: 201,
			wantError:  false,
		},
		{
			name: "status codes no match",
			expectation: &Expectation{
				StatusCodes: []int{200, 201, 202},
			},
			statusCode: 404,
			wantError:  true,
		},
		{
			name: "header match",
			expectation: &Expectation{
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			statusCode: 200,
			wantError:  false,
		},
		{
			name: "body contains match",
			expectation: &Expectation{
				BodyContains: "success",
			},
			body:       `{"status": "success"}`,
			statusCode: 200,
			wantError:  false,
		},
		{
			name: "body contains no match",
			expectation: &Expectation{
				BodyContains: "error",
			},
			body:       `{"status": "success"}`,
			statusCode: 200,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}

			// Create mock response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     make(http.Header),
			}

			for key, value := range tt.headers {
				resp.Header.Set(key, value)
			}

			err := ctx.checkExpectations(tt.expectation, resp, tt.body)

			if (err != nil) != tt.wantError {
				t.Errorf("checkExpectations() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestExecuteStep(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "success",
				"token":  "test-token-123",
			})
		case "/echo":
			w.WriteHeader(http.StatusOK)
			w.Header().Set("X-Custom-Header", r.Header.Get("X-Test-Header"))
			w.Write([]byte("Echo"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name      string
		step      Step
		variables map[string]string
		wantError bool
		checkFunc func(*testing.T, *StepResult)
	}{
		{
			name: "successful GET request",
			step: Step{
				Name:   "Test Step",
				URL:    server.URL + "/success",
				Method: "GET",
			},
			wantError: false,
			checkFunc: func(t *testing.T, result *StepResult) {
				if result.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", result.StatusCode)
				}
				if !strings.Contains(result.ResponseBody, "success") {
					t.Error("Response body should contain 'success'")
				}
			},
		},
		{
			name: "request with variable substitution",
			step: Step{
				Name:   "Var Test",
				URL:    "${baseURL}/success",
				Method: "GET",
			},
			variables: map[string]string{
				"baseURL": server.URL,
			},
			wantError: false,
			checkFunc: func(t *testing.T, result *StepResult) {
				if result.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", result.StatusCode)
				}
			},
		},
		{
			name: "request with headers",
			step: Step{
				Name:   "Header Test",
				URL:    server.URL + "/echo",
				Method: "GET",
				Headers: map[string]string{
					"X-Test-Header": "test-value",
				},
			},
			wantError: false,
			checkFunc: func(t *testing.T, result *StepResult) {
				if result.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", result.StatusCode)
				}
			},
		},
		{
			name: "request with JSON extraction",
			step: Step{
				Name:   "Extract Test",
				URL:    server.URL + "/success",
				Method: "GET",
				Extract: map[string]string{
					"authToken": "token",
					"status":    "status",
				},
			},
			wantError: false,
			checkFunc: func(t *testing.T, result *StepResult) {
				if len(result.ExtractedVars) == 0 {
					t.Error("No variables were extracted")
				}
				if result.ExtractedVars["authToken"] != "test-token-123" {
					t.Errorf("authToken = %q, want %q", result.ExtractedVars["authToken"], "test-token-123")
				}
			},
		},
		{
			name: "request with expectation",
			step: Step{
				Name:   "Expect Test",
				URL:    server.URL + "/success",
				Method: "GET",
				Expect: &Expectation{
					StatusCode:   200,
					BodyContains: "success",
				},
			},
			wantError: false,
		},
		{
			name: "request with failed expectation",
			step: Step{
				Name:   "Failed Expect",
				URL:    server.URL + "/success",
				Method: "GET",
				Expect: &Expectation{
					StatusCode: 404,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewContext()
			if err != nil {
				t.Fatal(err)
			}

			ctx.Client = server.Client()
			if tt.variables != nil {
				ctx.Variables = tt.variables
			}

			result, err := ctx.ExecuteStep(tt.step)

			if (err != nil) != tt.wantError {
				t.Errorf("ExecuteStep() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestScenarioExecute(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	scenario := &Scenario{
		Name:        "Test Scenario",
		Description: "Multi-step test",
		Steps: []Step{
			{
				Name:   "Step 1",
				URL:    server.URL + "/step1",
				Method: "GET",
			},
			{
				Name:   "Step 2",
				URL:    server.URL + "/step2",
				Method: "GET",
			},
		},
	}

	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	ctx.Client = server.Client()

	err = scenario.Execute(ctx)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if len(ctx.Results) != 2 {
		t.Errorf("Execute() produced %d results, want 2", len(ctx.Results))
	}
}

func TestFormatResults(t *testing.T) {
	ctx := &Context{
		Results: []StepResult{
			{
				StepName:   "Test Step",
				URL:        "https://example.com",
				Method:     "GET",
				StatusCode: 200,
			},
		},
	}

	jsonStr, err := ctx.FormatResults()
	if err != nil {
		t.Fatalf("FormatResults() error = %v", err)
	}

	// Verify it's valid JSON
	var results []StepResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		t.Errorf("FormatResults() produced invalid JSON: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Unmarshaled %d results, want 1", len(results))
	}

	if results[0].StepName != "Test Step" {
		t.Errorf("StepName = %q, want %q", results[0].StepName, "Test Step")
	}
}
