package app

import (
	"os"
	"strings"
	"testing"
)

func TestNewURLReader(t *testing.T) {
	reader := &URLReader{}

	if reader.urls == nil {
		reader.urls = make([]string, 0)
	}

	if len(reader.urls) != 0 {
		t.Error("New URLReader should have empty urls slice")
	}
}

func TestURLReaderAddURL(t *testing.T) {
	reader := &URLReader{}
	reader.urls = make([]string, 0)

	reader.AddURL("https://example.com")

	if len(reader.urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(reader.urls))
	}

	if reader.urls[0] != "https://example.com" {
		t.Errorf("Expected https://example.com, got %s", reader.urls[0])
	}
}

func TestURLReaderGetURLs(t *testing.T) {
	reader := &URLReader{
		urls: []string{
			"https://example1.com",
			"https://example2.com",
		},
	}

	urls := reader.GetURLs()

	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
	}
}

func TestURLReaderCount(t *testing.T) {
	reader := &URLReader{
		urls: []string{
			"https://example1.com",
			"https://example2.com",
			"https://example3.com",
		},
	}

	count := reader.Count()

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestURLReaderReadFromStdin(t *testing.T) {
	// Create a test string reader
	input := `https://example1.com
https://example2.com
# This is a comment
https://example3.com

https://example4.com`

	reader := &URLReader{urls: make([]string, 0)}
	stringReader := strings.NewReader(input)

	err := reader.readFromReader(stringReader)
	if err != nil {
		t.Fatalf("readFromReader failed: %v", err)
	}

	urls := reader.GetURLs()

	if len(urls) != 4 {
		t.Errorf("Expected 4 URLs, got %d", len(urls))
	}

	expected := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
		"https://example4.com",
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("URL %d: expected %s, got %s", i, expected[i], url)
		}
	}
}

func TestURLReaderReadFromFile(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "urls-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test data
	content := `https://api1.example.com
https://api2.example.com
# Comment line
https://api3.example.com`

	_, err = tmpfile.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Test reading
	reader := &URLReader{}
	err = reader.ReadFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadFromFile failed: %v", err)
	}

	urls := reader.GetURLs()

	if len(urls) != 3 {
		t.Errorf("Expected 3 URLs, got %d", len(urls))
	}

	expected := []string{
		"https://api1.example.com",
		"https://api2.example.com",
		"https://api3.example.com",
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("URL %d: expected %s, got %s", i, expected[i], url)
		}
	}
}

func TestURLReaderReadFromFileNotFound(t *testing.T) {
	reader := &URLReader{}
	err := reader.ReadFromFile("/nonexistent/file.txt")

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestURLReaderSkipEmptyLines(t *testing.T) {
	input := `https://example1.com


https://example2.com`

	reader := &URLReader{urls: make([]string, 0)}
	stringReader := strings.NewReader(input)

	err := reader.readFromReader(stringReader)
	if err != nil {
		t.Fatalf("readFromReader failed: %v", err)
	}

	urls := reader.GetURLs()

	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs (empty lines skipped), got %d", len(urls))
	}
}

func TestURLReaderSkipComments(t *testing.T) {
	input := `https://example1.com
# This is a comment
## Another comment
https://example2.com
### Yet another comment`

	reader := &URLReader{urls: make([]string, 0)}
	stringReader := strings.NewReader(input)

	err := reader.readFromReader(stringReader)
	if err != nil {
		t.Fatalf("readFromReader failed: %v", err)
	}

	urls := reader.GetURLs()

	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs (comments skipped), got %d", len(urls))
	}
}

func TestURLReaderTrimWhitespace(t *testing.T) {
	input := `  https://example1.com
	https://example2.com
		https://example3.com		`

	reader := &URLReader{urls: make([]string, 0)}
	stringReader := strings.NewReader(input)

	err := reader.readFromReader(stringReader)
	if err != nil {
		t.Fatalf("readFromReader failed: %v", err)
	}

	urls := reader.GetURLs()

	if len(urls) != 3 {
		t.Errorf("Expected 3 URLs, got %d", len(urls))
	}

	for i, url := range urls {
		if strings.TrimSpace(url) != url {
			t.Errorf("URL %d not properly trimmed: %q", i, url)
		}
	}
}

func TestURLReaderComplexFile(t *testing.T) {
	input := `# API Endpoints for Testing

https://api.example.com/users
https://api.example.com/products

# Authentication endpoints
https://api.example.com/auth/login
https://api.example.com/auth/logout


# Admin endpoints
https://api.example.com/admin/dashboard

### End of file`

	reader := &URLReader{urls: make([]string, 0)}
	stringReader := strings.NewReader(input)

	err := reader.readFromReader(stringReader)
	if err != nil {
		t.Fatalf("readFromReader failed: %v", err)
	}

	urls := reader.GetURLs()

	expectedCount := 5
	if len(urls) != expectedCount {
		t.Errorf("Expected %d URLs, got %d", expectedCount, len(urls))
	}

	// Verify specific URLs
	expectedURLs := []string{
		"https://api.example.com/users",
		"https://api.example.com/products",
		"https://api.example.com/auth/login",
		"https://api.example.com/auth/logout",
		"https://api.example.com/admin/dashboard",
	}

	for i, url := range urls {
		if url != expectedURLs[i] {
			t.Errorf("URL %d: expected %s, got %s", i, expectedURLs[i], url)
		}
	}
}
