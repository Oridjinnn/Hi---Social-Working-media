package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type appSyncMsg struct {
	source string
	reason string
	at     time.Time
}

func emitAppSyncCmd(source, reason string) tea.Cmd {
	return func() tea.Msg {
		return appSyncMsg{
			source: source,
			reason: reason,
			at:     time.Now(),
		}
	}
}
