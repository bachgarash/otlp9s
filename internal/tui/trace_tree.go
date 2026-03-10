package tui

import (
	"sort"
	"time"

	"github.com/user/otlp9s/internal/model"
)

// spanRow is a flattened span with tree metadata for rendering.
type spanRow struct {
	Span     *model.Span
	Depth    int    // nesting level (0 = root)
	IsLast   bool   // last sibling at this depth
	TraceID  string // for grouping header
	IsHeader bool   // true = this row is a trace group header, not a span
}

// buildTraceTree takes a flat list of span events and returns a flattened
// tree-ordered list with depth info. Spans are grouped by trace_id,
// traces ordered by earliest start time (newest last for auto-scroll).
func buildTraceTree(events []model.Event) []spanRow {
	// Group by trace ID.
	type traceInfo struct {
		spans []*model.Span
	}
	traces := make(map[string]*traceInfo)
	var traceOrder []string

	for i := range events {
		s := events[i].Span
		if s == nil {
			continue
		}
		ti, exists := traces[s.TraceID]
		if !exists {
			ti = &traceInfo{}
			traces[s.TraceID] = ti
			traceOrder = append(traceOrder, s.TraceID)
		}
		ti.spans = append(ti.spans, s)
	}

	// Sort traces by earliest span start time (oldest first → newest at bottom).
	sort.Slice(traceOrder, func(i, j int) bool {
		return earliestStart(traces[traceOrder[i]].spans).Before(
			earliestStart(traces[traceOrder[j]].spans))
	})

	// Build flattened rows.
	var rows []spanRow
	for _, tid := range traceOrder {
		ti := traces[tid]

		// Trace header row.
		rows = append(rows, spanRow{
			TraceID:  tid,
			IsHeader: true,
			Span:     ti.spans[0], // for service name in header
		})

		// Build tree within this trace.
		rows = append(rows, flattenTree(ti.spans)...)
	}

	return rows
}

// flattenTree orders spans as a depth-first tree.
func flattenTree(spans []*model.Span) []spanRow {
	if len(spans) == 0 {
		return nil
	}

	// Index by span ID.
	spanIDs := make(map[string]bool, len(spans))
	children := make(map[string][]*model.Span)

	for _, s := range spans {
		spanIDs[s.SpanID] = true
	}

	// Build children map. A span is a root if its parent isn't in this trace.
	var roots []*model.Span
	for _, s := range spans {
		pid := s.ParentSpanID
		if pid == "" || pid == "0000000000000000" || !spanIDs[pid] {
			roots = append(roots, s)
		} else {
			children[pid] = append(children[pid], s)
		}
	}

	// Sort by start time.
	sortByTime := func(sl []*model.Span) {
		sort.Slice(sl, func(i, j int) bool {
			return sl[i].StartTime.Before(sl[j].StartTime)
		})
	}
	sortByTime(roots)
	for k := range children {
		sortByTime(children[k])
	}

	if len(roots) == 0 {
		roots = spans
	}

	// DFS to produce flat rows with depth.
	var rows []spanRow
	var dfs func(s *model.Span, depth int, isLast bool)
	dfs = func(s *model.Span, depth int, isLast bool) {
		rows = append(rows, spanRow{
			Span:    s,
			Depth:   depth,
			IsLast:  isLast,
			TraceID: s.TraceID,
		})
		kids := children[s.SpanID]
		for i, kid := range kids {
			dfs(kid, depth+1, i == len(kids)-1)
		}
	}

	for i, root := range roots {
		dfs(root, 0, i == len(roots)-1)
	}

	return rows
}

func earliestStart(spans []*model.Span) time.Time {
	t := spans[0].StartTime
	for _, s := range spans[1:] {
		if s.StartTime.Before(t) {
			t = s.StartTime
		}
	}
	return t
}
