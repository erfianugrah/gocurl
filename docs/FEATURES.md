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

## Future Enhancements

### Potential Additions from go-perf-tester
1. **Response Headers Capture** (-H flag) - Capture specific response headers for debugging
2. **Range Request Support** (-range flag) - Test partial content delivery
3. **Effective URL Tracking** - Track redirects and final URLs
4. **Bell Curve Graphs** - Additional visualization modes
5. **Separate stdout/stderr** - JSON to stdout, stats to stderr for easier parsing

## Comparison with go-perf-tester

| Feature | go-perf-tester | gocurl | Notes |
|---------|----------------|---------|-------|
| URL List | ✅ | ✅ | Both support files and stdin |
| Histograms | ✅ | ✅ | gocurl uses 10ms buckets |
| p99.9/p99.99 | ✅ | ✅ | gocurl conditional on sample size |
| Signal Handling | ✅ | ✅ | Infrastructure in place |
| ASCII Graphs | ✅ | ✅ | gocurl has horizontal bars |
| Response Headers | ✅ | ⏳ | Planned feature |
| Range Requests | ✅ | ⏳ | Planned feature |
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
