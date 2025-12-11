package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	verbose   bool
	version   string
	commit    string
	buildDate string
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(verbose bool, version, commit, buildDate string) *JSONFormatter {
	return &JSONFormatter{
		verbose:   verbose,
		version:   version,
		commit:    commit,
		buildDate: buildDate,
	}
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
	// Create output with version metadata
	type OutputWithMeta struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildDate string `json:"build_date"`
		Timestamp string `json:"timestamp"`
		*client.TimingBreakdown
	}

	output := &OutputWithMeta{
		Version:         f.version,
		Commit:          f.commit,
		BuildDate:       f.buildDate,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TimingBreakdown: timing,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
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
	// Create output with version metadata
	type OutputWithMeta struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildDate string `json:"build_date"`
		Timestamp string `json:"timestamp"`
		*metrics.Stats
	}

	output := &OutputWithMeta{
		Version:   f.version,
		Commit:    f.commit,
		BuildDate: f.buildDate,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Stats:     stats,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
