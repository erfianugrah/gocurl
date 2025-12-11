package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/erfi/gocurl/internal/client"
)

// ExportCSV writes timing data to a CSV file
// Each row represents one individual request with all timing breakdowns
func ExportCSV(filename string, timings []*client.TimingBreakdown) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{
		"timestamp",
		"url",
		"status_code",
		"dns_lookup_ms",
		"tcp_connection_ms",
		"tls_handshake_ms",
		"server_processing_ms",
		"content_transfer_ms",
		"total_ms",
		"response_size_bytes",
		"connection_reused",
		"error",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write each timing as a row
	for _, timing := range timings {
		row := []string{
			time.Now().Format(time.RFC3339),
			timing.URL,
			fmt.Sprintf("%d", timing.StatusCode),
			fmt.Sprintf("%.2f", timing.DNSLookup.Seconds()*1000),
			fmt.Sprintf("%.2f", timing.TCPConnection.Seconds()*1000),
			fmt.Sprintf("%.2f", timing.TLSHandshake.Seconds()*1000),
			fmt.Sprintf("%.2f", timing.ServerProcessing.Seconds()*1000),
			fmt.Sprintf("%.2f", timing.ContentTransfer.Seconds()*1000),
			fmt.Sprintf("%.2f", timing.Total.Seconds()*1000),
			fmt.Sprintf("%d", timing.ResponseSize),
			fmt.Sprintf("%t", timing.ConnectionReused),
			timing.Error,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}
