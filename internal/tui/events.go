package tui

import tea "github.com/charmbracelet/bubbletea"

// tickMsg triggers a periodic refresh of the TUI.
type tickMsg struct{}

// newEventsMsg signals that new telemetry events are available.
type newEventsMsg struct{}

// waitForEvents returns a Cmd that blocks until the notify channel fires,
// then sends a newEventsMsg to trigger a re-render.
func waitForEvents(notify <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-notify
		return newEventsMsg{}
	}
}
