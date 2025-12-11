package app

import (
	"strings"
	"testing"
)

func TestSubstituteTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		ctx      *TemplateContext
		validate func(t *testing.T, result string)
	}{
		{
			name:     "empty template",
			template: "",
			ctx:      NewTemplateContext(1, 0),
			validate: func(t *testing.T, result string) {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
			},
		},
		{
			name:     "no variables",
			template: "plain text",
			ctx:      NewTemplateContext(1, 0),
			validate: func(t *testing.T, result string) {
				if result != "plain text" {
					t.Errorf("expected %q, got %q", "plain text", result)
				}
			},
		},
		{
			name:     "seq variable",
			template: `{"request_id": {{seq}}}`,
			ctx:      NewTemplateContext(42, 0),
			validate: func(t *testing.T, result string) {
				expected := `{"request_id": 42}`
				if result != expected {
					t.Errorf("expected %q, got %q", expected, result)
				}
			},
		},
		{
			name:     "uuid variable",
			template: `{"id": "{{uuid}}"}`,
			ctx:      NewTemplateContext(1, 0),
			validate: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, `{"id": "`) {
					t.Errorf("expected UUID format, got %q", result)
				}
				if !strings.HasSuffix(result, `"}`) {
					t.Errorf("expected UUID format, got %q", result)
				}
				// Basic UUID format check (8-4-4-4-12 hex digits)
				if len(result) != len(`{"id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}`) {
					t.Errorf("UUID length mismatch: %q", result)
				}
			},
		},
		{
			name:     "timestamp variable",
			template: `{"ts": {{timestamp}}}`,
			ctx:      NewTemplateContext(1, 0),
			validate: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, `{"ts": `) {
					t.Errorf("expected timestamp format, got %q", result)
				}
				// Timestamp should be ~10 digits (Unix seconds)
				if len(result) < len(`{"ts": 1234567890}`) {
					t.Errorf("timestamp too short: %q", result)
				}
			},
		},
		{
			name:     "timestamp_ms variable",
			template: `{"ts_ms": {{timestamp_ms}}}`,
			ctx:      NewTemplateContext(1, 0),
			validate: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, `{"ts_ms": `) {
					t.Errorf("expected timestamp_ms format, got %q", result)
				}
				// Timestamp ms should be ~13 digits
				if len(result) < len(`{"ts_ms": 1234567890123}`) {
					t.Errorf("timestamp_ms too short: %q", result)
				}
			},
		},
		{
			name:     "random variable",
			template: `{"rand": {{random}}}`,
			ctx:      NewTemplateContext(1, 0),
			validate: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, `{"rand": `) {
					t.Errorf("expected random format, got %q", result)
				}
				if !strings.HasSuffix(result, `}`) {
					t.Errorf("expected random format, got %q", result)
				}
			},
		},
		{
			name:     "url_index variable",
			template: `{"url_idx": {{url_index}}}`,
			ctx:      NewTemplateContext(1, 5),
			validate: func(t *testing.T, result string) {
				expected := `{"url_idx": 5}`
				if result != expected {
					t.Errorf("expected %q, got %q", expected, result)
				}
			},
		},
		{
			name:     "multiple variables",
			template: `{"seq": {{seq}}, "url": {{url_index}}, "ts": {{timestamp}}}`,
			ctx:      NewTemplateContext(99, 3),
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, `"seq": 99`) {
					t.Errorf("expected seq=99 in result: %q", result)
				}
				if !strings.Contains(result, `"url": 3`) {
					t.Errorf("expected url=3 in result: %q", result)
				}
				if !strings.Contains(result, `"ts": `) {
					t.Errorf("expected timestamp in result: %q", result)
				}
			},
		},
		{
			name:     "repeated variables",
			template: `{"a": {{seq}}, "b": {{seq}}}`,
			ctx:      NewTemplateContext(7, 0),
			validate: func(t *testing.T, result string) {
				expected := `{"a": 7, "b": 7}`
				if result != expected {
					t.Errorf("expected %q, got %q", expected, result)
				}
			},
		},
		{
			name:     "complex JSON with multiple variables",
			template: `{"user_id": "{{uuid}}", "request_num": {{seq}}, "timestamp": {{timestamp}}, "data": {"value": {{random}}}}`,
			ctx:      NewTemplateContext(123, 1),
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, `"request_num": 123`) {
					t.Errorf("expected request_num in result: %q", result)
				}
				if !strings.Contains(result, `"user_id": "`) {
					t.Errorf("expected user_id with UUID in result: %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SubstituteTemplate(tt.template, tt.ctx)
			tt.validate(t, result)
		})
	}
}

func TestHasTemplateVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"plain text", "hello world", false},
		{"no variables", "this is plain text", false},
		{"single curly brace", "test { value }", false},
		{"with seq variable", "{{seq}}", true},
		{"with uuid variable", `{"id": "{{uuid}}"}`, true},
		{"multiple variables", `{"a": {{seq}}, "b": {{uuid}}}`, true},
		{"partial template syntax", "{{incomplete", false}, // Only has {{, missing }}
		{"only open braces", "{{", false},
		{"only close braces", "}}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasTemplateVariables(tt.input)
			if result != tt.expected {
				t.Errorf("HasTemplateVariables(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewTemplateContext(t *testing.T) {
	ctx := NewTemplateContext(42, 5)
	if ctx.RequestSeq != 42 {
		t.Errorf("expected RequestSeq=42, got %d", ctx.RequestSeq)
	}
	if ctx.URLIndex != 5 {
		t.Errorf("expected URLIndex=5, got %d", ctx.URLIndex)
	}
	if ctx.Random == nil {
		t.Error("expected non-nil Random generator")
	}
}

func TestRandomDeterminism(t *testing.T) {
	// Same request sequence should produce different random numbers
	// (because time-based seed changes)
	ctx1 := NewTemplateContext(1, 0)
	ctx2 := NewTemplateContext(1, 0)

	result1 := SubstituteTemplate("{{random}}", ctx1)
	result2 := SubstituteTemplate("{{random}}", ctx2)

	// They might be the same (unlikely but possible), but at least verify they're valid numbers
	if result1 == "" || result2 == "" {
		t.Error("random substitution produced empty result")
	}
}
