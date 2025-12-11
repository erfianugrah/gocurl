#!/bin/bash
# Production API Health Monitoring Script
# Continuously monitor API health with alerts on failures

set -e

# Configuration
API_URL="${API_URL:-https://api.example.com/health}"
CHECK_INTERVAL="${CHECK_INTERVAL:-60}"  # seconds
P95_THRESHOLD="${P95_THRESHOLD:-500}"   # milliseconds
ERROR_THRESHOLD="${ERROR_THRESHOLD:-5}" # percent
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"      # Slack/Discord webhook URL

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# State tracking
CONSECUTIVE_FAILURES=0
MAX_CONSECUTIVE_FAILURES=3

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

send_alert() {
    local message="$1"
    local severity="$2"

    log "${RED}ALERT: ${message}${NC}"

    if [ -n "$ALERT_WEBHOOK" ]; then
        curl -X POST "$ALERT_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"text\": \"[${severity}] ${message}\"}" \
            --silent --output /dev/null || true
    fi
}

check_health() {
    log "Checking API health: $API_URL"

    # Run health check with gocurl
    result=$(gocurl "$API_URL" \
        -n 10 \
        -c 5 \
        --timeout 10s \
        -o json 2>&1)

    # Parse results
    status_code=$(echo "$result" | jq -r '.status_code // 0')
    p95=$(echo "$result" | jq -r '.p95 // 0')
    p99=$(echo "$result" | jq -r '.p99 // 0')
    success_rate=$(echo "$result" | jq -r '.success_rate // 0')
    total=$(echo "$result" | jq -r '.total // 0')

    # Calculate error rate
    error_rate=$(echo "scale=2; (100 - $success_rate)" | bc)

    log "Results: Status=$status_code, P95=${p95}ms, P99=${p99}ms, Errors=${error_rate}%"

    # Check thresholds
    if [ "$status_code" != "200" ]; then
        CONSECUTIVE_FAILURES=$((CONSECUTIVE_FAILURES + 1))
        send_alert "API returned status $status_code" "ERROR"
    elif (( $(echo "$p95 > $P95_THRESHOLD" | bc -l) )); then
        send_alert "P95 latency ${p95}ms exceeds threshold ${P95_THRESHOLD}ms" "WARNING"
    elif (( $(echo "$error_rate > $ERROR_THRESHOLD" | bc -l) )); then
        send_alert "Error rate ${error_rate}% exceeds threshold ${ERROR_THRESHOLD}%" "WARNING"
    else
        CONSECUTIVE_FAILURES=0
        log "${GREEN}✓ API health check passed${NC}"
    fi

    # Check consecutive failures
    if [ $CONSECUTIVE_FAILURES -ge $MAX_CONSECUTIVE_FAILURES ]; then
        send_alert "API has failed $CONSECUTIVE_FAILURES consecutive health checks!" "CRITICAL"
    fi
}

main() {
    log "Starting API health monitor"
    log "Target: $API_URL"
    log "Check interval: ${CHECK_INTERVAL}s"
    log "P95 threshold: ${P95_THRESHOLD}ms"
    log "Error threshold: ${ERROR_THRESHOLD}%"
    echo ""

    while true; do
        check_health
        echo ""
        sleep "$CHECK_INTERVAL"
    done
}

# Handle SIGINT/SIGTERM gracefully
trap 'log "Shutting down monitor..."; exit 0' INT TERM

main
