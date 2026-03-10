package pipeline

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/user/otlp9s/internal/model"
	"github.com/user/otlp9s/internal/store"
)

// Router receives decoded events from proxies and distributes them
// to storage and the TUI. It also maintains per-second rate counters.
type Router struct {
	Events chan model.Event // proxies write here
	buf    *store.RingBuffer
	idx    *store.TraceIndex

	// Rate counters (per-second, computed over a sliding window).
	spanCount   atomic.Int64
	metricCount atomic.Int64
	logCount    atomic.Int64
	traceCount  atomic.Int64

	// Snapshot of rates, updated every second.
	mu         sync.RWMutex
	SpanRate   int64
	MetricRate int64
	LogRate    int64
	TraceRate  int64

	// Subscribers receive a signal whenever new events arrive.
	// The TUI listens here to trigger a re-render.
	Notify chan struct{}
}

func NewRouter(buf *store.RingBuffer, idx *store.TraceIndex) *Router {
	return &Router{
		Events: make(chan model.Event, 4096),
		buf:    buf,
		idx:    idx,
		Notify: make(chan struct{}, 1),
	}
}

// Run processes incoming events. Call from a dedicated goroutine.
func (r *Router) Run() {
	go r.rateLoop()

	for ev := range r.Events {
		r.buf.Add(ev)

		switch ev.Type {
		case model.SignalTrace:
			r.spanCount.Add(1)
			if ev.Span != nil {
				r.idx.Add(ev.Span)
			}
		case model.SignalMetric:
			r.metricCount.Add(1)
		case model.SignalLog:
			r.logCount.Add(1)
		}

		// Non-blocking notify to TUI.
		select {
		case r.Notify <- struct{}{}:
		default:
		}
	}
}

// rateLoop samples counters every second and stores the delta as the rate.
func (r *Router) rateLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var prevSpan, prevMetric, prevLog int64

	for range ticker.C {
		curSpan := r.spanCount.Load()
		curMetric := r.metricCount.Load()
		curLog := r.logCount.Load()

		r.mu.Lock()
		r.SpanRate = curSpan - prevSpan
		r.MetricRate = curMetric - prevMetric
		r.LogRate = curLog - prevLog
		r.mu.Unlock()

		prevSpan = curSpan
		prevMetric = curMetric
		prevLog = curLog
	}
}

// Rates returns the current per-second rates.
func (r *Router) Rates() (spans, metrics, logs int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.SpanRate, r.MetricRate, r.LogRate
}

// Buffer returns the underlying ring buffer.
func (r *Router) Buffer() *store.RingBuffer { return r.buf }

// TraceIndex returns the trace index.
func (r *Router) TraceIndex() *store.TraceIndex { return r.idx }
