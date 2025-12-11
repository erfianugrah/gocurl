package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name  string
		check func(*testing.T)
	}{
		{
			name: "command exists",
			check: func(t *testing.T) {
				if rootCmd == nil {
					t.Fatal("rootCmd is nil")
				}
			},
		},
		{
			name: "has correct use",
			check: func(t *testing.T) {
				if !strings.Contains(rootCmd.Use, "gocurl") {
					t.Errorf("Use = %q, should contain 'gocurl'", rootCmd.Use)
				}
			},
		},
		{
			name: "has short description",
			check: func(t *testing.T) {
				if rootCmd.Short == "" {
					t.Error("Short description is empty")
				}
			},
		},
		{
			name: "has long description",
			check: func(t *testing.T) {
				if rootCmd.Long == "" {
					t.Error("Long description is empty")
				}
			},
		},
		{
			name: "has examples",
			check: func(t *testing.T) {
				if rootCmd.Example == "" {
					t.Error("Example is empty")
				}
			},
		},
		{
			name: "accepts maximum one argument",
			check: func(t *testing.T) {
				if rootCmd.Args == nil {
					t.Error("Args validator is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t)
		})
	}
}

func TestRootCommandFlags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		wantFlag     bool
		isPersistent bool
	}{
		{"output flag", "output", "o", true, true},
		{"verbose flag", "verbose", "v", true, true},
		{"quiet flag", "quiet", "q", true, true},
		{"requests flag", "requests", "n", true, false},
		{"concurrency flag", "concurrency", "c", true, false},
		{"header flag", "header", "H", true, false},
		{"method flag", "method", "X", true, false},
		{"timeout flag", "timeout", "", true, false}, // No shorthand
		{"insecure flag", "insecure", "k", true, false},
		{"warmup flag", "warmup", "", true, false},
		{"rps flag", "rps", "", true, false},
		{"ramp-up flag", "ramp-up", "", true, false},
		{"export-csv flag", "export-csv", "", true, false},
		{"capture-header flag", "capture-header", "", true, false},
		{"range flag", "range", "", true, false},
		{"scenario flag", "scenario", "s", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flag *pflag.Flag
			if tt.isPersistent {
				flag = rootCmd.PersistentFlags().Lookup(tt.flagName)
			} else {
				flag = rootCmd.Flags().Lookup(tt.flagName)
			}
			if tt.wantFlag && flag == nil {
				t.Errorf("Flag %q not found", tt.flagName)
				return
			}
			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestRootCommandFlagDefaults(t *testing.T) {
	// Reset flags to defaults before checking
	rootCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Changed = false
	})
	rootCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		flag.Changed = false
	})

	tests := []struct {
		name         string
		flagName     string
		defaultValue string
		isPersistent bool
	}{
		{"output default", "output", "table", true},
		{"method default", "method", "GET", false},
		{"requests default", "requests", "1", false},
		{"concurrency default", "concurrency", "1", false},
		{"warmup default", "warmup", "0", false},
		{"rps default", "rps", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flag *pflag.Flag
			if tt.isPersistent {
				flag = rootCmd.PersistentFlags().Lookup(tt.flagName)
			} else {
				flag = rootCmd.Flags().Lookup(tt.flagName)
			}
			if flag == nil {
				t.Fatalf("Flag %q not found", tt.flagName)
			}
			if flag.DefValue != tt.defaultValue {
				t.Errorf("Flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestExecuteWithTestServer(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "simple GET request",
			args:      []string{"-q", server.URL},
			wantError: false,
		},
		{
			name:      "with custom method",
			args:      []string{"-q", "-X", "POST", server.URL},
			wantError: false,
		},
		{
			name:      "with multiple requests",
			args:      []string{"-q", "-n", "2", server.URL},
			wantError: false,
		},
		{
			name:      "with json output",
			args:      []string{"-q", "-o", "json", server.URL},
			wantError: false,
		},
		{
			name:      "with header",
			args:      []string{"-q", "-H", "X-Test: value", server.URL},
			wantError: false,
		},
		{
			name:      "with capture header",
			args:      []string{"-q", "--capture-header", "Content-Type", server.URL},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command for each test
			cmd := &cobra.Command{
				Use:  rootCmd.Use,
				RunE: runHTTPTest,
			}

			// Copy both regular and persistent flags from rootCmd
			cmd.Flags().AddFlagSet(rootCmd.Flags())
			cmd.PersistentFlags().AddFlagSet(rootCmd.PersistentFlags())

			// Set args
			cmd.SetArgs(tt.args)

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Execute
			err := cmd.Execute()

			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestScenarioExecution(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create a temp scenario file
	scenarioYAML := `name: "Test Scenario"
description: "Test"
steps:
  - name: "Step 1"
    url: "` + server.URL + `"
    method: GET
    expect:
      status_code: 200
`

	tmpfile, err := os.CreateTemp("", "scenario-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(scenarioYAML)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test scenario execution
	cmd := &cobra.Command{
		Use:  rootCmd.Use,
		RunE: runHTTPTest,
	}
	cmd.Flags().AddFlagSet(rootCmd.Flags())
	cmd.PersistentFlags().AddFlagSet(rootCmd.PersistentFlags())
	cmd.SetArgs([]string{"-s", tmpfile.Name()})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Scenario execution failed: %v", err)
	}

	// Note: Scenario output goes to stdout directly, not to cmd output
	// Just verify it executed without error
}

func TestInvalidScenarioFile(t *testing.T) {
	cmd := &cobra.Command{
		Use:  rootCmd.Use,
		RunE: runHTTPTest,
	}
	cmd.Flags().AddFlagSet(rootCmd.Flags())
	cmd.PersistentFlags().AddFlagSet(rootCmd.PersistentFlags())
	cmd.SetArgs([]string{"-s", "/nonexistent/file.yaml"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent scenario file")
	}
}

func TestNoURLProvided(t *testing.T) {
	cmd := &cobra.Command{
		Use:  rootCmd.Use,
		RunE: runHTTPTest,
	}
	cmd.Flags().AddFlagSet(rootCmd.Flags())
	cmd.PersistentFlags().AddFlagSet(rootCmd.PersistentFlags())
	cmd.SetArgs([]string{"-n", "10"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when no URL provided")
	}
}

func TestURLListFile(t *testing.T) {
	// Reset global variables that might interfere from previous tests
	scenarioFile = ""
	urlListFile = ""

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	// Create temp URL list file
	tmpfile, err := os.CreateTemp("", "urls-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	urls := server.URL + "\n" + server.URL + "/path\n"
	if _, err := tmpfile.Write([]byte(urls)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test with URL list file
	cmd := &cobra.Command{
		Use:  rootCmd.Use,
		RunE: runHTTPTest,
	}
	cmd.Flags().AddFlagSet(rootCmd.Flags())
	cmd.PersistentFlags().AddFlagSet(rootCmd.PersistentFlags())
	cmd.SetArgs([]string{"-q", "-L", tmpfile.Name(), "-n", "1"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	if err != nil {
		t.Errorf("URL list execution failed: %v", err)
	}
}
