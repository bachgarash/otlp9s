package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/otlp9s/internal/model"
	"github.com/user/otlp9s/internal/pipeline"
)

type tab int

const (
	tabTraces tab = iota
	tabMetrics
	tabLogs
)

// selection holds a deep copy of the item pinned with Enter.
type selection struct {
	span      *model.Span
	traceRows []spanRow
	traceID   string
	isHeader  bool
	metric    *model.Metric
	log       *model.LogRecord
}

type Model struct {
	router *pipeline.Router

	activeTab tab
	cursor    int
	width     int
	height    int

	spans    []model.Event
	spanRows []spanRow
	metrics  []model.Event
	logs     []model.Event

	sel          *selection // pinned item for right pane
	filterText   string
	filterActive bool

	// streaming: when true data refreshes and cursor follows tail.
	// j/k freezes everything (like k9s). s/G resumes.
	streaming bool
}

func NewModel(router *pipeline.Router) Model {
	return Model{
		router: router,
		width:  120,
		height: 40,
		streaming: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForEvents(m.router.Notify),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.streaming {
			m.refreshData()
		}
		return m, tickCmd()

	case newEventsMsg:
		if m.streaming {
			m.refreshData()
		}
		return m, waitForEvents(m.router.Notify)

	case tea.KeyMsg:
		if m.filterActive {
			return m.handleFilterKey(msg)
		}
		return m.handleNormalKey(msg)
	}
	return m, nil
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	// --- Navigation ---
	case "j", "down":
		m.cursor++
		m.clampCursor()
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "pgdown":
		m.cursor += m.pageSize()
		m.clampCursor()
	case "pgup":
		m.cursor -= m.pageSize()
		if m.cursor < 0 {
			m.cursor = 0
		}

	// --- Tab switching ---
	case "1":
		m.activeTab = tabTraces
		m.cursor = 0
		m.sel = nil
	case "2":
		m.activeTab = tabMetrics
		m.cursor = 0
		m.sel = nil
	case "3":
		m.activeTab = tabLogs
		m.cursor = 0
		m.sel = nil
	case "tab", "right":
		m.activeTab = (m.activeTab + 1) % 3
		m.cursor = 0
		m.sel = nil
	case "shift+tab", "left":
		m.activeTab = (m.activeTab + 2) % 3
		m.cursor = 0
		m.sel = nil

	// --- Selection ---
	case "enter":
		m.pinSelection()
	case "esc":
		m.sel = nil

	// --- Toggle streaming ---
	case "s":
		m.streaming = !m.streaming
		if m.streaming {
			m.refreshData()
		}
	case "f", "G":
		m.streaming = true
		m.refreshData()

	case "/":
		m.filterActive = true
	}
	return m, nil
}

func (m *Model) pinSelection() {
	switch m.activeTab {
	case tabTraces:
		if m.cursor < 0 || m.cursor >= len(m.spanRows) {
			return
		}
		row := m.spanRows[m.cursor]
		s := &selection{traceID: row.TraceID, isHeader: row.IsHeader}
		if row.Span != nil {
			c := *row.Span
			c.Attributes = copyMap(row.Span.Attributes)
			s.span = &c
		}
		for _, r := range m.spanRows {
			if r.TraceID == row.TraceID && !r.IsHeader && r.Span != nil {
				sc := *r.Span
				sc.Attributes = copyMap(r.Span.Attributes)
				s.traceRows = append(s.traceRows, spanRow{
					Span:   &sc,
					Depth:  r.Depth,
					IsLast: r.IsLast,
				})
			}
		}
		m.sel = s

	case tabMetrics:
		if m.cursor < 0 || m.cursor >= len(m.metrics) {
			return
		}
		if met := m.metrics[m.cursor].Metric; met != nil {
			c := *met
			c.Attributes = copyMap(met.Attributes)
			c.ResourceAttributes = copyMap(met.ResourceAttributes)
			m.sel = &selection{metric: &c}
		}

	case tabLogs:
		if m.cursor < 0 || m.cursor >= len(m.logs) {
			return
		}
		if l := m.logs[m.cursor].Log; l != nil {
			c := *l
			c.Attributes = copyMap(l.Attributes)
			m.sel = &selection{log: &c}
		}
	}
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filterActive = false
		m.cursor = 0
		m.sel = nil
		m.refreshData()
	case "esc":
		m.filterActive = false
		m.filterText = ""
		m.cursor = 0
		m.sel = nil
		m.refreshData()
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
		}
	case "ctrl+u":
		m.filterText = ""
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.filterText += msg.String()
		}
	}
	return m, nil
}

func (m *Model) refreshData() {
	snapshot := m.router.Buffer().Snapshot()

	m.spans = m.spans[:0]
	m.metrics = m.metrics[:0]
	m.logs = m.logs[:0]

	for _, ev := range snapshot {
		if !filter(ev, m.filterText) {
			continue
		}
		switch ev.Type {
		case model.SignalTrace:
			m.spans = append(m.spans, ev)
		case model.SignalMetric:
			m.metrics = append(m.metrics, ev)
		case model.SignalLog:
			m.logs = append(m.logs, ev)
		}
	}

	m.spanRows = buildTraceTree(m.spans)

	// Auto-scroll to end (only called when streaming).
	m.cursor = m.listLen() - 1
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) clampCursor() {
	n := m.listLen() - 1
	if n < 0 {
		n = 0
	}
	if m.cursor > n {
		m.cursor = n
	}
}

func (m Model) listLen() int {
	switch m.activeTab {
	case tabTraces:
		return len(m.spanRows)
	case tabMetrics:
		return len(m.metrics)
	case tabLogs:
		return len(m.logs)
	}
	return 0
}

func (m Model) pageSize() int {
	ps := m.height - 10
	if ps < 5 {
		ps = 5
	}
	return ps
}

func (m Model) View() string {
	if m.width == 0 {
		return "initializing..."
	}

	spanRate, metricRate, logRate := m.router.Rates()
	header := renderHeader(spanRate, metricRate, logRate, m.width, !m.streaming)
	tabs := renderTabs(m.activeTab, len(m.spans), len(m.metrics), len(m.logs))

	var filterBar string
	if m.filterActive {
		filterBar = filterPromptStyle.Render("filter: ") +
			filterTextStyle.Render(m.filterText) +
			lipgloss.NewStyle().Foreground(colorPrimary).Render("▌")
	} else if m.filterText != "" {
		filterBar = filterPromptStyle.Render("filter: ") +
			filterTextStyle.Render(m.filterText) +
			lipgloss.NewStyle().Foreground(colorDim).Render("  (/ edit  esc clear)")
	}

	listHeight := m.height - 8
	if filterBar != "" {
		listHeight--
	}
	if listHeight < 5 {
		listHeight = 5
	}

	listWidth := m.width*2/5 - 3
	detailWidth := m.width - listWidth - 7

	var listContent, detailContent, listTitle string

	switch m.activeTab {
	case tabTraces:
		listTitle = "Traces"
		listContent = renderTraceList(m.spanRows, m.cursor, listHeight, listWidth)
		if m.sel != nil && m.sel.span != nil {
			if m.sel.isHeader {
				detailContent = renderTraceOverview(m.sel.traceID, m.sel.traceRows)
			} else {
				detailContent = renderSpanInContext(m.sel.span, m.sel.traceRows)
			}
		} else {
			detailContent = emptyStyle.Render("← Navigate with j/k, press Enter to inspect")
		}

	case tabMetrics:
		listTitle = "Metrics"
		listContent = renderMetricList(m.metrics, m.cursor, listHeight, listWidth)
		if m.sel != nil && m.sel.metric != nil {
			detailContent = renderMetricDetail(m.sel.metric)
		} else {
			detailContent = emptyStyle.Render("← Navigate with j/k, press Enter to inspect")
		}

	case tabLogs:
		listTitle = "Log Records"
		listContent = renderLogList(m.logs, m.cursor, listHeight, listWidth)
		if m.sel != nil && m.sel.log != nil {
			detailContent = renderLogDetail(m.sel.log)
		} else {
			detailContent = emptyStyle.Render("← Navigate with j/k, press Enter to inspect")
		}
	}

	listPanel := panelStyle.Width(listWidth).Height(listHeight).
		BorderForeground(colorBorder).
		Render(panelTitleStyle.Render(listTitle) + "\n" + listContent)

	detailPanel := panelStyle.Width(detailWidth).Height(listHeight).
		BorderForeground(colorBorder).
		Render(panelTitleStyle.Render("Details") + "\n" + detailContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)

	var status string
	if m.streaming {
		status = lipgloss.NewStyle().Bold(true).Foreground(colorGreen).Render(" ▼ LIVE ")
	} else {
		status = lipgloss.NewStyle().Bold(true).Foreground(colorYellow).Render(" ■ PAUSED  s to toggle ")
	}

	help := helpStyle.Render(
		"q quit  s resume  enter inspect  esc clear  j/k navigate  / filter",
	) + status

	parts := []string{header, tabs}
	if filterBar != "" {
		parts = append(parts, filterBar)
	}
	parts = append(parts, panels, help)

	return strings.Join(parts, "\n")
}
