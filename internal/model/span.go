package model

import "time"

// SignalType distinguishes the three OTLP signal categories.
type SignalType int

const (
	SignalTrace SignalType = iota
	SignalMetric
	SignalLog
)

// Span is a simplified representation of an OTLP span, retaining only
// the fields needed for debugging inspection. We intentionally keep this
// flat rather than mirroring the full proto hierarchy — the goal is fast
// rendering, not lossless roundtrip.
type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	ServiceName  string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Attributes   map[string]string
}
