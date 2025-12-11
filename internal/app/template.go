package app

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TemplateContext holds variables for template substitution
type TemplateContext struct {
	RequestSeq int        // Sequential number for this request (1-indexed)
	URLIndex   int        // Index of the URL being tested (0-indexed)
	Random     *rand.Rand // Random number generator
}

// NewTemplateContext creates a new template context
func NewTemplateContext(requestSeq, urlIndex int) *TemplateContext {
	return &TemplateContext{
		RequestSeq: requestSeq,
		URLIndex:   urlIndex,
		Random:     rand.New(rand.NewSource(time.Now().UnixNano() + int64(requestSeq))),
	}
}

// SubstituteTemplate replaces template variables in a string with actual values
//
// Supported variables:
//   - {{seq}} - Sequential number starting from 1
//   - {{uuid}} - Random UUID v4
//   - {{timestamp}} - Unix timestamp in seconds
//   - {{timestamp_ms}} - Unix timestamp in milliseconds
//   - {{random}} - Random integer (0-999999)
//   - {{url_index}} - Index of the URL being tested
func SubstituteTemplate(template string, ctx *TemplateContext) string {
	if template == "" {
		return template
	}

	result := template
	now := time.Now()

	// Replace template variables
	replacements := map[string]string{
		"{{seq}}":          fmt.Sprintf("%d", ctx.RequestSeq),
		"{{uuid}}":         uuid.New().String(),
		"{{timestamp}}":    fmt.Sprintf("%d", now.Unix()),
		"{{timestamp_ms}}": fmt.Sprintf("%d", now.UnixMilli()),
		"{{random}}":       fmt.Sprintf("%d", ctx.Random.Intn(1000000)),
		"{{url_index}}":    fmt.Sprintf("%d", ctx.URLIndex),
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// HasTemplateVariables checks if a string contains template variables
func HasTemplateVariables(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}
