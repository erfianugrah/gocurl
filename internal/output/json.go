package output

import (
	"encoding/json"
	"io"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	verbose bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(verbose bool) *JSONFormatter {
	return &JSONFormatter{verbose: verbose}
}

// Format formats a single timing result as JSON
func (f *JSONFormatter) Format(timing *client.TimingBreakdown) (string, error) {
	data, err := json.MarshalIndent(timing, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write writes a single timing result as JSON to the writer
func (f *JSONFormatter) Write(w io.Writer, timing *client.TimingBreakdown) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(timing)
}

// FormatMultiple formats multiple timing results as JSON
func (f *JSONFormatter) FormatMultiple(stats *metrics.Stats) (string, error) {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteMultiple writes multiple timing results as JSON to the writer
func (f *JSONFormatter) WriteMultiple(w io.Writer, stats *metrics.Stats) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(stats)
}
