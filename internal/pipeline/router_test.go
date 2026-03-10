package pipeline

import (
	"testing"
	"time"

	"github.com/user/otlp9s/internal/model"
	"github.com/user/otlp9s/internal/store"
)

func TestRouter_EventRouting(t *testing.T) {
	buf := store.NewRingBuffer(100)
	idx := store.NewTraceIndex()
	r := NewRouter(buf, idx)

	go r.Run()

	// Send a trace event.
	r.Events <- model.Event{
		Type:      model.SignalTrace,
		Timestamp: time.Now(),
		Span: &model.Span{
			TraceID: "t1",
			SpanID:  "s1",
			Name:    "test-span",
		},
	}

	// Send a metric event.
	r.Events <- model.Event{
		Type:      model.SignalMetric,
		Timestamp: time.Now(),
		Metric:    &model.Metric{Name: "test-metric"},
	}

	// Send a log event.
	r.Events <- model.Event{
		Type:      model.SignalLog,
		Timestamp: time.Now(),
		Log:       &model.LogRecord{Body: "test-log"},
	}

	// Wait for events to be processed.
	time.Sleep(50 * time.Millisecond)

	if buf.Len() != 3 {
		t.Fatalf("expected 3 events in buffer, got %d", buf.Len())
	}

	spans := idx.Get("t1")
	if len(spans) != 1 {
		t.Fatalf("expected 1 span in trace index, got %d", len(spans))
	}
	if spans[0].Name != "test-span" {
		t.Errorf("indexed span name: expected test-span, got %s", spans[0].Name)
	}
}

func TestRouter_NotifyChannel(t *testing.T) {
	buf := store.NewRingBuffer(100)
	idx := store.NewTraceIndex()
	r := NewRouter(buf, idx)

	go r.Run()

	r.Events <- model.Event{
		Type:      model.SignalLog,
		Timestamp: time.Now(),
		Log:       &model.LogRecord{Body: "hello"},
	}

	select {
	case <-r.Notify:
		// OK -- got notification.
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for Notify signal")
	}
}

func TestRouter_Rates(t *testing.T) {
	buf := store.NewRingBuffer(100)
	idx := store.NewTraceIndex()
	r := NewRouter(buf, idx)

	// Rates should be zero before any events.
	spans, metrics, logs := r.Rates()
	if spans != 0 || metrics != 0 || logs != 0 {
		t.Errorf("initial rates should be 0, got spans=%d metrics=%d logs=%d", spans, metrics, logs)
	}
}
