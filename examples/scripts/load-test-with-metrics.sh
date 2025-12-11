#!/bin/bash
# Comprehensive Load Testing Script with Metrics Collection
# Tests API under various load patterns and generates detailed reports

set -e

# Configuration
API_URL="${API_URL:-https://api.example.com}"
OUTPUT_DIR="${OUTPUT_DIR:-./load-test-results-$(date +%Y%m%d-%H%M%S)}"
WARMUP_REQUESTS="${WARMUP_REQUESTS:-10}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

run_test() {
    local test_name="$1"
    local requests="$2"
    local concurrency="$3"
    local additional_flags="$4"

    log "Running: $test_name"
    log "  Requests: $requests, Concurrency: $concurrency"

    local output_file="$OUTPUT_DIR/${test_name// /-}.json"
    local csv_file="$OUTPUT_DIR/${test_name// /-}.csv"

    gocurl "$API_URL" \
        -n "$requests" \
        -c "$concurrency" \
        --warmup "$WARMUP_REQUESTS" \
        --export-csv "$csv_file" \
        -o json \
        $additional_flags \
        > "$output_file" 2>&1

    # Extract key metrics
    local p95=$(jq -r '.p95 // 0' "$output_file")
    local p99=$(jq -r '.p99 // 0' "$output_file")
    local success_rate=$(jq -r '.success_rate // 0' "$output_file")
    local rps=$(jq -r '.requests_per_second // 0' "$output_file")

    log "  Results: P95=${p95}ms, P99=${p99}ms, Success=${success_rate}%, RPS=${rps}"
    success "Completed: $test_name"
    echo ""
}

generate_report() {
    log "Generating summary report..."

    local report_file="$OUTPUT_DIR/REPORT.md"

    cat > "$report_file" << EOF
# Load Test Report
Generated: $(date)
Target: $API_URL

## Test Configuration
- Warmup Requests: $WARMUP_REQUESTS per test
- Output Directory: $OUTPUT_DIR

## Test Results

EOF

    # Aggregate results from all JSON files
    for json_file in "$OUTPUT_DIR"/*.json; do
        if [ -f "$json_file" ]; then
            local test_name=$(basename "$json_file" .json | tr '-' ' ')
            local p50=$(jq -r '.p50 // "N/A"' "$json_file")
            local p95=$(jq -r '.p95 // "N/A"' "$json_file")
            local p99=$(jq -r '.p99 // "N/A"' "$json_file")
            local success=$(jq -r '.success_rate // "N/A"' "$json_file")
            local rps=$(jq -r '.requests_per_second // "N/A"' "$json_file")

            cat >> "$report_file" << EOF
### $test_name
- **P50 Latency**: ${p50}ms
- **P95 Latency**: ${p95}ms
- **P99 Latency**: ${p99}ms
- **Success Rate**: ${success}%
- **Throughput**: ${rps} req/s

EOF
        fi
    done

    cat >> "$report_file" << EOF
## Analysis

### Performance Observations
- Review P95 and P99 latencies for tail latency issues
- Check success rates for error patterns
- Compare throughput across different concurrency levels

### Recommendations
1. If P99 >> P95: Investigate tail latency causes (timeouts, GC, etc.)
2. If success rate < 99%: Check error responses in CSV files
3. If RPS plateaus: Possible bottleneck in application or infrastructure

## Files Generated
EOF

    for file in "$OUTPUT_DIR"/*; do
        if [ -f "$file" ]; then
            echo "- $(basename "$file")" >> "$report_file"
        fi
    done

    success "Report generated: $report_file"
}

main() {
    log "Starting comprehensive load test"
    log "Target: $API_URL"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    log "Output directory: $OUTPUT_DIR"
    echo ""

    # Test Suite
    run_test "Baseline (Low Load)" 100 10 ""
    run_test "Moderate Load" 500 25 ""
    run_test "High Concurrency" 1000 50 ""
    run_test "Stress Test" 2000 100 ""
    run_test "Rate Limited" 1000 50 "--rps 100"
    run_test "Gradual Ramp-up" 2000 100 "--ramp-up 30s"
    run_test "Sustained Load" 5000 50 "--rps 50"

    # Generate report
    echo ""
    generate_report

    # Summary
    echo ""
    log "Load test complete!"
    success "Total tests: 7"
    success "Results saved to: $OUTPUT_DIR"
    log "View report: cat $OUTPUT_DIR/REPORT.md"
}

# Handle interrupts
trap 'log "Test interrupted. Partial results saved to: $OUTPUT_DIR"; exit 1' INT TERM

main
