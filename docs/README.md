# Documentation

Internal documentation for gocurl architecture and design.

## For Users

See the main documentation:
- [**Main README**](../README.md) - Complete user guide with all features

## For Developers

### Architecture & Design

- [**PROJECT_STRUCTURE.md**](PROJECT_STRUCTURE.md) - Code organization and package layout
  - Directory structure
  - Package responsibilities
  - Dependencies
  - Design patterns

- [**FEATURES.md**](FEATURES.md) - Technical feature documentation
  - Implementation details
  - Feature comparisons
  - Performance considerations

- [**STREAMING_METRICS_GUIDE.md**](STREAMING_METRICS_GUIDE.md) - How to interpret streaming metrics
  - Understanding CV, mean delay, and patterns
  - Buffered vs progressive delivery
  - Real-world examples and interpretation
  - A/B testing guidance

## Code Documentation

For detailed code documentation:
```bash
# Generate and view godoc
godoc -http=:6060
# Then visit http://localhost:6060/pkg/github.com/erfi/gocurl/
```

## Quick Reference

### Project Structure
```
gocurl/
├── cmd/gocurl/              # CLI entry point
│   ├── main.go             # Main entry
│   └── root.go             # Cobra command setup
├── internal/
│   ├── app/                # Application logic
│   ├── client/             # HTTP client with tracing
│   ├── metrics/            # Statistics & analysis
│   └── output/             # Output formatters
└── docs/                   # This directory
```

### Key Components

- **client.Client**: HTTP client with performance measurement
- **client.Tracer**: httptrace integration for detailed timing
- **client.StreamingReader**: Progressive data delivery analysis
- **metrics.Collector**: Statistical aggregation (p50, p90, p95, p99)
- **output.Formatter**: Multi-format output (table, JSON, graph)

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -cover

# Verbose
go test ./... -v

# Specific package
go test ./internal/client -v
```

---

**Last Updated**: 2025-10-29
**Project Version**: 2.0.0
