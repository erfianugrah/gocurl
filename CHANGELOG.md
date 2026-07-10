# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.5.0]

Prometheus output format and a consolidated usage/automation reference.

### Added

- **Prometheus output format (`-o prom`).** Previously advertised in `--help` but
  unimplemented (it silently rendered a table). Now a real text-exposition
  formatter: aggregate load metrics as counters + a latency `summary`
  (`gocurl_request_duration_seconds` with quantiles, `_sum`, `_count`), plus
  `gocurl_requests_total`, `gocurl_error_rate`, `gocurl_requests_per_second`,
  `gocurl_responses_total{status}`, etc. A single request emits per-phase gauges
  (`gocurl_request_phase_seconds{phase=...}`). Durations are in **seconds**
  (Prometheus base-unit convention). Suitable for the node_exporter textfile
  collector or a Pushgateway.
- **`docs/USAGE.md`** - authoritative reference: full flag matrix, mode-selection
  rules, flag-combination gotchas, output-format/units guide, report generation
  (CSV / Prometheus textfile / JSON), and CI/automation recipes.
- Documented the `--stdin` flag (alias for `-L -`).

## [v1.4.0]

Correctness, accuracy, and honesty pass. Several long-standing behaviors that did
not match the documentation (or silently produced wrong results) were fixed.

### Behavior changes (may affect existing consumers)

- **JSON timing values are now fractional milliseconds.** `Duration` fields
  previously marshaled to integer milliseconds, truncating every sub-millisecond
  phase (DNS/TCP/TLS on fast or local endpoints) to `0`. They now encode as
  fractional numbers (e.g. `0.42`); whole values still encode without a decimal
  point (e.g. `150`), so integer consumers keep working. Scripts using shell
  integer comparison (`[ "$x" -gt N ]`) on these values should switch to a
  numeric comparison (e.g. `jq -e '.p95 > N'`).
- **4xx/5xx responses now count as failures.** `successful_requests` counts only
  requests that completed without a transport error *and* returned status < 400.
  `failed_requests` and `error_rate` now include HTTP error statuses. The
  `status_codes` distribution is unchanged. Single-request exit code is unchanged
  (an HTTP error status still exits 0, matching curl's default).
- **`--streaming` / `--expect-streaming` now error under load** instead of being
  silently ignored. They are single-request-only (one URL, `-n 1`). Previously
  `--expect-streaming -n 5` performed no streaming analysis and exited 0 - a
  false pass in CI. Use a single request for streaming validation.

### Fixed

- **Ramp-up now actually ramps.** Workers previously all activated only *after*
  the full ramp-up sleep, then fired simultaneously, so `--ramp-up` was just a
  fixed start delay with no gradual load. Workers now activate on a stagger
  schedule (worker `i` at `t = interval*i`) while consuming a pre-filled queue,
  so effective concurrency climbs linearly over the ramp-up duration.
- **Ctrl-C / SIGINT / SIGTERM now cancel in-flight requests** and report
  statistics for completed requests before exiting non-zero. Previously there was
  no wired signal handling and in-flight requests used a non-cancellable context.
- **Requests/sec and Duration are measured over the actual request window.** The
  throughput denominator previously started at collector creation, so ramp-up
  dead time and warmup wall-time deflated the reported rate. It now spans only
  recorded (non-warmup) requests via per-request timestamps.
- **Redirect count is now the true hop count.** Previously hardcoded to `1` for
  any redirect chain; now counted via the request context in `CheckRedirect`.
- **CSV `timestamp` column is now per-request** wall-clock time, not the identical
  export moment on every row.
- **Non-data notices moved to stderr.** The "Expanded N base URL(s)...",
  "Exported N requests...", and streaming-validation-passed messages went to
  stdout and corrupted `-o json` output. They now go to stderr, keeping stdout a
  clean data channel for `jq`.
- **Failed requests are marked with a cross, not a check.** A transport error or
  4xx/5xx status previously still printed a `✓` in table output; it now prints
  `✗` in red.
- **Timing breakdown percentages reconcile to 100%.** An `Other/Setup` row now
  accounts for time not attributed to a named phase (connection acquisition,
  inter-phase gaps).

### Removed

- Deleted 15 unreachable functions that were never wired up: the 12 `assess*`
  phase-rating helpers (the advertised "performance assessments"/"Excellent"
  column never existed in real output), `drawLatencyGraph`,
  `createHistogramBuckets`, and `SetupSignalHandler` (replaced by working
  `signal.NotifyContext`-based cancellation).

### Documentation

- README: corrected the single-request table example (there is no `ASSESSMENT`
  column), removed the "performance assessments" comparison-matrix claim, updated
  the JSON sample to fractional milliseconds, made the CI/CD `jq` threshold check
  float-safe, corrected the minimum Go version (1.24), and softened the
  timing-accuracy FAQ.
- FEATURES.md: corrected the signal-handling and color-coding descriptions to
  match actual behavior.

## [v1.3.1]

- Fix load test to process all URLs when using `-n 1` with multiple URLs.
- Add real-time progress indicator for load tests.
- Fix CI workflow to use bash shell on all platforms including Windows.
