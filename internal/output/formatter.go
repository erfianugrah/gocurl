package output

import (
	"io"

	"github.com/erfi/gocurl/internal/client"
	"github.com/erfi/gocurl/internal/metrics"
)

// Formatter defines the interface for different output formats
type Formatter interface {
	Format(timing *client.TimingBreakdown) (string, error)
	FormatMultiple(stats *metrics.Stats) (string, error)
	Write(w io.Writer, timing *client.TimingBreakdown) error
	WriteMultiple(w io.Writer, stats *metrics.Stats) error
}

// GetFormatter returns the appropriate formatter based on the format string
func GetFormatter(format string, verbose bool) (Formatter, error) {
	switch format {
	case "json":
		return NewJSONFormatter(verbose), nil
	case "table":
		return NewTableFormatter(verbose), nil
	case "graph":
		return NewGraphFormatter(verbose), nil
	default:
		return NewTableFormatter(verbose), nil
	}
}
