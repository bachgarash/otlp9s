package model

import "time"

// LogRecord captures one entry from an OTLP log export.
type LogRecord struct {
	Timestamp   time.Time
	Severity    string
	Body        string
	Attributes  map[string]string
	ServiceName string
	TraceID     string
	SpanID      string
}
