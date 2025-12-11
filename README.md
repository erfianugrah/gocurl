# gocurl

[![Build and Test](https://github.com/erfianugrah/gocurl/actions/workflows/ci.yml/badge.svg)](https://github.com/erfianugrah/gocurl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/erfianugrah/gocurl)](https://goreportcard.com/report/github.com/erfianugrah/gocurl)

> A modern HTTP performance measurement tool built in Go

`gocurl` is a production-grade CLI tool for measuring HTTP performance with detailed timing breakdowns, load testing capabilities, and beautiful output formatting.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
  - [Installation](#installation)
  - [Requirements](#requirements)
- [Usage](#usage)
  - [Basic Usage](#basic-usage)
  - [Output Formats](#output-formats)
  - [Load Testing](#load-testing)
  - [Multi-URL Testing](#multi-url-testing)
  - [Custom Headers and Methods](#custom-headers-and-methods)
  - [Response Inspection](#response-inspection-curl-like)
  - [Connection Control](#connection-control)
  - [Streaming & Buffering Detection](#streaming--buffering-detection)
- [Command Reference](#command-reference)
- [Examples](#examples)
- [Building from Source](#building-from-source)
- [Development](#development)
- [CI/CD](#cicd)
- [Contributing](#contributing)
- [License](#license)

## Features

- 🚀 **Detailed Performance Metrics** - DNS, TCP, TLS, server processing, and content transfer timing
- 🎨 **Waterfall Timeline** - Chrome DevTools-style visual timeline showing request phases
- 📊 **Multiple Output Formats** - Table, JSON, or ASCII graphs with histograms
- ⚡ **Load Testing** - Concurrent requests with configurable workers
- 🎯 **URL Parameterization** - Test query parameter variants (e.g., different image sizes)
- 🔥 **Warmup Phase** - Exclude initial requests from metrics to eliminate cold-start effects
- ⏱️ **Rate Limiting** - Control RPS for realistic load testing
- 📊 **CSV Export** - Export individual request timings for deep analysis in spreadsheets or data pipelines
- 🎭 **Request Body Templates** - Dynamic data generation with {{seq}}, {{uuid}}, {{timestamp}}, {{random}} variables
- 📈 **Ramp-up Configuration** - Gradually increase concurrency to test autoscaling and find performance limits
- 🎯 **Selective Header Capture** - Capture specific response headers (Cache-Control, ETag, etc.) without capturing everything
- 🔀 **Redirect Tracking** - Automatically track effective URLs and redirect counts in response data
- 📦 **Range Request Support** - Test partial content delivery with HTTP Range headers (HTTP 206)
- 🎬 **Scenario Testing** - Multi-step workflows with variable extraction, session management, and expectations
- 📈 **Statistical Analysis** - Percentiles (p50, p90, p95, p99, p99.9, p99.99)
- 🎯 **Multi-URL Testing** - Test multiple endpoints from files or stdin
- 🔐 **TLS Information** - Version, cipher suite, and SNI details
- 🔧 **curl-like Interface** - Familiar flags: `-i`, `-I`, `-H`, `-X`, `-k`
- 📝 **Response Inspection** - Headers, body, and error details
- 🌊 **Streaming Analysis** - Detect buffering, analyze chunk patterns, measure delivery characteristics
- 🔌 **Connection Control** - DNS resolution override (`--resolve`) and connection routing (`--connect-to`)

## Quick Start

### Installation

#### From Source

```bash
# Clone the repository
git clone https://github.com/erfianugrah/gocurl.git
cd gocurl

# Build using Make
make build

# Install to your PATH
make install
```

Or build manually:

```bash
# Build to bin/ directory
go build -o bin/gocurl ./cmd/gocurl

# Run from bin/
./bin/gocurl https://example.com
```

#### From GitHub Releases

Download pre-built binaries from the [Releases page](https://github.com/erfianugrah/gocurl/releases).

```bash
# Example for Linux
wget https://github.com/erfianugrah/gocurl/releases/latest/download/gocurl_Linux_x86_64.tar.gz
tar xzf gocurl_Linux_x86_64.tar.gz
sudo mv gocurl /usr/local/bin/
```

### Requirements

- **Go 1.21 or later** (for building from source)
- No external runtime dependencies

## Usage

### Basic Usage

```bash
# Simple request with performance breakdown
gocurl https://api.example.com

# Output:
# ✓ Status: 200 OK
# ✓ Time: 245ms
#
# Request Timeline:
#   ████████████████████████████████████████████████████████ 245ms
#   ■ DNS (12ms)  ■ TCP (45ms)  ■ TLS (156ms)  ■ Server (32ms)
#
# ┌───────────────────────────────────────────┐
# │ Timing Breakdown                          │
# ├───────────────────┬──────────┬────────────┤
# │ PHASE             │ DURATION │ % OF TOTAL │
# ├───────────────────┼──────────┼────────────┤
# │ DNS Lookup        │ 12ms     │ 4.9%       │
# │ TCP Connection    │ 45ms     │ 18.4%      │
# │ TLS Handshake     │ 156ms    │ 63.7%      │
# │ Server Processing │ 32ms     │ 13.1%      │
# └───────────────────┴──────────┴────────────┘
```

### Output Formats

#### Table Format (Default)
```bash
gocurl https://api.example.com
```
Beautiful terminal output with waterfall timeline visualization.

#### JSON Format
```bash
gocurl -o json https://api.example.com
```
Machine-readable output for CI/CD integration (values in milliseconds):
```json
{
  "dns_lookup": 12,
  "tcp_connection": 45,
  "tls_handshake": 156,
  "server_processing": 32,
  "content_transfer": 0,
  "total": 245,
  "status_code": 200,
  "tls_version": "TLS 1.3",
  "tls_cipher_suite": "TLS_AES_128_GCM_SHA256"
}
```

#### Graph Format
```bash
gocurl -o graph -n 100 -c 10 https://api.example.com
```
ASCII histogram showing latency distribution:
```
Latency Distribution:
       80-90ms │████████████████████ 45 (45.0%)
     290-300ms │████████ 18 (18.0%)
```

### Load Testing

#### Simple Load Test
```bash
# 100 requests with 10 concurrent workers
gocurl -n 100 -c 10 https://api.example.com
```

#### Advanced Load Test with Rate Limiting
```bash
# 1000 requests, 50 concurrent, limited to 100 RPS
gocurl -n 1000 -c 50 --rps 100 https://api.example.com
```

#### Load Test with Warmup
```bash
# 100 requests, skip first 10 from metrics
gocurl -n 100 -c 10 --warmup 10 https://api.example.com
```

#### URL Parameterization
Test multiple query parameter variants:
```bash
# Create query params file
cat > params.txt <<EOF
imwidth=1920
imwidth=1080
imwidth=480
EOF

# Test all variants (1 URL × 3 params = 3 total URLs)
gocurl https://cdn.example.com/image.jpg --query-params params.txt -n 10 -c 5
```

#### Combined Load Testing
Combine all features for comprehensive testing:
```bash
# Multiple URLs with query params, warmup, and rate limiting
gocurl -L urls.txt \
  --query-params params.txt \
  -n 50 \
  -c 10 \
  --warmup 5 \
  --rps 20
```

#### CSV Export
Export individual request data for analysis:
```bash
# Basic CSV export
gocurl https://api.example.com -n 100 -c 10 --export-csv results.csv

# With warmup (excluded from CSV)
gocurl https://api.example.com -n 100 -c 10 --warmup 10 --export-csv results.csv

# Multiple URLs with all features
gocurl -L urls.txt --query-params params.txt \
  -n 50 -c 10 --warmup 5 --rps 20 \
  --export-csv load-test.csv
```

**CSV contains** (12 columns):
- timestamp, url, status_code
- dns_lookup_ms, tcp_connection_ms, tls_handshake_ms
- server_processing_ms, content_transfer_ms, total_ms
- response_size_bytes, connection_reused, error

**Analyze with tools:**
```bash
# Command-line analysis
awk -F',' 'NR>1 {sum+=$9; count++} END {print "Avg:", sum/count "ms"}' results.csv

# Find slowest requests
awk -F',' 'NR>1 {print $9, $2}' results.csv | sort -rn | head -10

# Import into Excel, Google Sheets, or data analysis tools
# - Create pivot tables
# - Generate custom charts
# - Statistical modeling
```

#### Request Body Templates

Generate dynamic request bodies with template variables for realistic load testing:

**Available Variables:**
- `{{seq}}` - Sequential request number (1-indexed)
- `{{uuid}}` - Unique UUID v4 for each request
- `{{timestamp}}` - Unix timestamp in seconds
- `{{timestamp_ms}}` - Unix timestamp in milliseconds
- `{{random}}` - Random number (0-999999)
- `{{url_index}}` - Index of current URL (0-indexed, useful with `-L`)

**Basic Example:**
```bash
# Create users with unique IDs
gocurl -X POST \
  -H "Content-Type: application/json" \
  --data '{"user_id": "{{uuid}}", "request_num": {{seq}}, "timestamp": {{timestamp}}}' \
  https://api.example.com/users \
  -n 10 -c 3
```

**Event Simulation:**
```bash
# Simulate event stream with realistic data
gocurl -X POST \
  -H "Content-Type: application/json" \
  --data '{"event_id": "{{uuid}}", "seq": {{seq}}, "ts": {{timestamp_ms}}, "value": {{random}}}' \
  https://api.example.com/events \
  -n 1000 -c 20 --rps 50
```

**Multi-URL with Templates:**
```bash
# Different request bodies for each URL
cat > endpoints.txt <<EOF
https://api.example.com/users
https://api.example.com/orders
https://api.example.com/events
EOF

# Each URL gets requests with unique data
gocurl -L endpoints.txt \
  -X POST \
  -H "Content-Type: application/json" \
  --data '{"id": "{{uuid}}", "url_idx": {{url_index}}, "seq": {{seq}}}' \
  -n 20 -c 5
```

**When to Use Templates:**
- Testing ID generation and uniqueness constraints
- Simulating real user behavior with varied data
- Load testing time-series databases
- Stress testing with high-cardinality data
- Avoiding cache hits with unique request bodies

#### Ramp-up Configuration

Gradually increase concurrency to find performance limits and test autoscaling:

**Basic Ramp-up:**
```bash
# Ramp from 1 to 50 workers over 30 seconds
gocurl https://api.example.com \
  -n 1000 \
  -c 50 \
  --ramp-up 30s
```

Output shows:
```
Ramp-up: gradually increasing from 1 to 50 workers over 30s
Running load test: 1 URLs x 1000 requests = 1000 total requests with concurrency 50
```

**Find Breaking Point:**
```bash
# Gradually increase to 100 workers over 2 minutes
gocurl https://api.example.com/heavy \
  -n 5000 \
  -c 100 \
  --ramp-up 2m \
  --export-csv rampup-results.csv

# Analyze when errors start occurring
awk -F',' 'NR>1 {if ($12!="") print $1, $12}' rampup-results.csv
```

**Test Autoscaling:**
```bash
# Test if autoscaler responds to gradual load
gocurl https://k8s-app.example.com \
  -n 10000 \
  -c 200 \
  --ramp-up 5m \
  --rps 100 \
  --export-csv autoscale-test.csv
```

**Combined with Rate Limiting:**
```bash
# Controlled ramp-up with rate limiting
gocurl https://api.example.com \
  -n 2000 \
  -c 50 \
  --ramp-up 1m \
  --rps 100 \
  --warmup 10
```

**When to Use Ramp-up:**
- Testing autoscaling behavior (Kubernetes HPA, AWS Auto Scaling)
- Finding the exact concurrency level where performance degrades
- Avoiding thundering herd problems in initial load tests
- Testing gradual traffic increases (like Black Friday scenarios)
- Warming up connection pools and caches progressively

**How It Works:**
- Workers start at t=0, t=interval, t=2*interval, ..., t=duration
- Linear distribution: `interval = duration / (concurrency - 1)`
- Example: 10 workers over 30s = 1 worker every 3.33 seconds
- All workers continue running once started (not a wave pattern)

### Multi-URL Testing

#### From File
```bash
# Create a URL list file
cat > urls.txt <<EOF
https://api.example.com/users
https://api.example.com/products
https://api.example.com/orders
EOF

# Test all URLs
gocurl -L urls.txt -n 10 -c 5
```

#### From stdin
```bash
# Pipe URLs directly
echo -e "https://api1.com\nhttps://api2.com" | gocurl -L - -n 5

# From another command
cat endpoints.txt | gocurl -L - -n 50 -c 10
```

### Custom Headers and Methods

#### POST Request
```bash
gocurl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  --data '{"user":"john","action":"login"}' \
  https://api.example.com/auth
```

#### Custom Headers
```bash
gocurl -H "User-Agent: MyApp/1.0" \
       -H "Accept: application/json" \
       https://api.example.com
```

### Response Inspection (curl-like)

#### Show Response Headers
```bash
# Include headers in output (like curl -i)
gocurl -i https://api.example.com

# HEAD request (like curl -I)
gocurl -I https://api.example.com
```

#### Show Response Body
```bash
# Show response body
gocurl --show-body https://api.example.com/data

# Show error response bodies (4xx, 5xx)
gocurl --show-error https://api.example.com/404
```

#### Verbose Mode with TLS Details
```bash
# Detailed output including TLS information
gocurl -v https://api.example.com

# Output includes:
# - Response size
# - TLS version (1.0/1.1/1.2/1.3)
# - Cipher suite
# - SNI hostname
# - Connection reuse info
```

### Connection Control

#### DNS Resolution Override (`--resolve`)

Override DNS resolution for specific host:port combinations (like adding a temporary /etc/hosts entry):

```bash
# Test new server IP before DNS change
gocurl --resolve api.example.com:443:192.168.1.100 https://api.example.com

# Test multiple hosts
gocurl --resolve api1.com:443:10.0.0.1 \
       --resolve api2.com:443:10.0.0.2 \
       https://api1.com
```

#### Connection Override (`--connect-to`)

Connect to a different host:port than the URL specifies, while keeping the original Host header and SNI:

```bash
# Connect to backend server directly, bypassing load balancer
gocurl --connect-to api.example.com:443:backend1.internal:443 https://api.example.com

# Test local development with production domain
gocurl --connect-to api.example.com:443:localhost:8443 -k https://api.example.com

# Performance test specific backend nodes
gocurl --connect-to api.example.com:443:backend2.internal:443 \
       -n 100 -c 10 https://api.example.com
```

### Streaming & Buffering Detection

#### Streaming Analysis

Enable detailed analysis of streaming behavior and detect buffering issues:

```bash
# Basic streaming analysis
gocurl --streaming https://api.example.com/stream

# Output includes:
# - Response header analysis (Transfer-Encoding, Content-Length)
# - Buffering detection with confidence score
# - Chunk delivery pattern analysis (steady, burst, stalled)
# - Statistical metrics: CV, mean/stddev/min/max inter-chunk delays
# - Objective timing measurements (no subjective quality assessments)
```

**📖 Need help interpreting the metrics?** See [Streaming Metrics Guide](docs/STREAMING_METRICS_GUIDE.md) for:
- What CV, mean delay, and patterns mean
- How to identify buffered vs progressive delivery
- Real-world examples with interpretation

#### Validate Streaming (CI/CD)

Exit with error if streaming is not detected (useful for automated tests):

```bash
# Fail if endpoint is not streaming properly
gocurl --expect-streaming https://api.example.com/events || exit 1

# Use in CI/CD pipeline
if ! gocurl --expect-streaming https://sse.example.com/feed; then
  echo "FAIL: Streaming endpoint is buffered!"
  exit 1
fi
```

#### Stall Detection

Configure threshold for detecting pauses in data delivery:

```bash
# Detect stalls longer than 1 second
gocurl --streaming --stall-threshold 1s https://api.example.com/stream

# Shows:
# - Number of stalls detected
# - Total stall time
# - Position in stream where stalls occurred
```

### Advanced Options

```bash
# Skip TLS verification (for self-signed certs)
gocurl -k https://localhost:8443

# Custom timeout
gocurl --timeout 10s https://slow-api.example.com

# Verbose output
gocurl -v https://api.example.com

# Disable colors (for logging)
gocurl --no-color https://api.example.com
```

## Command Reference

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--output` | `-o` | Output format: table, json, graph | `table` |
| `--no-color` | | Disable colored output | `false` |
| `--verbose` | `-v` | Verbose output | `false` |
| `--quiet` | `-q` | Minimal output (errors only) | `false` |

### HTTP Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--requests` | `-n` | Number of requests per URL | `1` |
| `--concurrency` | `-c` | Concurrent workers | `1` |
| `--warmup` | | Number of warmup requests to skip from metrics (per URL) | `0` |
| `--rps` | | Rate limit: requests per second (0 = unlimited) | `0` |
| `--ramp-up` | | Gradually increase concurrency over duration (e.g., '30s', '1m') | |
| `--export-csv` | | Export individual request data to CSV file | |
| `--url-list` | `-L` | File with URLs (use '-' for stdin) | |
| `--query-params` | | File with query parameters (one per line) to append to each URL | |
| `--method` | `-X` | HTTP method | `GET` |
| `--header` | `-H` | Custom header (repeatable) | |
| `--data` | | Request body (supports template variables: {{seq}}, {{uuid}}, {{timestamp}}, {{timestamp_ms}}, {{random}}, {{url_index}}) | |
| `--timeout` | | Request timeout | `30s` |
| `--insecure` | `-k` | Skip TLS verification | `false` |

### Connection Control Flags

| Flag | Description | Format |
|------|-------------|--------|
| `--resolve` | Resolve host:port to address (repeatable) | `host:port:addr` |
| `--connect-to` | Connect to different host:port (repeatable) | `host1:port1:host2:port2` |

### Response Display Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--include` | `-i` | Include response headers in output | `false` |
| `--head` | `-I` | Make HEAD request (show headers only) | `false` |
| `--show-body` | | Show response body in output | `false` |
| `--show-error` | | Show response body for errors (4xx, 5xx) | `false` |
| `--capture-header` | | Capture specific response headers (repeatable) | |
| `--range` | | Request partial content (e.g., 'bytes=0-1023') | |
| `--scenario` | `-s` | Execute multi-step scenario from YAML file | |

### Streaming & Performance Analysis Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--streaming` | Enable detailed streaming metrics | `false` |
| `--expect-streaming` | Exit with error if streaming not detected (implies --streaming) | `false` |
| `--stall-threshold` | Duration threshold for detecting stalls | `500ms` |

## Examples

### API Performance Monitoring

```bash
# Monitor API and log p95 latency
while true; do
  P95=$(gocurl -n 100 -c 10 -o json https://api.example.com | jq '.p95')
  echo "$(date): P95=${P95}ms" >> performance.log
  sleep 60
done
```

### CI/CD Integration

```bash
# Fail build if p95 > 500ms
#!/bin/bash
RESULT=$(gocurl -n 100 -c 10 -o json https://api.example.com)
P95=$(echo $RESULT | jq '.p95')

if [ "$P95" -gt 500 ]; then
  echo "FAIL: P95 latency ${P95}ms exceeds 500ms threshold"
  exit 1
fi

echo "PASS: P95 latency ${P95}ms"
```

### Compare Environments

```bash
# Test production vs staging
echo "=== Production ===" > comparison.txt
gocurl -n 100 -c 10 https://api.prod.example.com >> comparison.txt

echo -e "\n=== Staging ===" >> comparison.txt
gocurl -n 100 -c 10 https://api.staging.example.com >> comparison.txt

cat comparison.txt
```

### Batch Endpoint Testing

```bash
# Test all Kubernetes ingresses
kubectl get ingress -o jsonpath='{.items[*].spec.rules[*].host}' | \
  tr ' ' '\n' | \
  sed 's|^|https://|' | \
  gocurl -L - -n 50 -c 10 -o graph
```

### Testing API Gateway Buffering

```bash
# Compare direct backend vs. through gateway
echo "=== Direct to Backend ===" > comparison.txt
gocurl --streaming --connect-to api.example.com:443:backend1.internal:443 \
  https://api.example.com/stream >> comparison.txt

echo -e "\n=== Through Gateway ===" >> comparison.txt
gocurl --streaming https://api.example.com/stream >> comparison.txt

cat comparison.txt
```

### Pre-Production DNS Testing

```bash
# Test new server IP before updating DNS
gocurl --resolve api.example.com:443:192.168.1.100 \
  -n 100 -c 10 \
  https://api.example.com

# If successful, update DNS
# dig api.example.com  # Verify DNS change
```

### Streaming Endpoint Validation

```bash
# Ensure streaming endpoint works in CI/CD
#!/bin/bash
set -e

echo "Testing SSE endpoint..."
if gocurl --expect-streaming \
          --stall-threshold 2s \
          https://api.example.com/events; then
  echo "✓ Streaming validation passed"
else
  echo "✗ Streaming validation failed"
  exit 1
fi
```

### Performance Regression Testing

```bash
# Save baseline
gocurl -n 1000 -c 50 -o json https://api.example.com > baseline.json

# Compare after changes
gocurl -n 1000 -c 50 -o json https://api.example.com > current.json

# Compare p95 values
BASELINE_P95=$(jq '.p95' baseline.json)
CURRENT_P95=$(jq '.p95' current.json)

echo "Baseline P95: ${BASELINE_P95}ms"
echo "Current P95: ${CURRENT_P95}ms"
```

### Image CDN Testing with Query Parameters

```bash
# Test different image sizes
cat > image-sizes.txt <<EOF
imwidth=1920
imwidth=1080
imwidth=480
EOF

# Test CDN with different image widths
gocurl https://cdn.example.com/photo.jpg \
  --query-params image-sizes.txt \
  -n 20 \
  -c 5 \
  --warmup 3 \
  --rps 10
```

This tests:
- 3 different image sizes
- 20 requests per size = 60 total
- First 3 requests per size excluded (warmup)
- Limited to 10 requests/second

## Output Examples

### Single Request (Table)
```
✓ Status: 200 OK
✓ Time: 245ms

┌───────────────────────────────────────────────────────────────┐
│ Performance Breakdown                                         │
├───────────────────┬──────────┬────────────┬───────────────────┤
│ METRIC            │ DURATION │ PERCENTAGE │ ASSESSMENT        │
├───────────────────┼──────────┼────────────┼───────────────────┤
│ DNS Lookup        │ 12ms     │ 4.9%       │ Excellent         │
│ TCP Connection    │ 45ms     │ 18.4%      │ Good              │
│ TLS Handshake     │ 156ms    │ 63.7%      │ Good              │
│ Server Processing │ 32ms     │ 13.1%      │ Fast              │
├───────────────────┼──────────┼────────────┼───────────────────┤
│ Total             │ 245ms    │ 100%       │ Excellent         │
└───────────────────┴──────────┴────────────┴───────────────────┘
```

### Load Test Results (Table)
```
Running load test: 100 requests with concurrency 10

=== Load Test Results ===
Total Requests: 100
Successful: 98
Failed: 2
Duration: 5.2s
Requests/sec: 19.23

┌────────────────────┐
│ Latency Statistics │
├──────────┬─────────┤
│ Min      │ 87ms    │
│ Max      │ 2.1s    │
│ Mean     │ 342ms   │
│ P50      │ 289ms   │
│ P90      │ 654ms   │
│ P95      │ 892ms   │
│ P99      │ 1.8s    │
└──────────┴─────────┘
```

### Load Test with Histogram (Graph)
```
=== Load Test Results ===
Total Requests: 100
Successful: 100
Duration: 3.5s
Requests/sec: 28.57

Latency Distribution:
       80-90ms │████████████████████████ 45 (45.0%)
     280-290ms │██████████ 18 (18.0%)
     290-300ms │████████ 15 (15.0%)
     650-660ms │████ 8 (8.0%)

Status Code Distribution:
  200 ████████████████████████████████ 95 (95.0%)
  503 ██ 5 (5.0%)
```

## Building from Source

### Using Make (Recommended)

```bash
# Build for your current platform
make build

# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux AMD64
make build-darwin   # macOS AMD64
make build-windows  # Windows AMD64

# Install to /usr/local/bin
make install

# View all available targets
make help
```

### Manual Build

```bash
# Build for your current platform
go build -o bin/gocurl ./cmd/gocurl

# Cross-platform build
GOOS=linux GOARCH=amd64 go build -o bin/gocurl-linux-amd64 ./cmd/gocurl
GOOS=darwin GOARCH=amd64 go build -o bin/gocurl-darwin-amd64 ./cmd/gocurl
GOOS=windows GOARCH=amd64 go build -o bin/gocurl-windows-amd64.exe ./cmd/gocurl

# Optimized build with version info
go build -ldflags="-s -w -X main.version=$(git describe --tags)" -o bin/gocurl ./cmd/gocurl
```

## Development

### Running Tests

```bash
# Using Make
make test

# With coverage report
make test-coverage

# Or use go directly
go test ./...
go test -v -cover ./...
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt

# Run all CI checks
make ci
```

### Project Structure
```
gocurl/
├── cmd/gocurl/           # CLI entry point
├── internal/
│   ├── app/             # Application logic
│   ├── client/          # HTTP client with tracing
│   ├── metrics/         # Statistics & analysis
│   └── output/          # Output formatters
├── docs/                # Documentation
├── .github/workflows/   # CI/CD workflows
├── Makefile            # Build automation
└── .goreleaser.yml     # Release configuration
```

## Troubleshooting

### TLS Certificate Errors
```bash
# Skip certificate verification (not recommended for production)
gocurl -k https://self-signed.example.com
```

### Timeout Issues
```bash
# Increase timeout for slow endpoints
gocurl --timeout 60s https://slow-api.example.com
```

### Too Many Open Files
```bash
# Reduce concurrency
gocurl -n 1000 -c 10 https://api.example.com  # Instead of -c 100
```

### DNS Resolution Issues
```bash
# Check DNS timing in verbose mode
gocurl -v https://api.example.com
```

## Comparison with Other Tools

| Feature | curl | gocurl | httpstat | hey |
|---------|------|--------|----------|-----|
| Single request timing | Basic | ✅ Detailed | ✅ Detailed | ❌ |
| Load testing | ❌ | ✅ | ❌ | ✅ |
| Multiple URLs | ❌ | ✅ | ❌ | ❌ |
| Histograms | ❌ | ✅ | ❌ | ✅ |
| JSON output | ❌ | ✅ | ✅ | ❌ |
| CSV export | ❌ | ✅ | ❌ | ❌ |
| Color output | ❌ | ✅ | ✅ | ❌ |
| Performance assessments | ❌ | ✅ | ❌ | ❌ |
| Streaming analysis | ❌ | ✅ | ❌ | ❌ |
| Buffering detection | ❌ | ✅ | ❌ | ❌ |
| DNS/Connection override | ✅ | ✅ | ❌ | ❌ |
| Scenario testing | ❌ | ✅ | ❌ | ❌ |
| Range requests | ❌ | ✅ | ❌ | ❌ |
| Redirect tracking | ❌ | ✅ | ❌ | ❌ |

## FAQ

**Q: How is this different from curl?**
A: gocurl focuses on performance measurement with detailed timing breakdowns, load testing, and statistical analysis. curl is a general-purpose data transfer tool.

**Q: Can I use this for production monitoring?**
A: Yes! gocurl is production-ready with JSON output for easy integration with monitoring systems.

**Q: What about HTTP/2 support?**
A: gocurl uses Go's standard HTTP client which supports HTTP/2 automatically.

**Q: Is it safe to use with self-signed certificates?**
A: Use the `-k` flag to skip verification, but this should only be used in testing environments.

**Q: How accurate are the timing measurements?**
A: Very accurate. gocurl uses Go's `httptrace` package which provides microsecond-precision timing for each phase of the request.

## CI/CD

This project uses GitHub Actions for continuous integration and deployment:

- **Build and Test**: Runs on every push and PR
  - Builds on Linux, macOS, and Windows
  - Runs full test suite with race detection
  - Linting (advisory, non-blocking)
  - Coverage reporting to Codecov

- **Releases**: Automated with GoReleaser
  - Triggered on version tags (e.g., `v1.0.0`)
  - Builds for 6 platforms (Linux/macOS/Windows × AMD64/ARM64)
  - Generates archives and checksums
  - Creates GitHub release with binaries

To create a release:
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Contributing

Contributions are welcome! Here's how to get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting:
   ```bash
   make test
   make lint
   ```
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

Please ensure:
- All tests pass
- Code follows Go conventions
- New features include tests
- Documentation is updated

## License

MIT License - See [LICENSE](LICENSE) file for details

Copyright (c) 2025 Erfi Anugrah

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) - CLI framework
- Tables powered by [go-pretty](https://github.com/jedib0t/go-pretty)
- Colors by [fatih/color](https://github.com/fatih/color)
- Releases automated with [GoReleaser](https://goreleaser.com/)

Inspired by tools like curl, httpstat, hey, and vegeta.

## Support

- 📖 [Full Documentation](docs/)
- 🐛 [Report Issues](https://github.com/erfianugrah/gocurl/issues)
- 💬 [Discussions](https://github.com/erfianugrah/gocurl/discussions)

---

**Made with ❤️ using Go**
