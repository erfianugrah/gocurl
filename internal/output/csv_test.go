package output

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/erfi/gocurl/internal/client"
)

func TestExportCSV(t *testing.T) {
	tests := []struct {
		name      string
		timings   []*client.TimingBreakdown
		wantError bool
		checkRows int // expected number of data rows (excluding header)
	}{
		{
			name: "single timing",
			timings: []*client.TimingBreakdown{
				{
					URL:              "https://example.com",
					StatusCode:       200,
					DNSLookup:        client.Duration(10000000),  // 10ms
					TCPConnection:    client.Duration(20000000),  // 20ms
					TLSHandshake:     client.Duration(30000000),  // 30ms
					ServerProcessing: client.Duration(40000000),  // 40ms
					ContentTransfer:  client.Duration(5000000),   // 5ms
					Total:            client.Duration(105000000), // 105ms
					ResponseSize:     1024,
					ConnectionReused: false,
					Error:            "",
				},
			},
			wantError: false,
			checkRows: 1,
		},
		{
			name: "multiple timings",
			timings: []*client.TimingBreakdown{
				{
					URL:        "https://example.com/1",
					StatusCode: 200,
					Total:      client.Duration(100000000),
				},
				{
					URL:        "https://example.com/2",
					StatusCode: 404,
					Total:      client.Duration(150000000),
					Error:      "not found",
				},
				{
					URL:              "https://example.com/3",
					StatusCode:       200,
					Total:            client.Duration(200000000),
					ConnectionReused: true,
				},
			},
			wantError: false,
			checkRows: 3,
		},
		{
			name:      "empty timings",
			timings:   []*client.TimingBreakdown{},
			wantError: false,
			checkRows: 0,
		},
		{
			name: "timing with error",
			timings: []*client.TimingBreakdown{
				{
					URL:        "https://example.com",
					StatusCode: 500,
					Total:      client.Duration(50000000),
					Error:      "connection timeout",
				},
			},
			wantError: false,
			checkRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp("", "test-*.csv")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())
			tmpfile.Close()

			// Export CSV
			err = ExportCSV(tmpfile.Name(), tt.timings)

			if (err != nil) != tt.wantError {
				t.Errorf("ExportCSV() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			// Read and verify the CSV
			file, err := os.Open(tmpfile.Name())
			if err != nil {
				t.Fatalf("failed to open CSV file: %v", err)
			}
			defer file.Close()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			if err != nil {
				t.Fatalf("failed to read CSV: %v", err)
			}

			// Check number of rows (header + data rows)
			expectedRows := tt.checkRows + 1 // +1 for header
			if len(records) != expectedRows {
				t.Errorf("CSV has %d rows, want %d", len(records), expectedRows)
			}

			// Check header
			if len(records) > 0 {
				expectedHeader := []string{
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

				if len(records[0]) != len(expectedHeader) {
					t.Errorf("CSV header has %d columns, want %d", len(records[0]), len(expectedHeader))
				}

				for i, col := range expectedHeader {
					if i < len(records[0]) && records[0][i] != col {
						t.Errorf("CSV header column %d = %q, want %q", i, records[0][i], col)
					}
				}
			}

			// Verify data rows
			for i, timing := range tt.timings {
				if i+1 >= len(records) {
					break
				}
				row := records[i+1]

				// Check URL column
				if len(row) > 1 && row[1] != timing.URL {
					t.Errorf("Row %d URL = %q, want %q", i, row[1], timing.URL)
				}

				// Check status code column
				if len(row) > 2 && !strings.Contains(row[2], "200") && !strings.Contains(row[2], "404") && !strings.Contains(row[2], "500") {
					t.Errorf("Row %d status code = %q", i, row[2])
				}

				// Check error column
				if len(row) > 11 && timing.Error != "" && row[11] != timing.Error {
					t.Errorf("Row %d error = %q, want %q", i, row[11], timing.Error)
				}
			}
		})
	}
}

func TestExportCSVInvalidPath(t *testing.T) {
	timings := []*client.TimingBreakdown{
		{
			URL:        "https://example.com",
			StatusCode: 200,
			Total:      client.Duration(100000000),
		},
	}

	// Try to write to an invalid path
	err := ExportCSV("/invalid/path/that/does/not/exist/test.csv", timings)
	if err == nil {
		t.Error("ExportCSV() with invalid path should return error")
	}
}

func TestExportCSVContentVerification(t *testing.T) {
	// Create a timing with specific values to verify formatting
	timings := []*client.TimingBreakdown{
		{
			URL:              "https://test.example.com/path",
			StatusCode:       201,
			DNSLookup:        client.Duration(15000000),  // 15ms
			TCPConnection:    client.Duration(25000000),  // 25ms
			TLSHandshake:     client.Duration(35000000),  // 35ms
			ServerProcessing: client.Duration(45000000),  // 45ms
			ContentTransfer:  client.Duration(10000000),  // 10ms
			Total:            client.Duration(130000000), // 130ms
			ResponseSize:     2048,
			ConnectionReused: true,
			Error:            "",
		},
	}

	// Create temp file
	tmpfile, err := os.CreateTemp("", "test-content-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Export CSV
	err = ExportCSV(tmpfile.Name(), timings)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}

	// Read the file content
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify specific values are present
	checks := []string{
		"https://test.example.com/path",
		"201",  // status code
		"2048", // response size
		"true", // connection reused
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("CSV content missing expected value %q\nContent:\n%s", check, contentStr)
		}
	}

	// Verify it contains timing values (formatted as decimal milliseconds)
	if !strings.Contains(contentStr, "15.00") { // DNS lookup
		t.Error("CSV should contain DNS lookup time")
	}
	if !strings.Contains(contentStr, "130.00") { // Total time
		t.Error("CSV should contain total time")
	}
}
