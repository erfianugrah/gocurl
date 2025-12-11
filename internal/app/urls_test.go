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

func TestReadQueryParamsFromReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "simple params",
			input: `imwidth=1920
imwidth=1080
imwidth=480`,
			expected: []string{"imwidth=1920", "imwidth=1080", "imwidth=480"},
		},
		{
			name: "params with comments and empty lines",
			input: `# Image width variants
imwidth=1920

imwidth=1080
# Mobile
imwidth=480`,
			expected: []string{"imwidth=1920", "imwidth=1080", "imwidth=480"},
		},
		{
			name: "complex params",
			input: `imwidth=1920&quality=high
imwidth=1080&quality=medium
imwidth=480&quality=low`,
			expected: []string{"imwidth=1920&quality=high", "imwidth=1080&quality=medium", "imwidth=480&quality=low"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := readQueryParamsFromReader(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("readQueryParamsFromReader failed: %v", err)
			}

			if len(params) != len(tt.expected) {
				t.Errorf("Expected %d params, got %d", len(tt.expected), len(params))
				return
			}

			for i, expected := range tt.expected {
				if params[i] != expected {
					t.Errorf("Param %d: expected %q, got %q", i, expected, params[i])
				}
			}
		})
	}
}

func TestAppendQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		params   string
		expected string
	}{
		{
			name:     "simple URL without query string",
			baseURL:  "https://example.com/image.jpg",
			params:   "imwidth=1920",
			expected: "https://example.com/image.jpg?imwidth=1920",
		},
		{
			name:     "URL with existing query string",
			baseURL:  "https://example.com/image.jpg?format=png",
			params:   "imwidth=1920",
			expected: "https://example.com/image.jpg?format=png&imwidth=1920",
		},
		{
			name:     "URL with multiple existing params",
			baseURL:  "https://example.com/image.jpg?format=png&quality=high",
			params:   "imwidth=1920",
			expected: "https://example.com/image.jpg?format=png&quality=high&imwidth=1920",
		},
		{
			name:     "complex query params",
			baseURL:  "https://example.com/api/data",
			params:   "limit=10&offset=20&sort=asc",
			expected: "https://example.com/api/data?limit=10&offset=20&sort=asc",
		},
		{
			name:     "empty params",
			baseURL:  "https://example.com/image.jpg",
			params:   "",
			expected: "https://example.com/image.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendQueryParams(tt.baseURL, tt.params)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExpandURLsWithQueryParams(t *testing.T) {
	tests := []struct {
		name        string
		baseURLs    []string
		queryParams []string
		expected    []string
	}{
		{
			name:     "single URL with multiple params",
			baseURLs: []string{"https://example.com/image.jpg"},
			queryParams: []string{
				"imwidth=1920",
				"imwidth=1080",
				"imwidth=480",
			},
			expected: []string{
				"https://example.com/image.jpg?imwidth=1920",
				"https://example.com/image.jpg?imwidth=1080",
				"https://example.com/image.jpg?imwidth=480",
			},
		},
		{
			name: "multiple URLs with multiple params",
			baseURLs: []string{
				"https://example.com/users",
				"https://example.com/products",
			},
			queryParams: []string{
				"limit=10&offset=0",
				"limit=20&offset=20",
			},
			expected: []string{
				"https://example.com/users?limit=10&offset=0",
				"https://example.com/users?limit=20&offset=20",
				"https://example.com/products?limit=10&offset=0",
				"https://example.com/products?limit=20&offset=20",
			},
		},
		{
			name: "URLs with existing query strings",
			baseURLs: []string{
				"https://example.com/image.jpg?format=png",
			},
			queryParams: []string{
				"imwidth=1920",
				"imwidth=1080",
			},
			expected: []string{
				"https://example.com/image.jpg?format=png&imwidth=1920",
				"https://example.com/image.jpg?format=png&imwidth=1080",
			},
		},
		{
			name:        "empty query params returns original URLs",
			baseURLs:    []string{"https://example.com/api"},
			queryParams: []string{},
			expected:    []string{"https://example.com/api"},
		},
		{
			name:     "empty base URLs",
			baseURLs: []string{},
			queryParams: []string{
				"param=value",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandURLsWithQueryParams(tt.baseURLs, tt.queryParams)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d URLs, got %d", len(tt.expected), len(result))
				t.Logf("Expected: %v", tt.expected)
				t.Logf("Got: %v", result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("URL %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestReadQueryParamsFromFile(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "params-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test data
	content := `# Image width variants
imwidth=1920
imwidth=1080
# Mobile size
imwidth=480`

	_, err = tmpfile.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Test reading
	params, err := ReadQueryParamsFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadQueryParamsFromFile failed: %v", err)
	}

	expected := []string{"imwidth=1920", "imwidth=1080", "imwidth=480"}
	if len(params) != len(expected) {
		t.Errorf("Expected %d params, got %d", len(expected), len(params))
		return
	}

	for i, exp := range expected {
		if params[i] != exp {
			t.Errorf("Param %d: expected %q, got %q", i, exp, params[i])
		}
	}
}
