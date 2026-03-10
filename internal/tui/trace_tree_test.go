package tui

import (
	"testing"
	"time"

	"github.com/user/otlp9s/internal/model"
)

func makeSpanEvent(traceID, spanID, parentID, name string, start time.Time) model.Event {
	d := 10 * time.Millisecond
	return model.Event{
		Type:      model.SignalTrace,
		Timestamp: start,
		Span: &model.Span{
			TraceID:      traceID,
			SpanID:       spanID,
			ParentSpanID: parentID,
			Name:         name,
			ServiceName:  "test-svc",
			StartTime:    start,
			EndTime:      start.Add(d),
			Duration:     d,
		},
	}
}

func TestBuildTraceTree_SingleTrace(t *testing.T) {
	base := time.Now()
	events := []model.Event{
		makeSpanEvent("t1", "root", "", "root-span", base),
		makeSpanEvent("t1", "child1", "root", "child-1", base.Add(1*time.Millisecond)),
		makeSpanEvent("t1", "child2", "root", "child-2", base.Add(2*time.Millisecond)),
	}

	rows := buildTraceTree(events)

	// Expect: 1 header + 3 spans = 4 rows.
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}

	// First row should be a header.
	if !rows[0].IsHeader {
		t.Error("first row should be a header")
	}
	if rows[0].TraceID != "t1" {
		t.Errorf("header trace ID: expected t1, got %s", rows[0].TraceID)
	}

	// Root span at depth 0.
	if rows[1].Span.Name != "root-span" {
		t.Errorf("expected root-span, got %s", rows[1].Span.Name)
	}
	if rows[1].Depth != 0 {
		t.Errorf("root depth: expected 0, got %d", rows[1].Depth)
	}

	// Children at depth 1.
	if rows[2].Depth != 1 || rows[3].Depth != 1 {
		t.Errorf("children depth: expected 1, got %d and %d", rows[2].Depth, rows[3].Depth)
	}
}

func TestBuildTraceTree_MultipleTraces(t *testing.T) {
	base := time.Now()
	events := []model.Event{
		makeSpanEvent("t2", "s1", "", "later-trace", base.Add(100*time.Millisecond)),
		makeSpanEvent("t1", "s2", "", "earlier-trace", base),
	}

	rows := buildTraceTree(events)

	// 2 headers + 2 spans = 4 rows.
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}

	// Earlier trace should come first.
	if rows[0].TraceID != "t1" {
		t.Errorf("first trace should be t1 (earlier), got %s", rows[0].TraceID)
	}
	if rows[2].TraceID != "t2" {
		t.Errorf("second trace should be t2 (later), got %s", rows[2].TraceID)
	}
}

func TestBuildTraceTree_Empty(t *testing.T) {
	rows := buildTraceTree(nil)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for nil input, got %d", len(rows))
	}
}

func TestBuildTraceTree_DeepNesting(t *testing.T) {
	base := time.Now()
	events := []model.Event{
		makeSpanEvent("t1", "a", "", "root", base),
		makeSpanEvent("t1", "b", "a", "level-1", base.Add(1*time.Millisecond)),
		makeSpanEvent("t1", "c", "b", "level-2", base.Add(2*time.Millisecond)),
		makeSpanEvent("t1", "d", "c", "level-3", base.Add(3*time.Millisecond)),
	}

	rows := buildTraceTree(events)

	// 1 header + 4 spans.
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(rows))
	}

	expectedDepths := []int{0, 1, 2, 3}
	for i, depth := range expectedDepths {
		if rows[i+1].Depth != depth {
			t.Errorf("row %d: expected depth %d, got %d", i+1, depth, rows[i+1].Depth)
		}
	}
}

func TestFlattenTree_OrphanedSpans(t *testing.T) {
	// Spans whose parents aren't in the set should be treated as roots.
	base := time.Now()
	spans := []*model.Span{
		{TraceID: "t1", SpanID: "a", ParentSpanID: "missing", Name: "orphan-1", StartTime: base, EndTime: base.Add(time.Millisecond)},
		{TraceID: "t1", SpanID: "b", ParentSpanID: "missing2", Name: "orphan-2", StartTime: base.Add(time.Millisecond), EndTime: base.Add(2 * time.Millisecond)},
	}

	rows := flattenTree(spans)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Both should be at depth 0 since they're roots.
	for i, r := range rows {
		if r.Depth != 0 {
			t.Errorf("row %d: expected depth 0, got %d", i, r.Depth)
		}
	}
}

func TestBuildTraceTree_IsLastFlag(t *testing.T) {
	base := time.Now()
	events := []model.Event{
		makeSpanEvent("t1", "root", "", "root", base),
		makeSpanEvent("t1", "c1", "root", "child-1", base.Add(1*time.Millisecond)),
		makeSpanEvent("t1", "c2", "root", "child-2", base.Add(2*time.Millisecond)),
	}

	rows := buildTraceTree(events)

	// rows[0] = header, rows[1] = root, rows[2] = child-1, rows[3] = child-2
	if rows[2].IsLast {
		t.Error("child-1 should not be IsLast")
	}
	if !rows[3].IsLast {
		t.Error("child-2 should be IsLast")
	}
}
