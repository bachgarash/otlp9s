package tui

import (
	"testing"
	"time"

	"github.com/user/otlp9s/internal/model"
)

func spanEvent(name, svc, traceID string) model.Event {
	return model.Event{
		Type:      model.SignalTrace,
		Timestamp: time.Now(),
		Span: &model.Span{
			Name:        name,
			ServiceName: svc,
			TraceID:     traceID,
		},
	}
}

func metricEvent(name, svc string) model.Event {
	return model.Event{
		Type:      model.SignalMetric,
		Timestamp: time.Now(),
		Metric: &model.Metric{
			Name:        name,
			ServiceName: svc,
		},
	}
}

func logEvent(body, sev, svc string) model.Event {
	return model.Event{
		Type:      model.SignalLog,
		Timestamp: time.Now(),
		Log: &model.LogRecord{
			Body:        body,
			Severity:    sev,
			ServiceName: svc,
		},
	}
}

func TestFilter_EmptyExprMatchesAll(t *testing.T) {
	ev := spanEvent("test", "svc", "t1")
	if !filter(ev, "") {
		t.Error("empty filter should match everything")
	}
}

func TestFilter_SubstringMatch(t *testing.T) {
	ev := spanEvent("GET /api/users", "user-service", "abc123")

	if !filter(ev, "api") {
		t.Error("should match span name substring")
	}
	if !filter(ev, "user-service") {
		t.Error("should match service name")
	}
	if !filter(ev, "ABC") {
		t.Error("should be case-insensitive")
	}
	if filter(ev, "nonexistent") {
		t.Error("should not match unrelated text")
	}
}

func TestFilter_ServiceNameEquals(t *testing.T) {
	ev := spanEvent("op", "my-service", "t1")

	if !filter(ev, "service.name = my-service") {
		t.Error("should match exact service name")
	}
	if !filter(ev, "service.name = MY-SERVICE") {
		t.Error("should be case-insensitive")
	}
	if filter(ev, "service.name = other-service") {
		t.Error("should not match different service")
	}
}

func TestFilter_SpanNameContains(t *testing.T) {
	ev := spanEvent("GET /api/users", "svc", "t1")

	if !filter(ev, "span.name contains /api") {
		t.Error("should match span name contains")
	}
	if filter(ev, "span.name contains /orders") {
		t.Error("should not match different path")
	}
}

func TestFilter_TraceID(t *testing.T) {
	ev := spanEvent("op", "svc", "abc123def456")

	if !filter(ev, "trace_id = abc123def456") {
		t.Error("should match exact trace ID")
	}
	if filter(ev, "trace_id = other") {
		t.Error("should not match different trace ID")
	}
}

func TestFilter_Severity(t *testing.T) {
	ev := logEvent("something failed", "ERROR", "svc")

	if !filter(ev, "severity = ERROR") {
		t.Error("should match severity")
	}
	if !filter(ev, "severity = error") {
		t.Error("should be case-insensitive")
	}
	if filter(ev, "severity = WARN") {
		t.Error("should not match different severity")
	}
}

func TestFilter_MetricNameContains(t *testing.T) {
	ev := metricEvent("http.server.request.duration", "svc")

	if !filter(ev, "metric.name contains request") {
		t.Error("should match metric name contains")
	}
	if filter(ev, "metric.name contains response") {
		t.Error("should not match different term")
	}
}

func TestFilter_SubstringOnMetrics(t *testing.T) {
	ev := metricEvent("cpu.usage", "infra-service")

	if !filter(ev, "cpu") {
		t.Error("should match metric name substring")
	}
	if !filter(ev, "infra") {
		t.Error("should match metric service name substring")
	}
}

func TestFilter_SubstringOnLogs(t *testing.T) {
	ev := logEvent("connection refused", "ERROR", "gateway")

	if !filter(ev, "refused") {
		t.Error("should match log body substring")
	}
	if !filter(ev, "error") {
		t.Error("should match severity substring case-insensitive")
	}
	if !filter(ev, "gateway") {
		t.Error("should match service name substring")
	}
}

func TestFilter_NilFields(t *testing.T) {
	// Event with nil span should not panic.
	ev := model.Event{Type: model.SignalTrace, Span: nil}
	if filter(ev, "something") {
		t.Error("nil span should not match")
	}

	ev = model.Event{Type: model.SignalMetric, Metric: nil}
	if filter(ev, "something") {
		t.Error("nil metric should not match")
	}

	ev = model.Event{Type: model.SignalLog, Log: nil}
	if filter(ev, "something") {
		t.Error("nil log should not match")
	}
}

func TestParseFilter_Structured(t *testing.T) {
	tests := []struct {
		input string
		key   string
		op    string
		val   string
	}{
		{"service.name = foo", "service.name", "=", "foo"},
		{"span.name contains /api", "span.name", "contains", "/api"},
		{"severity = ERROR", "severity", "=", "ERROR"},
	}

	for _, tt := range tests {
		parts := parseFilter(tt.input)
		if parts == nil {
			t.Errorf("parseFilter(%q) returned nil", tt.input)
			continue
		}
		if parts.key != tt.key || parts.op != tt.op || parts.val != tt.val {
			t.Errorf("parseFilter(%q) = {%s %s %s}, want {%s %s %s}",
				tt.input, parts.key, parts.op, parts.val, tt.key, tt.op, tt.val)
		}
	}
}

func TestParseFilter_PlainText(t *testing.T) {
	// Plain text without = or contains should return nil (fallback to substring).
	parts := parseFilter("hello world")
	if parts != nil {
		t.Errorf("plain text should return nil, got %+v", parts)
	}
}
