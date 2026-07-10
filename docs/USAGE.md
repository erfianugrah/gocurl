# gocurl Usage Reference

Authoritative reference for every flag, how modes are selected, which flags
combine (and which do not), the output formats, and report/automation recipes.
For a gentle introduction see the [README](../README.md); for the change history
see the [CHANGELOG](../CHANGELOG.md).

## Table of contents

- [Execution model (single vs load)](#execution-model-single-vs-load)
- [Flag reference](#flag-reference)
- [Combination rules and gotchas](#combination-rules-and-gotchas)
- [Output formats](#output-formats)
- [Units: milliseconds vs seconds](#units-milliseconds-vs-seconds)
- [Reports](#reports)
- [Automation recipes](#automation-recipes)
- [Exit codes](#exit-codes)

## Execution model (single vs load)

gocurl has two internal paths, selected automatically:

| Condition | Path | Behavior |
|-----------|------|----------|
| `-n 1` **and** exactly one URL | single-request | Keep-alive disabled, so DNS/TCP/TLS are measured fresh. Emits a per-request timing breakdown. Streaming analysis is available here. |
| `-n > 1` **or** more than one URL | load-test | Connection pooling enabled. Emits aggregate `Stats` (percentiles, RPS, status distribution). |

Consequences:

- A single request reports **per-phase** timings (DNS/TCP/TLS/server/transfer).
  A load test reports **percentiles** only - no per-phase split, because phases
  vary per request and connections are reused (so DNS/TCP/TLS are ~0 after the
  first request anyway).
- `--warmup`, `--rps`, `--ramp-up`, `--export-csv` only apply to the load path.
- `--streaming` / `--expect-streaming` only apply to the single-request path
  (see [Combination rules](#combination-rules-and-gotchas)).

## Flag reference

### Global

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Output format: `table`, `json`, `graph`, `prom` |
| `--verbose` | `-v` | `false` | Extra detail (TLS info, per-chunk streaming stats) |
| `--quiet` | `-q` | `false` | Suppress the stderr header/progress (results still print to stdout) |
| `--no-color` | | `false` | Disable ANSI color |

### Request

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--method` | `-X` | `GET` | HTTP method |
| `--header` | `-H` | | Custom header `Key: Value` (repeatable) |
| `--data` | | | Request body (supports template variables - see below) |
| `--timeout` | | `30s` | Per-request timeout (Go duration) |
| `--insecure` | `-k` | `false` | Skip TLS certificate verification |
| `--range` | | | `Range` header, e.g. `bytes=0-1023` (expects HTTP 206) |

### Load testing

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--requests` | `-n` | `1` | Requests **per URL** |
| `--concurrency` | `-c` | `1` | Concurrent workers |
| `--warmup` | | `0` | Requests per URL to run but exclude from metrics (cold-start elimination) |
| `--rps` | | `0` | Global rate limit in requests/sec (`0` = unlimited) |
| `--ramp-up` | | | Ramp concurrency 1 to `-c` over this duration, e.g. `30s`, `1m` |
| `--export-csv` | | | Write per-request rows to a CSV file |

### Input sources

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--url-list` | `-L` | | File of URLs, one per line (`-` = stdin). `#` comments and blank lines skipped |
| `--stdin` | | `false` | Read URLs from stdin (equivalent to `-L -`) |
| `--query-params` | | | File of query-param strings; Cartesian-expanded against each base URL |

### Response inspection

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--include` | `-i` | `false` | Include response headers in output |
| `--head` | `-I` | `false` | HEAD request |
| `--show-body` | | `false` | Print response body |
| `--show-error` | | `false` | Print body only for 4xx/5xx |
| `--capture-header` | | | Capture a specific response header (repeatable), e.g. `--capture-header ETag` |

### Connection control

| Flag | Default | Format | Description |
|------|---------|--------|-------------|
| `--resolve` | | `host:port:addr` | Override DNS for a host:port (like a temporary `/etc/hosts`) |
| `--connect-to` | | `host1:port1:host2:port2` | Dial a different host:port while keeping the original Host header + SNI |

### Streaming (single-request only)

| Flag | Default | Description |
|------|---------|-------------|
| `--streaming` | `false` | Chunk-level timing + buffering analysis |
| `--expect-streaming` | `false` | Exit non-zero if streaming is not detected (implies `--streaming`) |
| `--stall-threshold` | `500ms` | Gap between chunks that counts as a stall |

### Scenario

| Flag | Short | Description |
|------|-------|-------------|
| `--scenario` | `-s` | Run a multi-step YAML scenario (variable extraction, cookie session, expectations) |

### Request body template variables

Used in `--data`; each is substituted per request:

| Variable | Value |
|----------|-------|
| `{{seq}}` | Sequential request number (1-indexed) |
| `{{uuid}}` | Random UUID v4 |
| `{{timestamp}}` | Unix seconds |
| `{{timestamp_ms}}` | Unix milliseconds |
| `{{random}}` | Random integer 0-999999 |
| `{{url_index}}` | 0-indexed URL position (useful with `-L`) |

## Combination rules and gotchas

- **`--streaming` / `--expect-streaming` are single-request only.** With `-n > 1`
  or multiple URLs they return an error (they used to be silently ignored, which
  made `--expect-streaming` a false pass in CI). For streaming validation, use a
  single request: `gocurl --expect-streaming <url>`.
- **`--warmup` must be `< -n`** when `-n > 1` (otherwise every request would be a
  warmup and metrics would be empty). Warmup requests run but are excluded from
  stats and from `--export-csv`.
- **`--ramp-up` needs `-c > 1`** to have any effect. Worker `i` activates at
  `t = (ramp-up / (c-1)) * i`, so effective concurrency climbs linearly to `-c`.
  If the request queue drains before the ramp completes (few requests, fast
  responses), the run finishes early - increase `-n` to observe the full ramp.
- **`--rps` is a global limit** shared across all workers, not per-worker.
- **`--query-params` multiplies URLs.** N base URLs times M param lines = N*M
  effective URLs, and `-n` requests are issued per effective URL.
- **`-I` (HEAD)** implies headers-only; there is no body to measure transfer on.
- **`--connect-to` vs `--resolve`:** `--resolve` only swaps the IP for a host:port
  (DNS override); `--connect-to` swaps the entire destination host:port while
  preserving the original Host header and TLS SNI (useful for hitting a specific
  backend behind a load balancer).
- **Load-mode JSON/prom has no per-phase breakdown** - use a single request for
  the DNS/TCP/TLS/server split.
- **stdout is a clean data channel.** All progress/notices go to stderr, so
  `gocurl -o json ... | jq` and `-o prom ... > metrics.prom` are safe.

## Output formats

| Format | Flag | Best for | Notes |
|--------|------|----------|-------|
| table | `-o table` (default) | Interactive use | Waterfall timeline + phase breakdown (single); latency table + status distribution (load). Colorized. |
| json | `-o json` | Scripting / CI | Machine-readable. Timing values are **fractional milliseconds**. Single = timing breakdown; load = `Stats`. |
| graph | `-o graph` | Eyeballing distributions | ASCII latency histogram + status bars (load). |
| prom | `-o prom` | Monitoring / automation | Prometheus text exposition. Durations in **seconds**. See [Reports](#reports). |

## Units: milliseconds vs seconds

- **table / json / csv: milliseconds.** JSON encodes fractional ms (e.g. `0.42`);
  whole values encode without a decimal point (`150`).
- **prom: seconds.** Prometheus convention is base units, so
  `gocurl_request_duration_seconds` etc. are in seconds (e.g. `0.030`).

Do not mix the two in a dashboard: pick json (ms) or prom (s) and stay consistent.

## Reports

### CSV (per-request rows)

```bash
gocurl -n 500 -c 20 --export-csv results.csv https://api.example.com
```

Columns: `timestamp, url, status_code, dns_lookup_ms, tcp_connection_ms,
tls_handshake_ms, server_processing_ms, content_transfer_ms, total_ms,
response_size_bytes, connection_reused, error`. The `timestamp` is per request
(RFC3339 with nanoseconds). Warmup requests are excluded.

Quick analysis:

```bash
# mean total latency
mlr --csv stats1 -a mean -f total_ms results.csv
# slowest 10 requests
mlr --csv sort -nr total_ms then head -n 10 results.csv
# error rows only
mlr --csv filter '$error != ""' results.csv
```

### Prometheus textfile

```bash
# Write a snapshot for the node_exporter textfile collector
gocurl -o prom -n 200 -c 10 -q https://api.example.com \
  > /var/lib/node_exporter/textfile_collector/gocurl_api.prom
```

Validate the exposition:

```bash
gocurl -o prom -n 20 -c 5 -q https://api.example.com | promtool check metrics
```

### JSON snapshot

```bash
gocurl -o json -n 200 -c 10 -q https://api.example.com > report.json
jq '{p50, p95, p99, rps: .requests_per_second, err: .error_rate}' report.json
```

## Automation recipes

### CI latency gate (fail build if p95 too high)

```bash
#!/usr/bin/env bash
set -euo pipefail
RESULT=$(gocurl -o json -n 100 -c 10 -q "$TARGET_URL")
# p95 is fractional milliseconds; compare numerically with jq
if echo "$RESULT" | jq -e '.p95 > 500' >/dev/null; then
  echo "FAIL: p95 $(echo "$RESULT" | jq .p95)ms exceeds 500ms"
  exit 1
fi
echo "PASS: p95 $(echo "$RESULT" | jq .p95)ms"
```

### Streaming endpoint validation in CI

```bash
# Single request; exits non-zero if buffered
gocurl --expect-streaming --stall-threshold 2s "$SSE_URL"
```

### Cron-scraped Prometheus textfile

```bash
# /etc/cron.d/gocurl-probe  (every minute)
* * * * *  node_exp  gocurl -o prom -n 50 -c 5 -q https://api.example.com > /var/lib/node_exporter/textfile_collector/gocurl_api.prom.$$ && mv /var/lib/node_exporter/textfile_collector/gocurl_api.prom.$$ /var/lib/node_exporter/textfile_collector/gocurl_api.prom
```

Write to a temp file and `mv` (atomic rename) so the collector never reads a
half-written file.

### Push to a Pushgateway

```bash
gocurl -o prom -n 100 -c 10 -q https://api.example.com \
  | curl --data-binary @- http://pushgateway:9091/metrics/job/gocurl/target/api
```

### Compare two environments

```bash
for env in prod staging; do
  echo "=== $env ==="
  gocurl -o json -n 200 -c 20 -q "https://api.$env.example.com/health" \
    | jq '{p50, p95, p99, rps: .requests_per_second}'
done
```

### Continuous monitoring loop

```bash
while true; do
  p95=$(gocurl -o json -n 50 -c 5 -q https://api.example.com | jq .p95)
  printf '%s p95=%sms\n' "$(date -Is)" "$p95" >> latency.log
  sleep 60
done
```

(For anything long-running, prefer a real scheduler + the Prometheus textfile
recipe over a shell loop.)

## Exit codes

| Situation | Exit |
|-----------|------|
| Success | `0` |
| Single request: transport error (DNS/TCP/TLS failure, timeout) | `1` |
| Single request: HTTP 4xx/5xx | `0` (matches curl; the status is still reported) |
| `--expect-streaming` and streaming not detected | `1` |
| `--streaming`/`--expect-streaming` used under load | `1` (rejected) |
| Load test interrupted by SIGINT/SIGTERM | `1` (partial stats still reported) |
| Invalid flags / bad duration / unreadable file | `1` |

Note: a load test does **not** exit non-zero solely because some requests
returned 4xx/5xx - inspect `error_rate` / `failed_requests` in the output (or
`gocurl_error_rate` in prom) and gate on that in CI.
