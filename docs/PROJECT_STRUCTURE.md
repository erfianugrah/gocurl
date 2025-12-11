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
│   │   ├── templates.go    # Request body template variables
│   │   ├── urls.go         # URL list reader
│   │   ├── app_test.go     # Tests (24 test cases)
│   │   └── urls_test.go    # Tests
│   │
│   ├── client/             # HTTP client
│   │   ├── duration.go     # Duration type with JSON marshaling
│   │   ├── http.go         # HTTP client wrapper
│   │   ├── tracer.go       # httptrace integration
│   │   ├── streaming.go    # Streaming analysis & buffering detection
│   │   ├── http_test.go    # Tests
│   │   ├── tracer_test.go  # Tests
│   │   ├── streaming_test.go # Tests
│   │   ├── phase2_test.go  # Tests (Phase 2 features)
│   │   └── phase3_test.go  # Tests (Phase 3 features)
│   │
│   ├── metrics/            # Metrics collection
│   │   ├── collector.go    # Metrics aggregation
│   │   ├── types.go        # Stats structures & types
│   │   ├── collector_test.go # Tests (14)
│   │   └── types_test.go   # Tests (7)
│   │
│   ├── output/             # Output formatters
│   │   ├── formatter.go    # Formatter interface
│   │   ├── table.go        # Table output with go-pretty
│   │   ├── json.go         # JSON output
│   │   ├── graph.go        # Graph/histogram output
│   │   ├── streaming.go    # Streaming metrics output
│   │   ├── json_test.go    # Tests
│   │   └── output_test.go  # Tests
│   │
│   └── scenario/           # Multi-step scenario testing
│       ├── scenario.go     # Scenario execution engine
│       └── scenario_test.go # Tests
│
├── bin/                    # Compiled binaries
│   └── gocurl             # Main binary
│
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── coverage.out           # Test coverage report
│
├── docs/                   # Documentation
│   ├── README.md          # Documentation index
│   ├── FEATURES.md        # Feature documentation
│   ├── PROJECT_STRUCTURE.md # This file
│   └── STREAMING_METRICS_GUIDE.md # Streaming metrics guide
│
├── README.md              # Main user documentation
├── go.mod                 # Go module definition
└── go.sum                 # Dependency checksums
```

## Package Overview

### cmd/gocurl
Entry point for the CLI application. Contains Cobra command setup and flag definitions.

**Files**: 2 main + 1 test
**Tests**: 42 test cases (8 test suites)
**Coverage**: 72.2%
**Purpose**: CLI interface, flag handling, integration tests

### internal/app
Application orchestration and business logic. Handles URL reading, application flow, and signal handling.

**Files**: 4 main + 2 test
**Tests**: 84 test cases
**Coverage**: 49.1%
**Purpose**: Application logic, URL management, template variables

### internal/client
HTTP client implementation with httptrace integration for detailed timing measurements and streaming analysis.

**Files**: 4 main + 5 test
**Tests**: Comprehensive (http, tracer, streaming, Phase 2 & 3 features)
**Coverage**: 63.8%
**Purpose**: HTTP operations, timing measurement, streaming analysis, buffering detection

### internal/metrics
Metrics collection, statistical analysis, and histogram generation.

**Files**: 2 main + 2 test
**Tests**: 21
**Coverage**: 91.5%
**Purpose**: Statistics, percentiles, histograms

### internal/output
Output formatters for different formats (table, JSON, graph, streaming).

**Files**: 5 main + 2 test
**Tests**: JSON, table, graph output tests
**Coverage**: 50.1%
**Purpose**: Output formatting, streaming metrics display

### internal/scenario
Multi-step HTTP workflow testing with variable extraction and session management.

**Files**: 1 main + 1 test
**Tests**: Comprehensive scenario execution tests
**Coverage**: 87.8%
**Purpose**: Complex API workflow testing, session management

## File Counts

| Category | Count |
|----------|-------|
| Go source files | 20 |
| Test files | 13 |
| Documentation | 5 (4 in docs/ + README.md) |
| Total Go files | 33 |
| Total lines of code | ~6,000 |

## Dependencies

### Production Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/jedib0t/go-pretty/v6` - Table formatting
- `github.com/fatih/color` - Terminal colors
- `github.com/guptarohit/asciigraph` - ASCII graphs
- `gopkg.in/yaml.v3` - YAML parsing for scenario files

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
