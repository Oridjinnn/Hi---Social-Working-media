package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
)

// AuthGateModel is shown when startup auth validation fails.
type AuthGateModel struct {
	report config.AuthReport
	width  int
	height int
}

func NewAuthGateModel(report config.AuthReport) AuthGateModel {
	return AuthGateModel{report: report}
}

func (m AuthGateModel) Init() tea.Cmd { return nil }

func (m AuthGateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m AuthGateModel) View() string {
	title := SectionTitleStyle.Render("🔐 Authentication Required")
	level := string(m.report.Level)
	levelStyle := WarningStyle
	switch m.report.Level {
	case config.AuthInvalid:
		levelStyle = ErrorStyle
	case config.AuthMissing:
		levelStyle = WarningStyle
	case config.AuthDegraded:
		levelStyle = CaptionStyle
	}

	var issues strings.Builder
	for _, issue := range m.report.Issues {
		issues.WriteString(ErrorStyle.Render("  • "+issue) + "\n")
	}

	hints := m.report.FormatAuthHints()
	hintBlock := ""
	if hints != "" {
		hintBlock = "\n" + CardHeaderStyle.Render("How to fix") + "\n" + HelpStyle.Render(hints)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		levelStyle.Render(fmt.Sprintf("  status: %s", level)),
		"",
		issues.String(),
		hintBlock,
		"",
		HelpStyle.Render("  "+RenderKeyHint("q")+" quit"),
	)

	box := CardStyle.Width(64).BorderForeground(Warning).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
