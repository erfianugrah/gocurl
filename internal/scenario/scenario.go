package scenario

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
)

// Scenario represents a multi-step test scenario
type Scenario struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
	Config      Config `yaml:"config,omitempty"`
}

// Step represents a single request in a scenario
type Step struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	Extract map[string]string `yaml:"extract,omitempty"` // JSONPath extractions
	Expect  *Expectation      `yaml:"expect,omitempty"`
}

// Expectation defines expected response conditions
type Expectation struct {
	StatusCode   int               `yaml:"status_code,omitempty"`
	StatusCodes  []int             `yaml:"status_codes,omitempty"` // Accept any of these
	Headers      map[string]string `yaml:"headers,omitempty"`
	BodyContains string            `yaml:"body_contains,omitempty"`
}

// Config contains scenario-level configuration
type Config struct {
	Timeout  string `yaml:"timeout,omitempty"`
	Insecure bool   `yaml:"insecure,omitempty"`
}

// Context holds runtime state for scenario execution
type Context struct {
	Variables map[string]string
	CookieJar *cookiejar.Jar
	Client    *http.Client
	Results   []StepResult
}

// StepResult contains the result of executing a step
type StepResult struct {
	StepName      string
	URL           string
	Method        string
	StatusCode    int
	Duration      time.Duration
	ResponseBody  string
	ResponseSize  int64
	Error         string
	ExtractedVars map[string]string
}

// LoadScenario loads a scenario from a YAML file
func LoadScenario(filename string) (*Scenario, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file: %w", err)
	}

	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to parse scenario YAML: %w", err)
	}

	// Set defaults
	for i := range scenario.Steps {
		if scenario.Steps[i].Method == "" {
			scenario.Steps[i].Method = "GET"
		}
	}

	return &scenario, nil
}

// NewContext creates a new execution context
func NewContext() (*Context, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &Context{
		Variables: make(map[string]string),
		CookieJar: jar,
		Results:   []StepResult{},
	}, nil
}

// substituteVariables replaces ${var} with values from context
func (c *Context) substituteVariables(text string) string {
	result := text
	for key, value := range c.Variables {
		placeholder := "${" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// ExecuteStep executes a single scenario step
func (c *Context) ExecuteStep(step Step) (*StepResult, error) {
	result := &StepResult{
		StepName:      step.Name,
		Method:        step.Method,
		ExtractedVars: make(map[string]string),
	}

	// Substitute variables in URL
	url := c.substituteVariables(step.URL)
	result.URL = url

	// Substitute variables in body
	body := c.substituteVariables(step.Body)

	// Create request
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(step.Method, url, bodyReader)
	if err != nil {
		result.Error = err.Error()
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers with variable substitution
	for key, value := range step.Headers {
		substitutedValue := c.substituteVariables(value)
		req.Header.Set(key, substitutedValue)
	}

	// Execute request
	start := time.Now()
	resp, err := c.Client.Do(req)
	duration := time.Since(start)
	result.Duration = duration

	if err != nil {
		result.Error = err.Error()
		return result, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		return result, err
	}

	result.StatusCode = resp.StatusCode
	result.ResponseBody = string(bodyBytes)
	result.ResponseSize = int64(len(bodyBytes))

	// Check expectations
	if step.Expect != nil {
		if err := c.checkExpectations(step.Expect, resp, string(bodyBytes)); err != nil {
			result.Error = fmt.Sprintf("expectation failed: %v", err)
			return result, err
		}
	}

	// Extract variables from response
	if len(step.Extract) > 0 && len(bodyBytes) > 0 {
		// Try to parse as JSON
		if gjson.Valid(string(bodyBytes)) {
			for varName, jsonPath := range step.Extract {
				value := gjson.Get(string(bodyBytes), jsonPath)
				if value.Exists() {
					extractedValue := value.String()
					c.Variables[varName] = extractedValue
					result.ExtractedVars[varName] = extractedValue
				}
			}
		}
	}

	return result, nil
}

// checkExpectations validates response against expectations
func (c *Context) checkExpectations(expect *Expectation, resp *http.Response, body string) error {
	// Check status code
	if expect.StatusCode != 0 {
		if resp.StatusCode != expect.StatusCode {
			return fmt.Errorf("expected status %d, got %d", expect.StatusCode, resp.StatusCode)
		}
	}

	// Check status codes (any of)
	if len(expect.StatusCodes) > 0 {
		found := false
		for _, code := range expect.StatusCodes {
			if resp.StatusCode == code {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected one of status codes %v, got %d", expect.StatusCodes, resp.StatusCode)
		}
	}

	// Check headers
	for key, expectedValue := range expect.Headers {
		actualValue := resp.Header.Get(key)
		if actualValue != expectedValue {
			return fmt.Errorf("expected header %s=%s, got %s", key, expectedValue, actualValue)
		}
	}

	// Check body contains
	if expect.BodyContains != "" {
		if !strings.Contains(body, expect.BodyContains) {
			return fmt.Errorf("response body does not contain '%s'", expect.BodyContains)
		}
	}

	return nil
}

// Execute runs all steps in the scenario
func (s *Scenario) Execute(ctx *Context) error {
	for i, step := range s.Steps {
		result, err := ctx.ExecuteStep(step)
		ctx.Results = append(ctx.Results, *result)

		if err != nil {
			return fmt.Errorf("step %d (%s) failed: %w", i+1, step.Name, err)
		}
	}

	return nil
}

// FormatResults returns a JSON representation of all step results
func (ctx *Context) FormatResults() (string, error) {
	data, err := json.MarshalIndent(ctx.Results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}
	return string(data), nil
}
