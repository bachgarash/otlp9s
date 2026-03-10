# CLAUDE.md

## Project Overview

otlp9s is a terminal-based OTLP proxy/inspector. It sits between an OpenTelemetry collector and a backend, intercepting and displaying traces, metrics, and logs in a real-time TUI.

## Build & Run

```bash
make build                              # builds ./otlp9s binary
make run                                # builds + runs with --forward localhost:4320
go test ./...                           # run all tests
```

## Project Structure

```
cmd/otlp9s/main.go          # CLI entry point (cobra)
internal/
  proxy/                     # gRPC + HTTP reverse proxies
    grpc_proxy.go            # OTLP/gRPC collector service implementation
    http_proxy.go            # OTLP/HTTP endpoint handlers
  decoder/                   # OTLP protobuf → internal model
    traces.go, metrics.go, logs.go, common.go
  model/                     # Data types: Event, Span, Metric, LogRecord
    event.go, span.go, metric.go, log.go
  pipeline/                  # Event router with rate counters
    router.go
  store/                     # Storage
    ringbuffer.go            # Bounded circular buffer (thread-safe)
    trace_index.go           # Span grouping by trace ID
  tui/                       # Terminal UI (bubbletea + lipgloss)
    app.go                   # Main model, Update loop, key handling
    views.go                 # All rendering functions
    trace_tree.go            # Trace tree builder (DFS flattening)
    filter.go                # Filter expression parser + evaluator
    events.go                # Bubbletea messages (tick, newEvents)
examples/                    # Collector configs and demo integration script
```

## Key Architecture Decisions

- **Unified Event envelope**: All three signal types share `model.Event` with exactly one non-nil field (Span/Metric/Log).
- **Ring buffer**: Bounded memory with silent eviction of oldest events. Capacity set via `--buffer-size`.
- **Streaming toggle**: `s` key toggles between live streaming and frozen/browse mode. When frozen, data stops refreshing entirely (no background updates).
- **Pinned selection**: `Enter` deep-copies the selected item into `sel` so the detail panel is stable regardless of streaming state. Map iteration uses sorted keys to prevent flickering.
- **Rate counters**: Atomic counters sampled every second in `rateLoop()`.

## Code Conventions

- Go standard project layout with `internal/` packages
- No external test frameworks -- use standard `testing` package
- Module path: `github.com/user/otlp9s`
- All TUI state mutations happen through bubbletea's `Update` → return new model pattern
- Thread safety: `sync.RWMutex` for shared state (RingBuffer, TraceIndex, Router rates)
- Map rendering: always use `sortedKeys()` for deterministic output

## Common Tasks

- **Add a new filter field**: Update `fieldValue()` in `internal/tui/filter.go`
- **Add a new signal type display**: Add case to `refreshData()`, create render functions in `views.go`
- **Change TUI layout/colors**: Edit styles in `views.go` (top section)
- **Modify key bindings**: Edit `handleNormalKey()` in `app.go`
