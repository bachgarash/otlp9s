package store

import (
	"sync"

	"github.com/user/otlp9s/internal/model"
)

// TraceIndex groups spans by trace ID for tree reconstruction.
// It is append-only and bounded by the ring buffer that feeds it;
// when the ring buffer evicts old spans we do not remove them here
// (acceptable for an MVP debugging tool with short sessions).
type TraceIndex struct {
	mu     sync.RWMutex
	traces map[string][]*model.Span
}

func NewTraceIndex() *TraceIndex {
	return &TraceIndex{traces: make(map[string][]*model.Span)}
}

// Add indexes a span under its trace ID.
func (t *TraceIndex) Add(s *model.Span) {
	t.mu.Lock()
	t.traces[s.TraceID] = append(t.traces[s.TraceID], s)
	t.mu.Unlock()
}

// Get returns all spans for a given trace ID.
func (t *TraceIndex) Get(traceID string) []*model.Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	spans := t.traces[traceID]
	out := make([]*model.Span, len(spans))
	copy(out, spans)
	return out
}

// TraceIDs returns all known trace IDs.
func (t *TraceIndex) TraceIDs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	ids := make([]string, 0, len(t.traces))
	for id := range t.traces {
		ids = append(ids, id)
	}
	return ids
}
