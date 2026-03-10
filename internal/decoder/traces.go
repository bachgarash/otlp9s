package decoder

import (
	"time"

	"github.com/user/otlp9s/internal/model"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// DecodeTraces converts an OTLP trace export into internal Span events.
func DecodeTraces(req *tracepb.TracesData) []model.Event {
	var events []model.Event

	for _, rs := range req.ResourceSpans {
		svcName := ""
		if rs.Resource != nil {
			svcName = serviceName(rs.Resource.Attributes)
		}

		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				startTime := time.Unix(0, int64(s.StartTimeUnixNano))
				endTime := time.Unix(0, int64(s.EndTimeUnixNano))

				span := &model.Span{
					TraceID:      traceIDToHex(s.TraceId),
					SpanID:       spanIDToHex(s.SpanId),
					ParentSpanID: spanIDToHex(s.ParentSpanId),
					Name:         s.Name,
					ServiceName:  svcName,
					StartTime:    startTime,
					EndTime:      endTime,
					Duration:     endTime.Sub(startTime),
					Attributes:   attrsToMap(s.Attributes),
				}

				events = append(events, model.Event{
					Type:      model.SignalTrace,
					Timestamp: startTime,
					Span:      span,
				})
			}
		}
	}
	return events
}
