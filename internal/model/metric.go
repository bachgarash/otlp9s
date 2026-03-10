package model

import "time"

// MetricType mirrors the OTLP metric data types at a summary level.
type MetricType string

const (
	MetricGauge     MetricType = "gauge"
	MetricSum       MetricType = "sum"
	MetricHistogram MetricType = "histogram"
	MetricSummary   MetricType = "summary"
)

// Metric captures one data point from an OTLP metric export.
// For histogram/summary we store a string representation of the value;
// for gauge/sum we store the numeric value directly.
type Metric struct {
	Name               string
	Type               MetricType
	Value              string
	Timestamp          time.Time
	Attributes         map[string]string
	ResourceAttributes map[string]string
	ServiceName        string
}
