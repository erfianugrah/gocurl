package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// URLReader handles reading URLs from various sources
type URLReader struct {
	urls []string
}

// NewURLReader creates a new URL reader
func NewURLReader() *URLReader {
	return &URLReader{
		urls: make([]string, 0),
	}
}

// ReadFromFile reads URLs from a file, one per line
func (r *URLReader) ReadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return r.readFromReader(file)
}

// ReadFromStdin reads URLs from stdin
func (r *URLReader) ReadFromStdin() error {
	return r.readFromReader(os.Stdin)
}

// readFromReader reads URLs from any io.Reader
func (r *URLReader) readFromReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			r.urls = append(r.urls, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

// AddURL adds a single URL to the list
func (r *URLReader) AddURL(url string) {
	r.urls = append(r.urls, url)
}

// GetURLs returns all collected URLs
func (r *URLReader) GetURLs() []string {
	return r.urls
}

// Count returns the number of URLs
func (r *URLReader) Count() int {
	return len(r.urls)
}
