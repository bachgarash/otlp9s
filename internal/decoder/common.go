package decoder

import (
	"encoding/hex"
	"fmt"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

// attrsToMap flattens OTLP KeyValue slices into a simple string map.
// Complex values are rendered with fmt for readability.
func attrsToMap(kvs []*commonpb.KeyValue) map[string]string {
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = anyValueToString(kv.Value)
	}
	return m
}

func anyValueToString(v *commonpb.AnyValue) string {
	if v == nil {
		return ""
	}
	switch val := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return val.StringValue
	case *commonpb.AnyValue_IntValue:
		return fmt.Sprintf("%d", val.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%g", val.DoubleValue)
	case *commonpb.AnyValue_BoolValue:
		return fmt.Sprintf("%t", val.BoolValue)
	default:
		return fmt.Sprintf("%v", v.Value)
	}
}

func traceIDToHex(b []byte) string {
	return hex.EncodeToString(b)
}

func spanIDToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// serviceName extracts service.name from resource attributes.
func serviceName(attrs []*commonpb.KeyValue) string {
	for _, kv := range attrs {
		if kv.Key == "service.name" {
			return anyValueToString(kv.Value)
		}
	}
	return "unknown"
}
