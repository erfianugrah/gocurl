# New Features from go-perf-tester

## Features Incorporated

### 1. URL List File Input (-L flag)
Test multiple URLs from a file, one URL per line. Supports comments (lines starting with #).

```bash
# Create URLs file
cat > urls.txt <<EOF
# Production endpoints
https://api.example.com/users
https://api.example.com/products
https://api.example.com/orders
EOF

# Test all URLs
gocurl -L urls.txt -n 10 -c 5
```

### 2. Stdin Support
Pipe URLs directly into gocurl for flexible workflow integration.

```bash
# From a file
cat urls.txt | gocurl -L - -n 10 -c 5

# From command substitution
echo -e "https://api1.example.com\nhttps://api2.example.com" | gocurl -L - -n 5

# From another command
kubectl get ingress -o jsonpath='{.items[*].spec.rules[*].host}' | \
  tr ' ' '\n' | \
  sed 's|^|https://|' | \
  gocurl -L - -n 10
```

### 3. ASCII Graphs/Histogram Visualization (-o graph)
Visual representation of latency distribution with histogram and statistics.

```bash
# Graph output with histogram
gocurl -o graph -n 100 -c 10 https://api.example.com
```

**Output includes:**
- Latency histogram with ASCII bars
- Bucket-based distribution showing request clustering
- Visual identification of performance patterns
- Status code distribution with bars

**Example output:**
```
Latency Distribution:
       80-90ms │████████████████████████████████████ 40 (80.0%)
     390-400ms │██████ 5 (10.0%)
     400-410ms │██████ 5 (10.0%)

Status Code Distribution:
  200 ███████████████████████████████████████ 45 (90.0%)
  503 ████ 5 (10.0%)
```

### 4. Extended Percentiles (p90, p99.9, p99.99)
More detailed statistical analysis for large test runs.

- **p90**: Added for better understanding of "good enough" performance
- **p99.9**: Calculated when ≥1000 requests (catches rare outliers)
- **p99.99**: Calculated when ≥10,000 requests (ultra-rare edge cases)

```bash
# Large test to see extended percentiles
gocurl -n 10000 -c 50 https://api.example.com
```

### 5. Histogram Bucketing
Automatic histogram creation with 10ms buckets for latency distribution analysis.

- Buckets: 0-10ms, 10-20ms, 20-30ms, etc.
- Shows clustering of request latencies
- Helps identify performance modes (fast/slow requests)

### 6. Multi-URL Load Testing
Test multiple endpoints simultaneously with consolidated statistics.

```bash
# Test 3 URLs, 10 requests each = 30 total
gocurl -L urls.txt -n 10 -c 5

# Output shows:
# "Running load test: 3 URLs x 10 requests = 30 total requests with concurrency 5"
```

### 7. Signal Handling (Infrastructure Ready)
Graceful shutdown handling for SIGINT/SIGTERM signals.

- Ctrl+C during long tests triggers graceful shutdown
- 5-second timeout before forced exit
- Signal handling infrastructure in place at `internal/app/signals.go`

## Usage Examples

### Batch Testing Multiple Endpoints
```bash
# Create test suite
cat > api-test.txt <<EOF
# Core APIs
https://api.example.com/health
https://api.example.com/metrics
https://api.example.com/status

# Feature APIs
https://api.example.com/users/1
https://api.example.com/products/search
EOF

# Run comprehensive test
gocurl -L api-test.txt -n 50 -c 10 -o graph
```

### CI/CD Integration
```bash
# Test endpoints from deployment
kubectl get ingress -o json | \
  jq -r '.items[].spec.rules[].host' | \
  sed 's|^|https://|' | \
  gocurl -L - -n 100 -c 20 -o json > perf-results.json

# Parse results
cat perf-results.json | jq '.p95'
```

### Performance Monitoring
```bash
# Create monitoring script
#!/bin/bash
THRESHOLD_MS=500

while true; do
  p95=$(gocurl -n 100 -c 10 -o json https://api.example.com | jq '.p95')

  if [ "$p95" -gt "$THRESHOLD_MS" ]; then
    echo "ALERT: P95 latency ${p95}ms exceeds threshold ${THRESHOLD_MS}ms"
    # Send alert
  fi

  sleep 60
done
```

### Comparative Analysis
```bash
# Compare environments
echo "Production:" > comparison.txt
gocurl -n 100 -c 10 https://api.prod.example.com >> comparison.txt

echo "\nStaging:" >> comparison.txt
gocurl -n 100 -c 10 https://api.staging.example.com >> comparison.txt

# Review differences
cat comparison.txt
```

## Implementation Details

### URL Reader
- Location: `internal/app/urls.go`
- Supports: files, stdin, comments
- Automatic comment stripping (lines starting with #)
- Empty line handling

### Graph Formatter
- Location: `internal/output/graph.go`
- Histogram with configurable buckets
- Status code distribution bars
- Color-coded output

### Enhanced Metrics
- Location: `internal/metrics/collector.go`
- Automatic histogram creation
- Conditional extended percentiles (based on sample size)
- Memory-efficient bucketing

### Multi-URL Support
- Location: `internal/app/app.go`
- Job queue per URL
- Consolidated statistics across all URLs
- Clear reporting of total requests

## Performance Characteristics

### Memory Usage
- Histogram buckets: O(log n) space complexity
- URL list: O(urls) memory
- Request results: O(requests × urls) memory

### Throughput
- Multi-URL testing maintains high concurrency
- Worker pool efficiently distributes across all URLs
- No serialization between different URLs

### Accuracy
- Extended percentiles use linear interpolation
- Histogram buckets provide visual grouping
- Statistics calculated from raw measurements

### 8. URL Parameterization (--query-params)
Expand base URLs with query parameter variants for comprehensive testing of different parameter combinations.

```bash
# Create query params file
cat > params.txt <<EOF
imwidth=1920
imwidth=1080
imwidth=480
EOF

# Test all variants
gocurl https://cdn.example.com/image.jpg --query-params params.txt -n 10 -c 5
```

**Features:**
- Reads query parameters from file (one per line)
- Supports comments (`#`) and empty lines
- Handles URLs with existing query strings (appends with `&`)
- Creates Cartesian product: N URLs × M params = N×M total URLs
- Supports complex multi-parameter strings: `limit=10&offset=20`

**Example output:**
```
Expanded 1 base URL(s) × 3 query param variant(s) = 3 total URL(s)
```

**Generated URLs:**
- `https://cdn.example.com/image.jpg?imwidth=1920`
- `https://cdn.example.com/image.jpg?imwidth=1080`
- `https://cdn.example.com/image.jpg?imwidth=480`

### 9. Warmup Phase (--warmup)
Exclude initial requests from metrics to eliminate cold-start effects like JIT compilation, connection pool establishment, and cache warming.

```bash
# Run 100 requests but exclude first 10 from metrics
gocurl https://api.example.com -n 100 -c 10 --warmup 10
```

**Features:**
- Skips first N requests per URL from metrics collection
- Requests are still made (system is warmed up)
- Useful for accurate performance measurements
- Works with single and multiple URLs
- Validated: warmup count must be less than total requests

**Example output:**
```
Running load test: 1 URLs x 100 requests = 100 total requests with concurrency 10
Warmup: 10 requests per URL (10 total warmup) - not included in metrics

=== Load Test Results ===
Total Requests: 90  ← Only measured requests
```

### 10. Rate Limiting (--rps)
Control request rate for realistic load testing and to prevent overwhelming target servers.

```bash
# Limit to 50 requests per second
gocurl https://api.example.com -n 1000 -c 10 --rps 50
```

**Features:**
- Ticker-based rate limiting across all workers
- Prevents overwhelming target servers
- Useful for realistic load simulation
- `--rps 0` = unlimited (default)
- Accurate to ~95-98% of target rate

**Example output:**
```
Running load test: 1 URLs x 1000 requests = 1000 total requests with concurrency 10
Rate limit: 50 requests/second

Duration: 20.5s
Requests/sec: 48.78  ← Close to 50 RPS target
```

### 11. CSV Export (--export-csv)
Export individual request timing data to CSV for detailed analysis in spreadsheet tools or data pipelines.

```bash
# Export all request timings to CSV
gocurl https://api.example.com -n 100 -c 10 --export-csv results.csv
```

**Features:**
- Exports every individual request with full timing breakdown
- 12 columns: timestamp, url, status_code, dns_lookup_ms, tcp_connection_ms, tls_handshake_ms, server_processing_ms, content_transfer_ms, total_ms, response_size_bytes, connection_reused, error
- Warmup requests are excluded from CSV (consistent with metrics)
- Works with multiple URLs (all requests in one CSV)
- Works with all load testing features (rate limiting, warmup, concurrency)

**CSV Format:**
```csv
timestamp,url,status_code,dns_lookup_ms,tcp_connection_ms,tls_handshake_ms,server_processing_ms,content_transfer_ms,total_ms,response_size_bytes,connection_reused,error
2025-12-11T08:31:33+01:00,https://httpbin.org/get,200,20.82,91.82,200.08,91.91,0.05,404.76,162,false,
2025-12-11T08:31:33+01:00,https://httpbin.org/get,200,0.00,0.00,0.00,183.03,0.02,183.15,162,true,
```

**Example use cases:**
```bash
# Basic CSV export
gocurl https://api.example.com -n 100 -c 10 --export-csv results.csv

# With warmup (first 10 requests excluded from CSV)
gocurl https://api.example.com -n 100 -c 10 --warmup 10 --export-csv results.csv

# Multiple URLs with query params
gocurl https://cdn.example.com/image.jpg --query-params sizes.txt -n 50 --export-csv cdn-test.csv

# Rate-limited test with export
gocurl https://api.example.com -n 1000 -c 20 --rps 100 --export-csv load-test.csv
```

**Analysis with standard tools:**
```bash
# Import into spreadsheet (Excel, Google Sheets, LibreOffice)
# - Create pivot tables
# - Generate custom charts
# - Statistical analysis

# Command-line analysis with awk
awk -F',' 'NR>1 {sum+=$9; count++} END {print "Avg:", sum/count "ms"}' results.csv

# Find slowest requests
awk -F',' 'NR>1 {print $9, $2}' results.csv | sort -rn | head -10

# Count errors
awk -F',' 'NR>1 && $12!="" {print}' results.csv | wc -l
```

**Example output:**
```
=== Load Test Results ===
Total Requests: 100
Successful: 98
Failed: 2
Duration: 10.5s
Requests/sec: 9.52

[statistics table...]

Exported 100 requests to results.csv
```

### 12. Request Body Templates (--data with variables)
Dynamic request body generation using template variables for realistic load testing with unique data per request.

```bash
# Basic template with sequence number
gocurl https://api.example.com/users -X POST \
  -H "Content-Type: application/json" \
  --data '{"user_id": {{seq}}, "timestamp": {{timestamp}}}' \
  -n 100 -c 10
```

**Supported Variables:**
- `{{seq}}` - Sequential number starting from 1 (unique per request)
- `{{uuid}}` - Random UUID v4 (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- `{{timestamp}}` - Unix timestamp in seconds (e.g., `1702345678`)
- `{{timestamp_ms}}` - Unix timestamp in milliseconds (e.g., `1702345678901`)
- `{{random}}` - Random integer 0-999999 (e.g., `742891`)
- `{{url_index}}` - Index of the URL being tested (0-based, useful with multiple URLs)

**Features:**
- Automatic variable substitution in `--data` parameter
- Each request gets unique values (timestamps, UUIDs, random numbers)
- Sequential numbers help correlate requests in logs
- Works with all load testing features (concurrency, rate limiting, warmup)
- No external files needed - templates inline in command

**Example use cases:**
```bash
# Create users with unique IDs
gocurl https://api.example.com/users -X POST \
  -H "Content-Type: application/json" \
  --data '{"id": "{{uuid}}", "name": "user_{{seq}}", "created": {{timestamp}}}' \
  -n 50 -c 5

# Simulate realistic event stream
gocurl https://api.example.com/events -X POST \
  -H "Content-Type: application/json" \
  --data '{"event_id": "{{uuid}}", "seq": {{seq}}, "timestamp": {{timestamp_ms}}, "random_value": {{random}}}' \
  -n 1000 -c 20 --rps 50

# Load test with unique request IDs for tracing
gocurl https://api.example.com/process -X POST \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: {{uuid}}" \
  --data '{"request_num": {{seq}}, "data": "payload_{{random}}"}' \
  -n 500 -c 25

# Multiple URLs with URL-specific data
cat > urls.txt <<EOF
https://api1.example.com/endpoint
https://api2.example.com/endpoint
EOF

gocurl -L urls.txt -X POST \
  -H "Content-Type: application/json" \
  --data '{"target": {{url_index}}, "seq": {{seq}}, "uuid": "{{uuid}}"}' \
  -n 10 -c 3
```

**Example output data:**
```json
// Request 1:
{"user_id": 1, "uuid": "550e8400-e29b-41d4-a716-446655440000", "timestamp": 1702345678}

// Request 2:
{"user_id": 2, "uuid": "a1b2c3d4-e5f6-4789-a012-3456789abcde", "timestamp": 1702345679}

// Request 3:
{"user_id": 3, "uuid": "f9e8d7c6-b5a4-3210-9876-543210fedcba", "timestamp": 1702345679}
```

### 13. Ramp-up Configuration (--ramp-up)
Gradually increase concurrency over time to simulate realistic load patterns and observe system behavior under increasing pressure.

```bash
# Ramp up from 1 to 50 workers over 30 seconds
gocurl https://api.example.com -n 1000 -c 50 --ramp-up 30s
```

**Features:**
- Linear ramp-up: starts with 1 worker, gradually adds workers until reaching target concurrency
- Useful for testing autoscaling behavior
- Helps warm up connection pools gradually
- Identifies performance thresholds (when does the system start degrading?)
- Works with all load testing features (rate limiting, warmup, CSV export)

**How it works:**
- With `--ramp-up 30s -c 10`, workers are added every 3.33 seconds
- Worker 1 starts immediately
- Worker 2 starts after 3.33s
- Worker 3 starts after 6.66s
- ...
- Worker 10 starts after 30s

**Example output:**
```
Running load test: 1 URLs x 1000 requests = 1000 total requests with concurrency 50
Ramp-up: gradually increasing from 1 to 50 workers over 30s

=== Load Test Results ===
Total Requests: 1000
Successful: 998
Failed: 2
Duration: 65.2s
Requests/sec: 15.31
```

**Example use cases:**
```bash
# Test autoscaling response time
gocurl https://api.example.com -n 5000 -c 100 --ramp-up 1m

# Gradual load increase with rate limiting
gocurl https://api.example.com -n 10000 -c 200 --ramp-up 2m --rps 500

# Find performance breaking point
gocurl https://api.example.com -n 20000 -c 500 --ramp-up 5m --export-csv ramp-test.csv

# Test with warmup and ramp-up
gocurl https://api.example.com -n 5000 -c 100 --warmup 50 --ramp-up 30s

# Combined: ramp-up + rate limit + templates + CSV export
gocurl https://api.example.com/users -X POST \
  -H "Content-Type: application/json" \
  --data '{"user_id": {{seq}}, "uuid": "{{uuid}}"}' \
  -n 10000 -c 100 --ramp-up 1m --rps 200 --export-csv load-test.csv
```

**When to use ramp-up:**
- **Autoscaling Testing**: See how quickly your infrastructure responds to increasing load
- **Connection Pool Warmup**: Gradually establish connections instead of instant burst
- **Performance Profiling**: Identify at what concurrency level performance degrades
- **Realistic Load Patterns**: Most real-world traffic increases gradually, not instantly
- **Kubernetes HPA Testing**: Verify Horizontal Pod Autoscaler responds correctly

## Feature 14: Selective Header Capture

**Flag**: `--capture-header <header-name>` (repeatable)

Capture specific response headers without including all headers (unlike `-i` which captures everything).

**Use cases:**
- Monitor cache behavior (Cache-Control, ETag, Last-Modified)
- Track custom application headers
- Debug CDN configurations (X-Cache, X-Cache-Status)
- Export specific headers to CSV for analysis

**Example usage:**
```bash
# Capture specific headers
gocurl https://cdn.example.com/image.jpg \
  --capture-header Cache-Control \
  --capture-header ETag \
  --capture-header X-Cache \
  -o json

# Output includes only specified headers:
{
  "response_headers": {
    "Cache-Control": "max-age=3600",
    "ETag": "abc123xyz",
    "X-Cache": "HIT"
  }
}

# Load testing with header capture
gocurl https://api.example.com \
  -n 1000 -c 50 \
  --capture-header X-RateLimit-Remaining \
  --export-csv rate-limits.csv
```

## Feature 15: Effective URL Tracking

**Fields**: `effective_url`, `redirect_count` (in JSON output)

Automatically tracks redirect chains and captures the final destination URL.

**Use cases:**
- Verify redirect behavior (301/302/303/307/308)
- Measure redirect overhead in timing
- Debug redirect loops
- Track URL shorteners and affiliate links

**Example usage:**
```bash
# Track redirects
gocurl https://httpbin.org/redirect/3 -o json | jq '{url, effective_url, redirect_count}'

# Output:
{
  "url": "https://httpbin.org/redirect/3",
  "effective_url": "https://httpbin.org/get",
  "redirect_count": 1
}

# Test redirect performance
gocurl https://short.link/abc -n 100 -c 10 -o json | \
  jq '[.redirect_count, .total] | @csv'
```

**Captured automatically:**
- Original URL
- Final effective URL after all redirects
- Approximate redirect count
- Total time including redirects

## Feature 16: Range Request Support

**Flag**: `--range <byte-range>`

Request partial content using HTTP Range header for testing CDN byte-range serving.

**Use cases:**
- Test CDN partial content support (HTTP 206)
- Validate resume capabilities
- Test video/audio streaming byte-range serving
- Measure partial vs full download performance

**Example usage:**
```bash
# Request first 1KB
gocurl https://kernel.org/pub/linux/kernel/v6.x/ChangeLog-6.1.1 \
  --range 'bytes=0-1023' \
  -o json

# Output shows partial content:
{
  "status_code": 206,
  "content_range": "bytes 0-1023/32332",
  "response_size": 1024,
  "total": 276
}

# Test range serving performance
gocurl https://cdn.example.com/large-file.bin \
  --range 'bytes=0-1048575' \
  -n 100 -c 10

# Request from offset (resume simulation)
gocurl https://download.example.com/file.iso \
  --range 'bytes=1073741824-' \
  -o json
```

**Range header formats:**
- `bytes=0-1023` - First 1024 bytes
- `bytes=1024-2047` - Next 1024 bytes
- `bytes=1024-` - From byte 1024 to end
- `bytes=-1024` - Last 1024 bytes

**Captured data:**
- Status code (206 for partial content, 200 if not supported)
- `Content-Range` header value
- Actual response size vs requested range

## Feature 17: Scenario Testing

**Flag**: `--scenario <yaml-file>` or `-s <yaml-file>`

Execute multi-step HTTP workflows with variable extraction, session management, and expectations.

**Use cases:**
- Test complete user journeys (login → browse → checkout)
- API integration testing with token flows
- Complex business workflow validation
- Automated API acceptance tests

**Scenario file format (YAML):**
```yaml
name: "User Authentication Flow"
description: "Test login and authenticated requests"

config:
  timeout: "30s"
  insecure: false

steps:
  - name: "Login"
    url: "https://api.example.com/auth/login"
    method: POST
    headers:
      Content-Type: "application/json"
    body: '{"username": "testuser", "password": "pass123"}'
    extract:
      token: "access_token"  # JSONPath extraction
      user_id: "user.id"
    expect:
      status_code: 200
      body_contains: "access_token"

  - name: "Get Profile"
    url: "https://api.example.com/users/${user_id}"
    method: GET
    headers:
      Authorization: "Bearer ${token}"
    expect:
      status_code: 200

  - name: "Update Profile"
    url: "https://api.example.com/users/${user_id}"
    method: PUT
    headers:
      Authorization: "Bearer ${token}"
      Content-Type: "application/json"
    body: '{"name": "Updated Name"}'
    expect:
      status_codes: [200, 204]
```

**Features:**
- **Variable Extraction**: Extract values from JSON responses using JSONPath
- **Variable Substitution**: Use `${variable}` in URLs, headers, and body
- **Session Management**: Automatic cookie handling across steps
- **Expectations**: Validate status codes, headers, and body content
- **Detailed Results**: Per-step timing, status, and error reporting

**Example execution:**
```bash
# Run scenario
gocurl --scenario login-flow.yaml

# Output:
=== Running Scenario: User Authentication Flow ===
Description: Test login and authenticated requests

=== Scenario Results ===

✓ Step 1: Login
  URL: https://api.example.com/auth/login
  Method: POST
  Status: 200
  Duration: 234ms
  Size: 156 bytes
  Extracted variables:
    token = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
    user_id = 12345

✓ Step 2: Get Profile
  URL: https://api.example.com/users/12345
  Method: GET
  Status: 200
  Duration: 89ms
  Size: 512 bytes

✓ Step 3: Update Profile
  URL: https://api.example.com/users/12345
  Method: PUT
  Status: 204
  Duration: 156ms
  Size: 0 bytes

Total Steps: 3
Successful: 3
Failed: 0
```

## Comparison with go-perf-tester

| Feature | go-perf-tester | gocurl | Notes |
|---------|----------------|---------|-------|
| URL List | ✅ | ✅ | Both support files and stdin |
| Histograms | ✅ | ✅ | gocurl uses 10ms buckets |
| p99.9/p99.99 | ✅ | ✅ | gocurl conditional on sample size |
| Signal Handling | ✅ | ✅ | Infrastructure in place |
| ASCII Graphs | ✅ | ✅ | gocurl has horizontal bars |
| Response Headers | ✅ | ✅ | gocurl has selective capture |
| Range Requests | ✅ | ✅ | Both support byte-ranges |
| Scenario Testing | ❌ | ✅ | Multi-step workflows |
| Cobra CLI | ❌ | ✅ | Better flag handling |
| Table Output | ❌ | ✅ | Beautiful formatted tables |
| Color Coding | ❌ | ✅ | Performance assessments |
| JSON Output | ✅ | ✅ | Both support JSON |

## Migration from go-perf-tester

```bash
# go-perf-tester command
go-perf-tester -c 10 -n 100 -L urls.txt -graphs

# Equivalent gocurl command
gocurl -c 10 -n 100 -L urls.txt -o graph
```

Key differences:
- gocurl uses `-o graph` instead of `-graphs` flag
- gocurl provides multiple output formats (table/json/graph)
- gocurl uses Cobra for better help/completion
- gocurl has colored assessments in table mode
