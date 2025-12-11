# gocurl Examples

This directory contains production-ready examples and templates for using gocurl effectively.

## 📁 Directory Structure

```
examples/
├── scenarios/          # YAML scenario files for multi-step workflows
├── scripts/            # Bash scripts for monitoring and load testing
└── README.md          # This file
```

## 🎬 Scenarios

### E-Commerce Flow (`scenarios/e-commerce-flow.yaml`)
Complete shopping user journey with 7 steps:
- Browse homepage → Search → View product → Add to cart → View cart → Checkout → Confirmation

**Usage:**
```bash
gocurl --scenario examples/scenarios/e-commerce-flow.yaml
```

**Features demonstrated:**
- Variable extraction and substitution
- Session ID tracking across requests
- POST requests with JSON payloads
- Multi-step workflow validation

---

### API Authentication Flow (`scenarios/api-auth-flow.yaml`)
JWT authentication workflow with token refresh:
- Health check → Login → Authenticated requests → Token refresh → Logout

**Usage:**
```bash
gocurl --scenario examples/scenarios/api-auth-flow.yaml
```

**Features demonstrated:**
- JWT token extraction from JSON responses
- Authorization header usage
- Token refresh flow
- Multiple HTTP methods (GET, POST, PUT)

---

### CDN Validation (`scenarios/cdn-validation.yaml`)
Test CDN caching behavior and performance:
- Cold cache → Warm cache → Cache validation → Conditional requests

**Usage:**
```bash
gocurl --scenario examples/scenarios/cdn-validation.yaml
```

**Features demonstrated:**
- Cache-Control testing
- ETag validation
- Conditional requests (If-None-Match, If-Modified-Since)
- CDN cache bypass

---

## 🔧 Scripts

### Production Monitoring (`scripts/monitor-api-health.sh`)
Continuously monitor API health with automatic alerting.

**Usage:**
```bash
# Basic usage
API_URL=https://api.example.com ./examples/scripts/monitor-api-health.sh

# With custom thresholds
API_URL=https://api.example.com \
P95_THRESHOLD=300 \
ERROR_THRESHOLD=2 \
CHECK_INTERVAL=30 \
./examples/scripts/monitor-api-health.sh

# With Slack alerts
API_URL=https://api.example.com \
ALERT_WEBHOOK=https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
./examples/scripts/monitor-api-health.sh
```

**Features:**
- Continuous health checks with configurable intervals
- P95 latency threshold monitoring
- Error rate tracking
- Consecutive failure detection
- Slack/Discord webhook alerts
- Graceful shutdown on SIGINT/SIGTERM

**Environment Variables:**
- `API_URL` - API endpoint to monitor (default: https://api.example.com/health)
- `CHECK_INTERVAL` - Seconds between checks (default: 60)
- `P95_THRESHOLD` - P95 latency threshold in ms (default: 500)
- `ERROR_THRESHOLD` - Error rate threshold in % (default: 5)
- `ALERT_WEBHOOK` - Webhook URL for alerts (optional)

---

### Comprehensive Load Testing (`scripts/load-test-with-metrics.sh`)
Run a suite of load tests and generate detailed reports.

**Usage:**
```bash
# Basic usage
API_URL=https://api.example.com ./examples/scripts/load-test-with-metrics.sh

# With custom configuration
API_URL=https://api.example.com \
OUTPUT_DIR=./my-test-results \
WARMUP_REQUESTS=20 \
./examples/scripts/load-test-with-metrics.sh
```

**Test Suite Includes:**
1. **Baseline** - 100 req, 10 concurrent
2. **Moderate Load** - 500 req, 25 concurrent
3. **High Concurrency** - 1000 req, 50 concurrent
4. **Stress Test** - 2000 req, 100 concurrent
5. **Rate Limited** - 1000 req, 50 concurrent, 100 RPS
6. **Gradual Ramp-up** - 2000 req, 100 concurrent, 30s ramp-up
7. **Sustained Load** - 5000 req, 50 concurrent, 50 RPS

**Generated Files:**
- JSON files for each test with full metrics
- CSV files with individual request data
- `REPORT.md` - Markdown summary with analysis

**Environment Variables:**
- `API_URL` - API endpoint to test (required)
- `OUTPUT_DIR` - Results directory (default: ./load-test-results-TIMESTAMP)
- `WARMUP_REQUESTS` - Warmup requests per test (default: 10)

---

## 📊 Analyzing Results

### View JSON Metrics
```bash
cat output.json | jq '{p95, p99, success_rate, requests_per_second}'
```

### Analyze CSV Data
```bash
# Calculate average latency
awk -F',' 'NR>1 {sum+=$9; count++} END {print "Avg:", sum/count "ms"}' results.csv

# Find slowest requests
awk -F',' 'NR>1 {print $9, $2}' results.csv | sort -rn | head -10

# Count errors
awk -F',' 'NR>1 && $12!="" {print $12}' results.csv | sort | uniq -c
```

### Compare Test Results
```bash
# Compare P95 across tests
for file in load-test-results-*/*.json; do
  p95=$(jq -r '.p95' "$file")
  echo "$(basename $file .json): ${p95}ms"
done
```

---

## 🚀 Best Practices

### Scenario Testing
1. **Start Simple** - Begin with 2-3 steps, expand as needed
2. **Use Realistic Data** - Extract actual values from responses
3. **Validate Responses** - Add `expect` blocks for critical assertions
4. **Handle Errors Gracefully** - Use `status_codes` array for flexible validation

### Load Testing
1. **Always Use Warmup** - Exclude cold-start effects from metrics
2. **Monitor During Tests** - Watch server metrics alongside gocurl output
3. **Gradual Increase** - Use `--ramp-up` to avoid thundering herd
4. **Rate Limit** - Use `--rps` for controlled, realistic load

### Production Monitoring
1. **Set Appropriate Thresholds** - Based on your SLAs
2. **Monitor Trends** - P95/P99 trends matter more than single values
3. **Alert on Patterns** - Consecutive failures, not single blips
4. **Export to Metrics Systems** - Feed gocurl JSON to Prometheus, Datadog, etc.

---

## 🔗 Related Documentation

- [Full Feature Documentation](../docs/FEATURES.md)
- [Main README](../README.md)
- [Scenario Testing Guide](../docs/FEATURES.md#feature-17-scenario-testing)

---

## 💡 Tips

- Use `-q` flag for quiet output in automated scripts
- Use `-o json` for machine-readable output
- Combine `--export-csv` with load tests for deep analysis
- Run scenarios in CI/CD pipelines for regression testing
- Use `--timeout` to detect slow responses early

---

## 🐛 Troubleshooting

### Scenario fails immediately
- Check YAML syntax with `yamllint scenarios/your-file.yaml`
- Verify URL is accessible: `curl -I <url>`
- Check expectations aren't too strict

### Load test shows inconsistent results
- Increase warmup requests: `--warmup 20`
- Check network stability
- Verify server isn't throttling/rate-limiting

### CSV file is too large
- Reduce number of requests: `-n 100` instead of `-n 10000`
- Test with fewer URLs
- Use multiple smaller tests instead of one large test

---

**Need help?** Open an issue at https://github.com/erfianugrah/gocurl/issues
