# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-03-10

### Added

- **Core engine**: unified Event model with Span, Metric, and LogRecord types
- **Storage**: thread-safe RingBuffer with bounded capacity and automatic eviction
- **Trace index**: span grouping by trace ID for tree reconstruction
- **Pipeline**: event Router with per-second rate counters and TUI notification
- **gRPC proxy**: OTLP/gRPC collector service that taps and forwards traces, metrics, and logs
- **HTTP proxy**: OTLP/HTTP endpoint handlers for `/v1/traces`, `/v1/metrics`, `/v1/logs`
- **Decoder**: OTLP protobuf to internal model conversion for all three signal types
- **Terminal UI**: interactive TUI built with bubbletea and lipgloss
  - Three-tab layout: Traces, Metrics, Logs
  - Trace tree visualization with parent-child span hierarchy
  - Full-width cursor highlight bar for clear navigation
  - Streaming toggle (`s` key) -- freeze/resume like k9s
  - Pin & inspect with `Enter` -- detail panel stays stable
  - Sorted attribute rendering to prevent flickering
  - Filtering with structured expressions and plain substring search
- **CLI**: cobra-based entry point with `--forward`, `--listen`, `--grpc`, `--http`, `--buffer-size` flags
- **CI**: GitHub Actions workflow for build, test (with race detector), and go vet
- **Release**: cross-platform release workflow (linux/darwin, amd64/arm64) with auto-generated changelog
- **Examples**: collector config, demo override, and opentelemetry-demo integration script
- **License**: The Unlicense (public domain)

[Unreleased]: https://github.com/bachgarash/otlp9s/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/bachgarash/otlp9s/releases/tag/v0.1.0
