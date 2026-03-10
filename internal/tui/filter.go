package tui

import (
	"strings"

	"github.com/user/otlp9s/internal/model"
)

// filter evaluates a simple filter expression against an event.
// Supported forms:
//
//	service.name = <value>
//	span.name contains <value>
//	trace_id = <value>
//	severity = <value>
//	metric.name contains <value>
//	<any substring> — plain text matching against key fields
func filter(ev model.Event, expr string) bool {
	if expr == "" {
		return true
	}

	expr = strings.TrimSpace(expr)

	// Structured filter: "key op value"
	if parts := parseFilter(expr); parts != nil {
		return evalFilter(ev, parts)
	}

	// Fallback: plain substring match against summary fields.
	lower := strings.ToLower(expr)
	switch ev.Type {
	case model.SignalTrace:
		if ev.Span == nil {
			return false
		}
		return containsLower(ev.Span.Name, lower) ||
			containsLower(ev.Span.ServiceName, lower) ||
			containsLower(ev.Span.TraceID, lower)
	case model.SignalMetric:
		if ev.Metric == nil {
			return false
		}
		return containsLower(ev.Metric.Name, lower) ||
			containsLower(ev.Metric.ServiceName, lower)
	case model.SignalLog:
		if ev.Log == nil {
			return false
		}
		return containsLower(ev.Log.Body, lower) ||
			containsLower(ev.Log.Severity, lower) ||
			containsLower(ev.Log.ServiceName, lower)
	}
	return true
}

type filterParts struct {
	key string
	op  string // "=" or "contains"
	val string
}

func parseFilter(expr string) *filterParts {
	// Try "key contains value"
	if idx := strings.Index(expr, " contains "); idx > 0 {
		return &filterParts{
			key: strings.TrimSpace(expr[:idx]),
			op:  "contains",
			val: strings.TrimSpace(expr[idx+10:]),
		}
	}
	// Try "key = value"
	if parts := strings.SplitN(expr, "=", 2); len(parts) == 2 {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k != "" && v != "" {
			return &filterParts{key: k, op: "=", val: v}
		}
	}
	return nil
}

func evalFilter(ev model.Event, f *filterParts) bool {
	val := fieldValue(ev, f.key)
	switch f.op {
	case "=":
		return strings.EqualFold(val, f.val)
	case "contains":
		return containsLower(val, strings.ToLower(f.val))
	}
	return false
}

func fieldValue(ev model.Event, key string) string {
	switch key {
	case "service.name":
		switch ev.Type {
		case model.SignalTrace:
			if ev.Span != nil {
				return ev.Span.ServiceName
			}
		case model.SignalMetric:
			if ev.Metric != nil {
				return ev.Metric.ServiceName
			}
		case model.SignalLog:
			if ev.Log != nil {
				return ev.Log.ServiceName
			}
		}
	case "span.name":
		if ev.Span != nil {
			return ev.Span.Name
		}
	case "trace_id":
		if ev.Span != nil {
			return ev.Span.TraceID
		}
		if ev.Log != nil {
			return ev.Log.TraceID
		}
	case "severity":
		if ev.Log != nil {
			return ev.Log.Severity
		}
	case "metric.name":
		if ev.Metric != nil {
			return ev.Metric.Name
		}
	}
	return ""
}

func containsLower(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), sub)
}
