package store

import (
	"sync"

	"github.com/user/otlp9s/internal/model"
)

// RingBuffer provides bounded storage for telemetry events.
// When capacity is reached the oldest entries are silently dropped.
// All methods are safe for concurrent use.
type RingBuffer struct {
	mu    sync.RWMutex
	items []model.Event
	cap   int
	head  int // next write position
	len   int // current number of items
}

func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		items: make([]model.Event, capacity),
		cap:   capacity,
	}
}

// Add inserts an event, overwriting the oldest if full.
func (r *RingBuffer) Add(e model.Event) {
	r.mu.Lock()
	r.items[r.head] = e
	r.head = (r.head + 1) % r.cap
	if r.len < r.cap {
		r.len++
	}
	r.mu.Unlock()
}

// Snapshot returns a copy of all stored events in insertion order (oldest first).
func (r *RingBuffer) Snapshot() []model.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Event, r.len)
	start := 0
	if r.len == r.cap {
		start = r.head // oldest item when buffer is full
	}
	for i := 0; i < r.len; i++ {
		out[i] = r.items[(start+i)%r.cap]
	}
	return out
}

// Len returns the current number of stored events.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.len
}
