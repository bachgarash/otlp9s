package model

import "time"

// Event is the unified envelope sent through the internal pipeline.
// Exactly one of Span/Metric/Log is non-nil.
type Event struct {
	Type      SignalType
	Timestamp time.Time
	Span      *Span
	Metric    *Metric
	Log       *LogRecord
}
