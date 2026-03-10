# otlp9s

**Interactive OTLP debugger** -- inspect OpenTelemetry traffic in your terminal.

`otlp9s` is a lightweight terminal proxy that sits between your OpenTelemetry collector (or SDK) and your backend. It intercepts all three signal types -- **traces**, **metrics**, and **logs** -- and renders them in a real-time interactive TUI.

```
your app --> collector --> otlp9s --> backend (Jaeger, Tempo, etc.)
```

## Features

- **Live streaming** of traces, metrics, and logs with per-second rate counters
- **Trace tree visualization** with parent-child span hierarchy
- **Dual protocol support** -- OTLP/gRPC and OTLP/HTTP
- **Transparent forwarding** -- traffic passes through to your real backend untouched
- **Freeze & browse** -- pause the stream to inspect data, like k9s log view
- **Pin & inspect** -- select any item to see full attributes in the detail panel
- **Filtering** -- structured filters (`service.name = foo`) or plain substring search
- **Bounded memory** -- ring buffer with configurable capacity; oldest events evicted automatically

## Quick Start

### Build

```bash
make build
```

### Run

```bash
# Intercept gRPC OTLP on :4317, forward to backend on :4320
otlp9s --forward localhost:4320

# Enable both gRPC and HTTP proxies
otlp9s --forward localhost:4320 --grpc --http

# Custom listen addresses
otlp9s --forward localhost:4320 --listen :4317 --http --http-listen :4318
```

### Point your collector at otlp9s

```yaml
# In your collector config, export to otlp9s instead of your backend:
exporters:
  otlp/tap:
    endpoint: localhost:4317
    tls:
      insecure: true
```

See [`examples/collector-config.yaml`](examples/collector-config.yaml) for a full example.

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--forward` | *(required)* | OTLP backend address to forward traffic to |
| `--listen` | `:4317` | gRPC listen address |
| `--grpc` | `true`* | Enable OTLP/gRPC proxy |
| `--http` | `false` | Enable OTLP/HTTP proxy |
| `--http-listen` | `:4318` | HTTP listen address (when `--http` is set) |
| `--buffer-size` | `10000` | Ring buffer capacity (max events in memory) |

\* gRPC is enabled by default if neither `--grpc` nor `--http` is specified.

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` / `Home` | Jump to top |
| `PgDn` / `PgUp` | Page down / up |
| `1` `2` `3` | Switch to Traces / Metrics / Logs tab |
| `Tab` / `Shift+Tab` | Cycle tabs |

### Streaming Control

| Key | Action |
|-----|--------|
| `s` | Toggle pause/resume stream |
| `f` / `G` | Resume streaming |

### Inspection

| Key | Action |
|-----|--------|
| `Enter` | Pin selected item in the detail panel |
| `Esc` | Clear pinned selection |

### Filtering

| Key | Action |
|-----|--------|
| `/` | Open filter prompt |
| `Enter` | Apply filter |
| `Esc` | Clear filter |
| `Ctrl+U` | Clear filter text |

### Filter Expressions

```
service.name = my-service        # exact match on service name
span.name contains /api          # substring match on span name
trace_id = abc123...             # filter by trace ID
severity = ERROR                 # filter logs by severity
metric.name contains cpu         # filter metrics by name
http                             # plain substring across all fields
```

### Other

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |

## Architecture

```
                    +------------------+
                    |    otlp9s TUI    |
                    |  (bubbletea)     |
                    +--------+---------+
                             |
                    +--------+---------+
                    |     Router       |
                    |  rate counters   |
                    +--+----+------+---+
                       |    |      |
              +--------+  +-+--+   +--------+
              |           |    |            |
         RingBuffer   TraceIndex     Notify channel
              |                         |
     +--------+--------+          +-----+-----+
     |  gRPC Proxy      |         |  HTTP Proxy |
     |  (:4317)         |         |  (:4318)    |
     +--------+---------+         +------+------+
              |                          |
              +-------- forward ---------+
              |                          |
         OTLP backend (Jaeger, Tempo, etc.)
```

### Internal Packages

| Package | Purpose |
|---------|---------|
| `cmd/otlp9s` | CLI entry point (cobra) |
| `internal/proxy` | gRPC and HTTP reverse proxies |
| `internal/decoder` | OTLP protobuf to internal model conversion |
| `internal/model` | Unified event types (Span, Metric, LogRecord) |
| `internal/pipeline` | Event router with rate counters |
| `internal/store` | Ring buffer + trace index |
| `internal/tui` | Terminal UI (bubbletea + lipgloss) |

## Usage with OpenTelemetry Demo

A helper script is included to run otlp9s with the [opentelemetry-demo](https://github.com/open-telemetry/opentelemetry-demo):

```bash
# Clone the demo first, then:
DEMO_DIR=~/dev/opentelemetry-demo ./examples/run-with-demo.sh
```

This will:
1. Build otlp9s
2. Patch the demo's collector config to route traces through otlp9s
3. Start the demo with docker compose
4. Launch otlp9s as a man-in-the-middle between the collector and Jaeger
5. Restore the original config on exit

## Development

```bash
# Build
make build

# Run with auto-rebuild
make run

# Clean
make clean

# Run tests
go test ./...
```

## Requirements

- Go 1.25+
- An OTLP-compatible backend to forward to

## License

[The Unlicense](LICENSE) — public domain.
