package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/otlp9s/internal/model"
)

// sortedKeys returns map keys in sorted order for stable rendering.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- Color palette ---

var (
	colorPrimary   = lipgloss.Color("#7C3AED") // purple
	colorSecondary = lipgloss.Color("#06B6D4") // cyan
	colorMuted     = lipgloss.Color("#6B7280") // gray
	colorText      = lipgloss.Color("#E5E7EB") // light gray
	colorDim       = lipgloss.Color("#4B5563") // dim gray
	colorGreen     = lipgloss.Color("#10B981")
	colorYellow    = lipgloss.Color("#F59E0B")
	colorRed       = lipgloss.Color("#EF4444")
	colorOrange    = lipgloss.Color("#F97316")
	colorBlue      = lipgloss.Color("#3B82F6")
	colorSurface   = lipgloss.Color("#1F2937") // dark bg
	colorBorder    = lipgloss.Color("#374151")
)

// --- Styles ---

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			PaddingLeft(1)

	statsLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	statsValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 2)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 2)

	tabCountStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	listCursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary)

	listNormalStyle = lipgloss.NewStyle().
			Foreground(colorText)

	listTimestampStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	listServiceStyle = lipgloss.NewStyle().
				Foreground(colorSecondary)

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginBottom(1)

	detailKeyStyle = lipgloss.NewStyle().
			Foreground(colorBlue)

	detailValStyle = lipgloss.NewStyle().
			Foreground(colorText)

	detailSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorYellow).
				MarginTop(1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMuted).
			PaddingLeft(1)

	filterPromptStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true).
				PaddingLeft(1)

	filterTextStyle = lipgloss.NewStyle().
			Foreground(colorText)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			PaddingLeft(1)

	emptyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			PaddingLeft(2).
			PaddingTop(1)

	severityStyles = map[string]lipgloss.Style{
		"FATAL": lipgloss.NewStyle().Bold(true).Foreground(colorRed),
		"ERROR": lipgloss.NewStyle().Bold(true).Foreground(colorRed),
		"WARN":  lipgloss.NewStyle().Foreground(colorOrange),
		"INFO":  lipgloss.NewStyle().Foreground(colorGreen),
		"DEBUG": lipgloss.NewStyle().Foreground(colorMuted),
		"TRACE": lipgloss.NewStyle().Foreground(colorDim),
	}
)

func severityStyle(sev string) lipgloss.Style {
	if s, ok := severityStyles[strings.ToUpper(sev)]; ok {
		return s
	}
	return lipgloss.NewStyle().Foreground(colorText)
}

// --- Header ---

func renderHeader(spanRate, metricRate, logRate int64, width int, paused bool) string {
	title := headerStyle.Render("◈ otlp9s")

	var statusIndicator string
	if paused {
		statusIndicator = lipgloss.NewStyle().Bold(true).Foreground(colorYellow).Render(" ⏸ PAUSED ")
	} else {
		statusIndicator = lipgloss.NewStyle().Foreground(colorGreen).Render(" ● ")
	}

	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		statusIndicator,
		statsLabelStyle.Render("spans/s "),
		statsValueStyle.Render(fmt.Sprintf("%-6d", spanRate)),
		statsLabelStyle.Render("  metrics/s "),
		statsValueStyle.Render(fmt.Sprintf("%-6d", metricRate)),
		statsLabelStyle.Render("  logs/s "),
		statsValueStyle.Render(fmt.Sprintf("%-6d", logRate)),
	)

	separator := lipgloss.NewStyle().Foreground(colorBorder).
		Render(strings.Repeat("─", max(0, width-2)))

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, title, stats),
		separator,
	)
}

// --- Tabs ---

func renderTabs(active tab, spanCount, metricCount, logCount int) string {
	type tabInfo struct {
		t     tab
		name  string
		count int
		icon  string
	}
	tabs := []tabInfo{
		{tabTraces, "Traces", spanCount, "⬡"},
		{tabMetrics, "Metrics", metricCount, "◆"},
		{tabLogs, "Logs", logCount, "▤"},
	}

	var parts []string
	for _, t := range tabs {
		label := fmt.Sprintf("%s %s %s",
			t.icon, t.name,
			tabCountStyle.Render(fmt.Sprintf("(%d)", t.count)),
		)
		if t.t == active {
			parts = append(parts, tabActiveStyle.Render(label))
		} else {
			parts = append(parts, tabInactiveStyle.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// --- Trace list (tree-grouped) + detail ---

var (
	traceHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary)

	treeConnectorStyle = lipgloss.NewStyle().
				Foreground(colorDim)
)

func renderTraceList(rows []spanRow, cursor int, height int, width int) string {
	if len(rows) == 0 {
		return emptyStyle.Render("Waiting for traces...")
	}

	var b strings.Builder
	start, end := visibleRange(cursor, len(rows), height)

	for i := start; i < end; i++ {
		row := rows[i]

		if row.IsHeader {
			svc := ""
			if row.Span != nil {
				svc = row.Span.ServiceName
			}
			tid := row.TraceID
			if len(tid) > 12 {
				tid = tid[:12]
			}
			line := fmt.Sprintf("─ trace %s  %s", tid, svc)
			if i == cursor {
				b.WriteString(listCursorStyle.Render(padRow("▸"+line, width)))
			} else {
				b.WriteString(traceHeaderStyle.Render(" " + line))
			}
			b.WriteString("\n")
			continue
		}

		s := row.Span
		if s == nil {
			continue
		}

		var prefix string
		if row.Depth > 0 {
			indent := strings.Repeat("│ ", row.Depth-1)
			if row.IsLast {
				prefix = indent + "└─"
			} else {
				prefix = indent + "├─"
			}
		}
		prefix = treeConnectorStyle.Render(prefix)

		name := truncate(s.Name, 22)
		dur := formatDuration(s.Duration)

		line := fmt.Sprintf("%s %s %s", prefix, name, dur)
		if i == cursor {
			b.WriteString(listCursorStyle.Render(padRow("▸ "+line, width)))
		} else {
			b.WriteString(listNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString(renderScrollHint(cursor, len(rows), height))
	return b.String()
}

// renderTraceOverview shows the full trace tree when a trace header is selected.
func renderTraceOverview(traceID string, rows []spanRow) string {
	var b strings.Builder

	tid := traceID
	if len(tid) > 16 {
		tid = tid[:16] + "…"
	}
	b.WriteString(detailHeaderStyle.Render("  ⬡ Trace " + tid))
	b.WriteString("\n")

	if len(rows) == 0 {
		b.WriteString(emptyStyle.Render("  No spans in this trace"))
		return b.String()
	}

	writeKV(&b, "spans", fmt.Sprintf("%d", len(rows)))

	// Compute total duration from earliest start to latest end.
	earliest := rows[0].Span.StartTime
	latest := rows[0].Span.EndTime
	for _, r := range rows[1:] {
		if r.Span.StartTime.Before(earliest) {
			earliest = r.Span.StartTime
		}
		if r.Span.EndTime.After(latest) {
			latest = r.Span.EndTime
		}
	}
	writeKV(&b, "duration", formatDuration(latest.Sub(earliest)))

	// Collect unique services.
	svcSet := make(map[string]bool)
	for _, r := range rows {
		svcSet[r.Span.ServiceName] = true
	}
	var svcs []string
	for s := range svcSet {
		svcs = append(svcs, s)
	}
	sort.Strings(svcs)
	writeKV(&b, "services", strings.Join(svcs, ", "))

	// Show the span tree.
	b.WriteString("\n")
	b.WriteString(detailSectionStyle.Render("  Span Tree"))
	b.WriteString("\n")

	for _, r := range rows {
		s := r.Span
		var prefix string
		if r.Depth > 0 {
			indent := strings.Repeat("│ ", r.Depth-1)
			if r.IsLast {
				prefix = indent + "└─"
			} else {
				prefix = indent + "├─"
			}
		}

		svc := listServiceStyle.Render(truncate(s.ServiceName, 10))
		dur := formatDuration(s.Duration)
		name := truncate(s.Name, 20)

		b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
			treeConnectorStyle.Render(prefix),
			svc,
			detailValStyle.Render(name),
			listTimestampStyle.Render(dur),
		))
	}

	return b.String()
}

// renderSpanInContext shows span details + its position in the trace tree.
func renderSpanInContext(s *model.Span, traceRows []spanRow) string {
	if s == nil {
		return emptyStyle.Render("← Select a span to inspect")
	}

	var b strings.Builder

	b.WriteString(detailHeaderStyle.Render("  ⬡ " + s.Name))
	b.WriteString("\n\n")

	writeKV(&b, "service", s.ServiceName)
	writeKV(&b, "duration", formatDuration(s.Duration))
	writeKV(&b, "span", s.SpanID)
	if s.ParentSpanID != "" && s.ParentSpanID != "0000000000000000" {
		writeKV(&b, "parent", s.ParentSpanID)
	}
	writeKV(&b, "start", s.StartTime.Format("15:04:05.000"))
	writeKV(&b, "end", s.EndTime.Format("15:04:05.000"))

	if len(s.Attributes) > 0 {
		b.WriteString("\n")
		b.WriteString(detailSectionStyle.Render("  Attributes"))
		b.WriteString("\n")
		for _, k := range sortedKeys(s.Attributes) {
			writeKV(&b, k, s.Attributes[k])
		}
	}

	// Show position in trace tree — highlight the selected span.
	b.WriteString("\n")
	b.WriteString(detailSectionStyle.Render("  Trace Tree"))
	b.WriteString("\n")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)

	for _, r := range traceRows {
		rs := r.Span
		var prefix string
		if r.Depth > 0 {
			indent := strings.Repeat("│ ", r.Depth-1)
			if r.IsLast {
				prefix = indent + "└─"
			} else {
				prefix = indent + "├─"
			}
		}

		name := truncate(rs.Name, 20)
		dur := formatDuration(rs.Duration)
		svc := truncate(rs.ServiceName, 10)

		line := fmt.Sprintf("  %s %s %s %s",
			treeConnectorStyle.Render(prefix), svc, name, dur)

		if rs.SpanID == s.SpanID {
			b.WriteString(selectedStyle.Render("▸" + line))
		} else {
			b.WriteString(listNormalStyle.Render(" " + line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// --- Metric list + detail ---

func renderMetricList(metrics []model.Event, cursor int, height int, width int) string {
	if len(metrics) == 0 {
		return emptyStyle.Render("Waiting for metrics...")
	}

	var b strings.Builder
	start, end := visibleRange(cursor, len(metrics), height)

	for i := start; i < end; i++ {
		m := metrics[i].Metric
		if m == nil {
			continue
		}

		ts := listTimestampStyle.Render(m.Timestamp.Format("15:04:05"))
		typeBadge := renderTypeBadge(string(m.Type))
		name := truncate(m.Name, 22)
		val := truncate(m.Value, 10)

		line := fmt.Sprintf("%s %s %-22s %s", ts, typeBadge, name, val)
		if i == cursor {
			b.WriteString(listCursorStyle.Render(padRow("▸ "+line, width)))
		} else {
			b.WriteString(listNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString(renderScrollHint(cursor, len(metrics), height))
	return b.String()
}

func renderMetricDetail(m *model.Metric) string {
	if m == nil {
		return emptyStyle.Render("← Select a metric to inspect")
	}

	var b strings.Builder

	b.WriteString(detailHeaderStyle.Render("  ◆ " + m.Name))
	b.WriteString("\n\n")

	writeKV(&b, "type", string(m.Type))
	writeKV(&b, "value", m.Value)
	writeKV(&b, "service", m.ServiceName)
	writeKV(&b, "time", m.Timestamp.Format("15:04:05.000"))

	if len(m.Attributes) > 0 {
		b.WriteString("\n")
		b.WriteString(detailSectionStyle.Render("  Attributes"))
		b.WriteString("\n")
		for _, k := range sortedKeys(m.Attributes) {
			writeKV(&b, k, m.Attributes[k])
		}
	}

	if len(m.ResourceAttributes) > 0 {
		b.WriteString("\n")
		b.WriteString(detailSectionStyle.Render("  Resource"))
		b.WriteString("\n")
		for _, k := range sortedKeys(m.ResourceAttributes) {
			writeKV(&b, k, m.ResourceAttributes[k])
		}
	}

	return b.String()
}

// --- Log list + detail ---

func renderLogList(logs []model.Event, cursor int, height int, width int) string {
	if len(logs) == 0 {
		return emptyStyle.Render("Waiting for logs...")
	}

	var b strings.Builder
	start, end := visibleRange(cursor, len(logs), height)

	for i := start; i < end; i++ {
		l := logs[i].Log
		if l == nil {
			continue
		}

		ts := listTimestampStyle.Render(l.Timestamp.Format("15:04:05"))
		sev := l.Severity
		if sev == "" {
			sev = "---"
		}
		sevStr := severityStyle(sev).Render(fmt.Sprintf("%-5s", truncate(sev, 5)))
		body := truncate(l.Body, 35)

		line := fmt.Sprintf("%s %s %s", ts, sevStr, body)
		if i == cursor {
			b.WriteString(listCursorStyle.Render(padRow("▸ "+line, width)))
		} else {
			b.WriteString(listNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString(renderScrollHint(cursor, len(logs), height))
	return b.String()
}

func renderLogDetail(l *model.LogRecord) string {
	if l == nil {
		return emptyStyle.Render("← Select a log to inspect")
	}

	var b strings.Builder

	sevDisplay := l.Severity
	if sevDisplay == "" {
		sevDisplay = "unknown"
	}
	b.WriteString(detailHeaderStyle.Render("  ▤ " + sevDisplay))
	b.WriteString("\n\n")

	writeKV(&b, "severity", sevDisplay)
	writeKV(&b, "service", l.ServiceName)
	writeKV(&b, "time", l.Timestamp.Format("15:04:05.000"))

	if l.TraceID != "" && l.TraceID != "00000000000000000000000000000000" {
		writeKV(&b, "trace", l.TraceID)
		writeKV(&b, "span", l.SpanID)
	}

	b.WriteString("\n")
	b.WriteString(detailSectionStyle.Render("  Body"))
	b.WriteString("\n")
	// Wrap long body text.
	body := l.Body
	for len(body) > 0 {
		chunk := body
		if len(chunk) > 60 {
			chunk = body[:60]
			body = body[60:]
		} else {
			body = ""
		}
		b.WriteString("  " + detailValStyle.Render(chunk) + "\n")
	}

	if len(l.Attributes) > 0 {
		b.WriteString("\n")
		b.WriteString(detailSectionStyle.Render("  Attributes"))
		b.WriteString("\n")
		for _, k := range sortedKeys(l.Attributes) {
			writeKV(&b, k, l.Attributes[k])
		}
	}

	return b.String()
}

// --- Helpers ---

func writeKV(b *strings.Builder, key, val string) {
	b.WriteString(fmt.Sprintf("  %s %s\n",
		detailKeyStyle.Render(fmt.Sprintf("%-16s", key)),
		detailValStyle.Render(val),
	))
}

func renderTypeBadge(t string) string {
	var color lipgloss.Color
	switch t {
	case "gauge":
		color = colorGreen
	case "sum":
		color = colorBlue
	case "histogram":
		color = colorOrange
	default:
		color = colorMuted
	}
	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%-5s", truncate(t, 5)))
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func visibleRange(cursor, total, height int) (start, end int) {
	if total <= height {
		return 0, total
	}
	// Keep cursor centered when possible.
	half := height / 2
	start = cursor - half
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

func renderScrollHint(cursor, total, height int) string {
	if total <= height {
		return ""
	}
	pos := float64(cursor) / float64(total-1) * 100
	return listTimestampStyle.Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", cursor+1, total, pos))
}

// padRow pads a string with spaces to fill width, so background color spans the full row.
func padRow(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible < width {
		s += strings.Repeat(" ", width-visible)
	}
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
