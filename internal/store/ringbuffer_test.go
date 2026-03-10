package store

import (
	"sync"
	"testing"
	"time"

	"github.com/user/otlp9s/internal/model"
)

func makeEvent(typ model.SignalType, name string) model.Event {
	ev := model.Event{Type: typ, Timestamp: time.Now()}
	switch typ {
	case model.SignalTrace:
		ev.Span = &model.Span{Name: name}
	case model.SignalMetric:
		ev.Metric = &model.Metric{Name: name}
	case model.SignalLog:
		ev.Log = &model.LogRecord{Body: name}
	}
	return ev
}

func TestRingBuffer_AddAndLen(t *testing.T) {
	rb := NewRingBuffer(5)
	if rb.Len() != 0 {
		t.Fatalf("expected len 0, got %d", rb.Len())
	}

	rb.Add(makeEvent(model.SignalTrace, "a"))
	rb.Add(makeEvent(model.SignalTrace, "b"))
	rb.Add(makeEvent(model.SignalTrace, "c"))

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}
}

func TestRingBuffer_SnapshotOrder(t *testing.T) {
	rb := NewRingBuffer(5)

	for i := 0; i < 5; i++ {
		rb.Add(makeEvent(model.SignalTrace, string(rune('a'+i))))
	}

	snap := rb.Snapshot()
	if len(snap) != 5 {
		t.Fatalf("expected 5 items, got %d", len(snap))
	}

	for i, ev := range snap {
		want := string(rune('a' + i))
		if ev.Span.Name != want {
			t.Errorf("snap[%d]: expected %q, got %q", i, want, ev.Span.Name)
		}
	}
}

func TestRingBuffer_Eviction(t *testing.T) {
	rb := NewRingBuffer(3)

	// Add 5 items to a buffer of capacity 3.
	for i := 0; i < 5; i++ {
		rb.Add(makeEvent(model.SignalTrace, string(rune('a'+i))))
	}

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}

	snap := rb.Snapshot()
	// Should contain c, d, e (oldest a, b evicted).
	expected := []string{"c", "d", "e"}
	for i, ev := range snap {
		if ev.Span.Name != expected[i] {
			t.Errorf("snap[%d]: expected %q, got %q", i, expected[i], ev.Span.Name)
		}
	}
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := NewRingBuffer(100)
	var wg sync.WaitGroup

	// Concurrent writers.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.Add(makeEvent(model.SignalTrace, "x"))
			}
		}()
	}

	// Concurrent readers.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = rb.Snapshot()
				_ = rb.Len()
			}
		}()
	}

	wg.Wait()

	if rb.Len() != 100 {
		t.Fatalf("expected len 100 (capped), got %d", rb.Len())
	}
}

func TestRingBuffer_SnapshotIsACopy(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Add(makeEvent(model.SignalTrace, "a"))

	snap1 := rb.Snapshot()
	rb.Add(makeEvent(model.SignalTrace, "b"))
	snap2 := rb.Snapshot()

	if len(snap1) != 1 {
		t.Fatalf("snap1 should have 1 item, got %d", len(snap1))
	}
	if len(snap2) != 2 {
		t.Fatalf("snap2 should have 2 items, got %d", len(snap2))
	}
}
