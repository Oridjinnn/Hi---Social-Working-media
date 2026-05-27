package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/utils"
)

type DetailModel struct {
	signal      *models.Signal
	width       int
	height      int
	showConnect bool
	quitting    bool
}

func NewDetailModel(signal *models.Signal) *DetailModel {
	return &DetailModel{
		signal: signal,
	}
}

func (m *DetailModel) Init() tea.Cmd {
	return nil
}

func (m *DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.quitting = true
			return m, nil
		case "right", "enter", "o":
			if m.signal != nil {
				utils.OpenURL(m.signal.GitHubURL)
			}
		case "c":
			if m.signal != nil {
				return m, func() tea.Msg {
					return ConnectMsg{SignalID: m.signal.ID, Signal: m.signal}
				}
			}
		}
	}
	return m, nil
}

func (m *DetailModel) View() string {
	if m.signal == nil {
		return "Loading..."
	}

	s := m.signal
	width := m.width
	if width <= 0 {
		width = 60
	}
	contentW := width - 8

	// ── Build all sections ─────────────────────────────────────────────────

	// Header
	header := H1Style.Render(fmt.Sprintf("#%d %s", s.ID, s.Title)) + "\n\n" +
		lipgloss.JoinHorizontal(lipgloss.Left,
			StatusColor(string(s.Status)).Render(string(s.Status)),
			"  ",
			SignalTypeStyle.Render(string(s.Type)),
			"  ",
			lipgloss.NewStyle().Foreground(MutedLight).Render("by @"+s.Author.GitHubUsername),
			"  ",
			renderDetailTrustBadge(models.ScoreSignal(*s)),
			renderGhostBadge(s.IsGhost),
		)

	// Divider
	div := RenderDivider(contentW, BorderColor)

	// Metadata section (two columns)
	var metaLines []string
	addField := func(label, value string) {
		lbl := lipgloss.NewStyle().Foreground(Muted).Render(label)
		val := lipgloss.NewStyle().Foreground(Foreground).Render(value)
		metaLines = append(metaLines, fmt.Sprintf("  %s%s", lbl, val))
	}
	addField("Project:", s.Project)
	addField("Stack:", strings.Join(s.Stack, ", "))
	addField("Needs:", strings.Join(s.Needs, ", "))
	addField("Difficulty:", string(s.Difficulty))
	addField("Commitment:", string(s.Commitment))
	addField("Contact:", fmt.Sprintf("%s (%s)", s.Contact, s.ContactURL))
	addField("Created:", s.CreatedAt.Format("Jan 2, 2006"))
	rank := models.ScoreSignal(*s)
	addField("Trust:", fmt.Sprintf("%.0f/100 · %s", rank.Score, rank.Hint))

	metaSection := CardHeaderStyle.Render("📋 Details") + "\n" +
		strings.Join(metaLines, "\n")

	// Stats row
	statsRow := "  " + lipgloss.JoinHorizontal(lipgloss.Left,
		statBox("Views", fmt.Sprintf("%d", s.ViewCount)),
		statBox("Connections", fmt.Sprintf("%d", s.ConnectCount)),
	)

	// Description
	var bodySection string
	if s.Body != "" {
		bodySection = div + "\n" +
			CardHeaderStyle.Render("📝 Description") + "\n" +
			BodyStyle.Render(s.Body)
	}

	// Footer
	footer := HelpStyle.Render(fmt.Sprintf("  %s open  %s connect  %s back",
		RenderKeyHint("→", "enter"),
		RenderKeyHint("c"),
		RenderKeyHint("esc", "q"),
	))

	// ── Single outer card ──────────────────────────────────────────────────
	inner := lipgloss.JoinVertical(lipgloss.Left,
		header,
		div,
		metaSection,
		"",
		statsRow,
		bodySection,
		"",
		footer,
	)

	return "\n" + CardStyle.Width(contentW+4).Render(inner)
}
