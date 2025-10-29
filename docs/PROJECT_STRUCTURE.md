# Project Structure

## Directory Layout

```
gocurl/
├── cmd/gocurl/              # CLI entry point
│   ├── main.go             # Main executable
│   └── root.go             # Cobra commands & flags
│
├── internal/                # Internal packages
│   ├── app/                # Application logic
│   │   ├── app.go          # Main application orchestration
│   │   ├── signals.go      # Signal handling (SIGINT/SIGTERM)
│   │   ├── urls.go         # URL list reader
│   │   └── urls_test.go    # Tests (11)
│   │
│   ├── client/             # HTTP client
│   │   ├── http.go         # HTTP client wrapper
│   │   ├── tracer.go       # httptrace integration
│   │   ├── http_test.go    # Tests (10)
│   │   └── tracer_test.go  # Tests (4)
│   │
│   ├── metrics/            # Metrics collection
│   │   ├── collector.go    # Metrics aggregation
│   │   ├── types.go        # Stats structures & types
│   │   ├── collector_test.go # Tests (14)
│   │   └── types_test.go   # Tests (7)
│   │
│   └── output/             # Output formatters
│       ├── formatter.go    # Formatter interface
│       ├── table.go        # Table output with go-pretty
│       ├── json.go         # JSON output
│       ├── graph.go        # Graph/histogram output
│       └── json_test.go    # Tests (5)
│
├── bin/                    # Compiled binaries
│   └── gocurl             # Main binary
│
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── coverage.out           # Test coverage report
│
└── Documentation/
    ├── CLAUDE.md          # Implementation guide
    ├── README.md          # User documentation
    ├── QUICKSTART.md      # Quick reference
    ├── FEATURES.md        # New features docs
    ├── SUMMARY.md         # Enhancement summary
    ├── TEST_SUMMARY.md    # Test documentation
    ├── COMPLETION_REPORT.md # Project completion
    └── PROJECT_STRUCTURE.md # This file
```

## Package Overview

### cmd/gocurl
Entry point for the CLI application. Contains Cobra command setup and flag definitions.

**Files**: 2
**Purpose**: CLI interface

### internal/app
Application orchestration and business logic. Handles URL reading, application flow, and signal handling.

**Files**: 3 main + 1 test
**Tests**: 11
**Coverage**: 16.0%
**Purpose**: Application logic, URL management

### internal/client
HTTP client implementation with httptrace integration for detailed timing measurements.

**Files**: 2 main + 2 test
**Tests**: 14
**Coverage**: 85.7%
**Purpose**: HTTP operations, timing measurement

### internal/metrics
Metrics collection, statistical analysis, and histogram generation.

**Files**: 2 main + 2 test
**Tests**: 21
**Coverage**: 98.6%
**Purpose**: Statistics, percentiles, histograms

### internal/output
Output formatters for different formats (table, JSON, graph).

**Files**: 4 main + 1 test
**Tests**: 5
**Coverage**: 4.0%
**Purpose**: Output formatting

## File Counts

| Category | Count |
|----------|-------|
| Go source files | 13 |
| Test files | 6 |
| Documentation | 8 |
| Total Go files | 19 |
| Total lines of code | ~3,500 |

## Dependencies

### Production Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/jedib0t/go-pretty/v6` - Table formatting
- `github.com/fatih/color` - Terminal colors
- `github.com/guptarohit/asciigraph` - ASCII graphs

### Standard Library Usage
- `net/http` - HTTP client
- `net/http/httptrace` - Request tracing
- `crypto/tls` - TLS configuration
- `encoding/json` - JSON marshaling
- `time` - Duration and timing
- `sync` - Concurrency primitives

## Key Design Patterns

### Interface-Based Design
All major components use interfaces for testability:
- `HTTPClient` interface
- `Formatter` interface
- `MetricsCollector` (implicit)

### Worker Pool Pattern
Load testing uses goroutines with channels for concurrent request execution.

### Factory Pattern
`GetFormatter()` returns appropriate formatter based on output type.

### Observer Pattern
Metrics collector records all timing measurements for aggregation.

## Build Artifacts

### Binary
- `bin/gocurl` - Main executable (~8-10 MB)

### Test Artifacts
- `coverage.out` - Coverage profile for analysis

## Notes

- All internal packages follow Go's standard project layout
- Tests are colocated with source files
- No external dependencies for core HTTP functionality
- Clean separation of concerns between packages
- Well-defined interfaces for extensibility
