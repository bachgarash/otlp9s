package decoder

import (
	"fmt"
	"time"

	"github.com/user/otlp9s/internal/model"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// DecodeMetrics converts an OTLP metrics export into internal Metric events.
func DecodeMetrics(req *metricspb.MetricsData) []model.Event {
	var events []model.Event

	for _, rm := range req.ResourceMetrics {
		svcName := ""
		resAttrs := map[string]string{}
		if rm.Resource != nil {
			svcName = serviceName(rm.Resource.Attributes)
			resAttrs = attrsToMap(rm.Resource.Attributes)
		}

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				for _, dp := range extractDataPoints(m) {
					events = append(events, model.Event{
						Type:      model.SignalMetric,
						Timestamp: dp.Timestamp,
						Metric: &model.Metric{
							Name:               m.Name,
							Type:               dp.Type,
							Value:              dp.Value,
							Timestamp:          dp.Timestamp,
							Attributes:         dp.Attributes,
							ResourceAttributes: resAttrs,
							ServiceName:        svcName,
						},
					})
				}
			}
		}
	}
	return events
}

type dataPoint struct {
	Type       model.MetricType
	Value      string
	Timestamp  time.Time
	Attributes map[string]string
}

// extractDataPoints handles the different metric data variants.
func extractDataPoints(m *metricspb.Metric) []dataPoint {
	var out []dataPoint

	switch d := m.Data.(type) {
	case *metricspb.Metric_Gauge:
		for _, dp := range d.Gauge.DataPoints {
			out = append(out, numberDP(model.MetricGauge, dp))
		}
	case *metricspb.Metric_Sum:
		for _, dp := range d.Sum.DataPoints {
			out = append(out, numberDP(model.MetricSum, dp))
		}
	case *metricspb.Metric_Histogram:
		for _, dp := range d.Histogram.DataPoints {
			out = append(out, dataPoint{
				Type:       model.MetricHistogram,
				Value:      fmt.Sprintf("count=%d sum=%g", dp.GetCount(), dp.GetSum()),
				Timestamp:  time.Unix(0, int64(dp.TimeUnixNano)),
				Attributes: attrsToMap(dp.Attributes),
			})
		}
	case *metricspb.Metric_Summary:
		for _, dp := range d.Summary.DataPoints {
			out = append(out, dataPoint{
				Type:       model.MetricSummary,
				Value:      fmt.Sprintf("count=%d sum=%g", dp.Count, dp.Sum),
				Timestamp:  time.Unix(0, int64(dp.TimeUnixNano)),
				Attributes: attrsToMap(dp.Attributes),
			})
		}
	}
	return out
}

func numberDP(mt model.MetricType, dp *metricspb.NumberDataPoint) dataPoint {
	var val string
	switch v := dp.Value.(type) {
	case *metricspb.NumberDataPoint_AsInt:
		val = fmt.Sprintf("%d", v.AsInt)
	case *metricspb.NumberDataPoint_AsDouble:
		val = fmt.Sprintf("%g", v.AsDouble)
	}
	return dataPoint{
		Type:       mt,
		Value:      val,
		Timestamp:  time.Unix(0, int64(dp.TimeUnixNano)),
		Attributes: attrsToMap(dp.Attributes),
	}
}
