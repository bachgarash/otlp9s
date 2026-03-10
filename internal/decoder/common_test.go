package decoder

import (
	"testing"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

func TestAttrsToMap(t *testing.T) {
	kvs := []*commonpb.KeyValue{
		{Key: "http.method", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "GET"}}},
		{Key: "http.status_code", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 200}}},
		{Key: "success", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: true}}},
		{Key: "latency", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 1.5}}},
	}

	m := attrsToMap(kvs)

	tests := map[string]string{
		"http.method":      "GET",
		"http.status_code": "200",
		"success":          "true",
		"latency":          "1.5",
	}

	for k, want := range tests {
		if got := m[k]; got != want {
			t.Errorf("attrsToMap[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestAttrsToMap_Empty(t *testing.T) {
	m := attrsToMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}

func TestAnyValueToString(t *testing.T) {
	tests := []struct {
		name  string
		value *commonpb.AnyValue
		want  string
	}{
		{"nil", nil, ""},
		{"string", &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "hello"}}, "hello"},
		{"int", &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 42}}, "42"},
		{"double", &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 3.14}}, "3.14"},
		{"bool_true", &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: true}}, "true"},
		{"bool_false", &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: false}}, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyValueToString(tt.value)
			if got != tt.want {
				t.Errorf("anyValueToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceName(t *testing.T) {
	attrs := []*commonpb.KeyValue{
		{Key: "host.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "server-1"}}},
		{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "my-service"}}},
	}

	got := serviceName(attrs)
	if got != "my-service" {
		t.Errorf("serviceName() = %q, want %q", got, "my-service")
	}
}

func TestServiceName_Missing(t *testing.T) {
	attrs := []*commonpb.KeyValue{
		{Key: "host.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "server-1"}}},
	}

	got := serviceName(attrs)
	if got != "unknown" {
		t.Errorf("serviceName() = %q, want %q", got, "unknown")
	}
}

func TestTraceIDToHex(t *testing.T) {
	id := []byte{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89}
	got := traceIDToHex(id)
	want := "abcdef0123456789abcdef0123456789"
	if got != want {
		t.Errorf("traceIDToHex() = %q, want %q", got, want)
	}
}

func TestSpanIDToHex(t *testing.T) {
	id := []byte{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89}
	got := spanIDToHex(id)
	want := "abcdef0123456789"
	if got != want {
		t.Errorf("spanIDToHex() = %q, want %q", got, want)
	}
}

func TestTraceIDToHex_Empty(t *testing.T) {
	got := traceIDToHex(nil)
	if got != "" {
		t.Errorf("traceIDToHex(nil) = %q, want empty", got)
	}
}
