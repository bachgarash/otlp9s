package store

import (
	"sort"
	"testing"

	"github.com/user/otlp9s/internal/model"
)

func TestTraceIndex_AddAndGet(t *testing.T) {
	idx := NewTraceIndex()

	s1 := &model.Span{TraceID: "t1", SpanID: "s1", Name: "root"}
	s2 := &model.Span{TraceID: "t1", SpanID: "s2", Name: "child"}
	s3 := &model.Span{TraceID: "t2", SpanID: "s3", Name: "other"}

	idx.Add(s1)
	idx.Add(s2)
	idx.Add(s3)

	spans := idx.Get("t1")
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans for t1, got %d", len(spans))
	}

	spans = idx.Get("t2")
	if len(spans) != 1 {
		t.Fatalf("expected 1 span for t2, got %d", len(spans))
	}

	spans = idx.Get("nonexistent")
	if len(spans) != 0 {
		t.Fatalf("expected 0 spans for nonexistent, got %d", len(spans))
	}
}

func TestTraceIndex_GetReturnsCopy(t *testing.T) {
	idx := NewTraceIndex()
	idx.Add(&model.Span{TraceID: "t1", SpanID: "s1"})

	spans1 := idx.Get("t1")
	spans1[0] = nil // mutate the returned slice

	spans2 := idx.Get("t1")
	if spans2[0] == nil {
		t.Fatal("Get should return a copy; mutation should not affect index")
	}
}

func TestTraceIndex_TraceIDs(t *testing.T) {
	idx := NewTraceIndex()

	idx.Add(&model.Span{TraceID: "t1", SpanID: "s1"})
	idx.Add(&model.Span{TraceID: "t2", SpanID: "s2"})
	idx.Add(&model.Span{TraceID: "t1", SpanID: "s3"})

	ids := idx.TraceIDs()
	sort.Strings(ids)

	if len(ids) != 2 {
		t.Fatalf("expected 2 trace IDs, got %d", len(ids))
	}
	if ids[0] != "t1" || ids[1] != "t2" {
		t.Fatalf("unexpected trace IDs: %v", ids)
	}
}
