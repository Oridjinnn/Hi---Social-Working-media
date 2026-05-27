package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/models"
)

type ToastModel struct {
	event    models.ConnectionEvent
	visible  bool
	timer    int
	maxTicks int
}

func NewToast(event models.ConnectionEvent) ToastModel {
	return ToastModel{
		event:    event,
		visible:  true,
		maxTicks: 12, // ~6 seconds at 500ms per tick
	}
}

func (m ToastModel) Init() tea.Cmd {
	return m.tick()
}

func (m ToastModel) tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return toastTickMsg{}
	})
}

func (m ToastModel) Update(msg tea.Msg) (ToastModel, tea.Cmd) {
	switch msg := msg.(type) {
	case toastTickMsg:
		m.timer++
		if m.timer >= m.maxTicks {
			m.visible = false
			return m, nil
		}
		return m, m.tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.visible = false
			return m, nil
		}
	}
	return m, nil
}

func (m ToastModel) View() string {
	if !m.visible {
		return ""
	}

	actor := m.event.ActorUsername
	project := fmt.Sprintf("signal #%d", m.event.SignalID)
	if m.event.EventType == "connect" {
		project = "your signal"
	}

	var content string
	var toastStyle lipgloss.Style

	switch m.event.EventType {
	case "connect":
		content = fmt.Sprintf("⚡ New connection\n@%s → %s", actor, project)
		toastStyle = ToastSuccessStyle
	case "view":
		content = fmt.Sprintf("👁 @%s viewed %s", actor, project)
		toastStyle = ToastStyle
	case "auth_required":
		content = "🔒 Authentication required"
		toastStyle = ToastWarningStyle
	default:
		content = fmt.Sprintf("⚡ @%s → %s", actor, project)
		toastStyle = ToastStyle
	}

	help := CaptionStyle.Render("[Esc] Dismiss")

	return "\n" + toastStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			strings.TrimSpace(content),
			help,
		),
	)
}

type toastTickMsg struct{}
