#!/usr/bin/env bash
set -euo pipefail

# -------------------------------------------------------------------
# Run otlp9s with the opentelemetry-demo.
#
# Architecture:
#   demo apps → otel-collector → otlp9s (host:4317) → jaeger (host:16317)
#
# The collector's extras config is patched to redirect traces to the host,
# where otlp9s intercepts and forwards to Jaeger.
# -------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_DIR="${DEMO_DIR:-$HOME/dev/opentelemetry-demo}"

OTLP9S_PORT=4317
JAEGER_HOST_PORT=16317
EXTRAS_FILE="$DEMO_DIR/src/otel-collector/otelcol-config-extras.yml"

# --- Build ---
echo "==> Building otlp9s..."
cd "$PROJECT_DIR"
go build -o otlp9s ./cmd/otlp9s/

# --- Patch collector config ---
echo "==> Patching collector extras to redirect traces to otlp9s on host..."
cp "$EXTRAS_FILE" "$EXTRAS_FILE.bak"

cat > "$EXTRAS_FILE" <<'YAML'
# Patched by otlp9s — redirect traces to host where otlp9s is running.
# Original backed up as otelcol-config-extras.yml.bak

exporters:
  otlp_grpc/jaeger:
    endpoint: "host.docker.internal:4317"
    tls:
      insecure: true
YAML

# Restore the original config on exit.
cleanup() {
  echo ""
  echo "==> Restoring original collector config..."
  mv "$EXTRAS_FILE.bak" "$EXTRAS_FILE"
  echo "    Done. Run 'docker compose restart otel-collector' in the demo dir to revert."
}
trap cleanup EXIT

# --- Start demo ---
echo "==> Starting opentelemetry-demo..."
cd "$DEMO_DIR"

# Map Jaeger gRPC to a non-conflicting host port.
JAEGER_GRPC_PORT="$JAEGER_HOST_PORT:4317" docker compose up -d

echo "==> Waiting for containers..."
sleep 5

# Restart the collector to pick up the patched config.
echo "==> Restarting otel-collector with patched config..."
docker compose restart otel-collector
sleep 2

echo ""
echo "==> Starting otlp9s..."
echo "    Listening on :$OTLP9S_PORT (collector sends traces here)"
echo "    Forwarding to localhost:$JAEGER_HOST_PORT (Jaeger)"
echo "    Jaeger UI: http://localhost:16686"
echo ""

cd "$PROJECT_DIR"
exec ./otlp9s \
  --listen ":$OTLP9S_PORT" \
  --forward "localhost:$JAEGER_HOST_PORT"
