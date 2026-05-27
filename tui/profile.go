package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/models"
)

type ProfileModel struct {
	user         *models.User
	signals      []models.Signal
	width        int
	height       int
	showOwn      bool
	username     string
	cfg          *config.Config
	editing      bool
	editCursor   int
	inputs       []textinput.Model
}

func NewProfileModel(user *models.User, signals []models.Signal, showOwn bool, username string, cfg *config.Config) ProfileModel {
	inputs := make([]textinput.Model, 2)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Your bio..."
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Stack (go, ai, cli)..."

	return ProfileModel{
		user:     user,
		signals:  signals,
		showOwn:  showOwn,
		username: username,
		cfg:      cfg,
		inputs:   inputs,
	}
}

func (m ProfileModel) Init() tea.Cmd {
	return nil
}

func (m ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "esc":
				m.editing = false
				return m, nil
			case "tab", "down":
				m.inputs[m.editCursor].Blur()
				m.editCursor = (m.editCursor + 1) % len(m.inputs)
				m.inputs[m.editCursor].Focus()
				return m, nil
			case "enter":
				if m.cfg != nil {
					m.cfg.Stack = strings.Split(m.inputs[1].Value(), ",")
				}
				m.editing = false
				return m, nil
			}

			var cmds []tea.Cmd
			for i := range m.inputs {
				var cmd tea.Cmd
				m.inputs[i], cmd = m.inputs[i].Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "e":
			if m.showOwn {
				m.editing = true
				m.editCursor = 0
				m.inputs[0].Focus()
				if m.cfg != nil {
					m.inputs[1].SetValue(strings.Join(m.cfg.Stack, ", "))
				}
				return m, nil
			}
		case "q", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ProfileModel) View() string {
	if m.user == nil {
		if m.showOwn {
			return m.renderOwnProfile()
		}
		return "  Loading profile..."
	}

	return m.renderFullProfile()
}

func (m ProfileModel) renderOwnProfile() string {
	if m.editing {
		return m.renderEditMode()
	}

	// Header
	header := lipgloss.JoinVertical(lipgloss.Left,
		H1Style.Render(fmt.Sprintf("@%s", m.username)),
		SubtitleStyle.Render("Your Personal Developer Identity"),
	)

	// Stat cards
	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		statBox("Tier", m.cfg.GetTier()),
		statBox("Focus", strings.Join(m.cfg.Stack, " · ")),
	)

	// Intent section — show user's own signals if available
	var intentContent string
	if len(m.signals) == 0 {
		intentContent = CaptionStyle.Render("  No active signals. Press 'n' in Feed to post one.")
	} else {
		var signalLines []string
		for i, s := range m.signals {
			if i >= 5 {
				break
			}
			statusDot := lipgloss.NewStyle().Foreground(Success).Render("●")
			if string(s.Status) == "filled" {
				statusDot = lipgloss.NewStyle().Foreground(Muted).Render("●")
			} else if string(s.Status) == "expired" {
				statusDot = lipgloss.NewStyle().Foreground(Danger).Render("●")
			}
			signalLines = append(signalLines, fmt.Sprintf("  %s #%d %s [%s]", statusDot, s.ID, s.Title, s.Status))
		}
		intentContent = strings.Join(signalLines, "\n")
	}
	intent := CardStyle.Width(m.width - 4).Render(
		CardHeaderStyle.Render("PERSONAL INTENT") + "\n" +
			intentContent,
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		"  "+stats,
		"",
		intent,
		"",
		HelpStyle.Render(fmt.Sprintf("  %s edit profile  %s switch tabs", RenderKeyHint("e"), RenderKeyHint("tab"))),
	)

	return "\n" + content
}

func (m ProfileModel) renderEditMode() string {
	title := CardHeaderStyle.Render("Edit HI.OS Profile")

	var b strings.Builder
	b.WriteString(title + "\n\n")

	labels := []string{"Bio", "Stack"}
	for i, input := range m.inputs {
		label := LabelStyle.Render("  " + labels[i] + ":")
		b.WriteString(label + "\n" + "  " + input.View() + "\n\n")
	}

	b.WriteString(CaptionStyle.Render(fmt.Sprintf("  %s save  %s cancel  %s move", RenderKeyHint("enter"), RenderKeyHint("esc"), RenderKeyHint("tab"))))

	return CardStyle.Width(m.width - 4).Render(b.String())
}

func (m ProfileModel) renderFullProfile() string {
	u := m.user

	// Header with badges
	header := H1Style.Render(fmt.Sprintf("@%s", u.GitHubUsername))

	if m.showOwn {
		header += " " + SuccessStyle.Render("(you)")
	}

	if u.IsSupporter {
		header += " " + ButtonSecondary.Render("★ supporter")
	}
	if u.IsSeedUser {
		header += " " + ButtonSecondary.Render("● seed")
	}

	bio := ""
	if u.Bio != "" {
		bio = SubtitleStyle.Render(u.Bio)
	}

	// Stats row using statBox
	stats := lipgloss.JoinHorizontal(lipgloss.Left,
		statBox("Signals", fmt.Sprintf("%d", u.SignalCount)),
		statBox("Connections", fmt.Sprintf("%d", u.ConnectionCount)),
		statBox("Successes", fmt.Sprintf("%d", u.SuccessCount)),
	)

	// GitHub info card
	ghCard := CardStyle.Width(m.width - 4).Render(
		CardHeaderStyle.Render("🐙 GitHub") + "\n" +
			BodyStyle.Render(fmt.Sprintf("  Followers: %d  ·  Public repos: %d", u.Followers, u.PublicRepos)),
	)

	// Recent signals
	signalsSection := ""
	if len(m.signals) > 0 && m.showOwn {
		signalsContent := CardHeaderStyle.Render("Recent Signals") + "\n"
		for i, s := range m.signals {
			if i >= 5 {
				break
			}
			statusDot := lipgloss.NewStyle().Foreground(Success).Render("●")
			if string(s.Status) == "filled" {
				statusDot = lipgloss.NewStyle().Foreground(Muted).Render("●")
			} else if string(s.Status) == "expired" {
				statusDot = lipgloss.NewStyle().Foreground(Danger).Render("●")
			}
			signalsContent += fmt.Sprintf("  %s #%d %s [%s]\n", statusDot, s.ID, s.Title, s.Status)
		}
		signalsSection = CardStyle.Width(m.width - 4).Render(signalsContent)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		bio,
		"",
		"  "+stats,
		"",
		ghCard,
		signalsSection,
		"",
		HelpStyle.Render(fmt.Sprintf("  %s back", RenderKeyHint("q"))),
	)

	return "\n" + content
}