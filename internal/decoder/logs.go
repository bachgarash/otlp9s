package decoder

import (
	"time"

	"github.com/user/otlp9s/internal/model"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// DecodeLogs converts an OTLP log export into internal LogRecord events.
func DecodeLogs(req *logspb.LogsData) []model.Event {
	var events []model.Event

	for _, rl := range req.ResourceLogs {
		svcName := ""
		if rl.Resource != nil {
			svcName = serviceName(rl.Resource.Attributes)
		}

		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				ts := time.Unix(0, int64(lr.TimeUnixNano))
				rec := &model.LogRecord{
					Timestamp:   ts,
					Severity:    lr.SeverityText,
					Body:        anyValueToString(lr.Body),
					Attributes:  attrsToMap(lr.Attributes),
					ServiceName: svcName,
					TraceID:     traceIDToHex(lr.TraceId),
					SpanID:      spanIDToHex(lr.SpanId),
				}

				events = append(events, model.Event{
					Type:      model.SignalLog,
					Timestamp: ts,
					Log:       rec,
				})
			}
		}
	}
	return events
}
