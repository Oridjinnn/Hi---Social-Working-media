package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/supabase"
)

type WizardStep int

const (
	StepType WizardStep = iota
	StepProject
	StepStack
	StepNeeds
	StepDifficulty
	StepCommitment
	StepContact
	StepContactURL
	StepConfirm
	totalSteps int = 8 // StepConfirm is step 9 (index 8), so 9 total
)

type WizardModel struct {
	step        WizardStep
	client      *github.Client
	supaClient  *supabase.Client
	cfgUsername string
	cfg         *config.Config
	signal      models.Signal
	textInputs  []textinput.Model
	cursor      int
	choices     []string
	width       int
	height      int
	err         error
	done        bool
}

func NewWizard(client *github.Client, supaClient *supabase.Client, username string, cfg *config.Config) WizardModel {
	ti := make([]textinput.Model, 3)
	for i := range ti {
		ti[i] = textinput.New()
		ti[i].Prompt = "> "
		ti[i].CharLimit = 100
	}
	ti[0].Placeholder = "Project name (e.g., Rewind)"
	ti[0].Focus()

	return WizardModel{
		step:        StepType,
		client:      client,
		supaClient:  supaClient,
		cfgUsername: username,
		cfg:         cfg,
		signal: models.Signal{
			Status:     models.SignalStatusOpen,
			Commitment: models.CommitmentCasual,
			Difficulty: models.DifficultyIntermediate,
			Contact:    models.ContactGitHub,
		},
		textInputs: ti,
		choices:    []string{"contributor", "beginner", "vibe_coder", "hiring", "showcase"},
	}
}

func (m *WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case wizardDoneMsg:
		m.done = true
		return m, nil

	case errorMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			m.done = true
			return m, nil

		case "enter":
			return m.handleEnter()

		case "up":
			if m.step == StepType || m.step == StepDifficulty || m.step == StepCommitment || m.step == StepContact {
				if m.cursor > 0 {
					m.cursor--
				}
			}

		case "down":
			if m.step == StepType || m.step == StepDifficulty || m.step == StepCommitment || m.step == StepContact {
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			}

		case "tab":
			for i := range m.textInputs {
				if m.textInputs[i].Focused() {
					m.textInputs[i].Blur()
					if i+1 < len(m.textInputs) {
						m.textInputs[i+1].Focus()
					}
					break
				}
			}
		}
	}

	if m.step == StepProject || m.step == StepStack || m.step == StepNeeds || m.step == StepContactURL {
		for i := range m.textInputs {
			var cmd tea.Cmd
			m.textInputs[i], cmd = m.textInputs[i].Update(msg)
			if cmd != nil {
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m *WizardModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepType:
		types := []string{"contributor", "beginner", "vibe_coder", "hiring", "showcase"}
		if m.cursor < len(types) {
			m.signal.Type = models.SignalType(types[m.cursor])
		}
		m.step = StepProject
		m.cursor = 0
		m.textInputs[0].Focus()

	case StepProject:
		m.signal.Project = m.textInputs[0].Value()
		m.textInputs[0].Blur()
		m.step = StepStack
		m.textInputs[1].Focus()
		m.textInputs[1].Placeholder = "golang, ai, cli (comma separated)"
		m.textInputs[1].SetValue("")

	case StepStack:
		stackStr := m.textInputs[1].Value()
		if stackStr != "" {
			m.signal.Stack = strings.Split(stackStr, ",")
			for i := range m.signal.Stack {
				m.signal.Stack[i] = strings.TrimSpace(m.signal.Stack[i])
			}
		}
		m.textInputs[1].Blur()
		m.step = StepNeeds
		m.textInputs[2].Focus()
		m.textInputs[2].Placeholder = "tester, docs (comma separated)"
		m.textInputs[2].SetValue("")

	case StepNeeds:
		needsStr := m.textInputs[2].Value()
		if needsStr != "" {
			m.signal.Needs = strings.Split(needsStr, ",")
			for i := range m.signal.Needs {
				m.signal.Needs[i] = strings.TrimSpace(m.signal.Needs[i])
			}
		}
		m.textInputs[2].Blur()
		m.step = StepDifficulty
		m.cursor = 0
		m.choices = []string{"beginner", "intermediate", "advanced"}

	case StepDifficulty:
		difficulties := []string{"beginner", "intermediate", "advanced"}
		if m.cursor < len(difficulties) {
			m.signal.Difficulty = models.DifficultyLevel(difficulties[m.cursor])
		}
		m.step = StepCommitment
		m.cursor = 0
		m.choices = []string{"casual", "part-time", "full-time"}

	case StepCommitment:
		commitments := []string{"casual", "part-time", "full-time"}
		if m.cursor < len(commitments) {
			m.signal.Commitment = models.CommitmentLevel(commitments[m.cursor])
		}
		m.step = StepContact
		m.cursor = 0
		m.choices = []string{"github", "discord", "email", "matrix"}

	case StepContact:
		methods := []string{"github", "discord", "email", "matrix"}
		if m.cursor < len(methods) {
			m.signal.Contact = models.ContactMethod(methods[m.cursor])
		}
		m.step = StepContactURL
		m.textInputs[0].Focus()
		m.textInputs[0].Placeholder = "https://github.com/youruser or contact URL"
		m.textInputs[0].SetValue("")

	case StepContactURL:
		m.signal.ContactURL = m.textInputs[0].Value()
		m.textInputs[0].Blur()
		m.step = StepConfirm

	case StepConfirm:
		return m, m.submit()
	}

	return m, nil
}

func (m *WizardModel) submit() tea.Cmd {
	return func() tea.Msg {
		limit := 20
		tier := "free"
		if m.cfg != nil {
			limit = m.cfg.SignalLimit()
			tier = m.cfg.GetTier()
		}
		if err := m.client.CheckSignalLimit(m.cfgUsername, limit, tier); err != nil {
			return errorMsg{err: err}
		}

		m.signal.Title = fmt.Sprintf("[%s] %s", m.signal.Type, m.signal.Project)
		created, err := m.client.CreateSignal(&m.signal)
		if err != nil {
			return errorMsg{err: fmt.Errorf("creating signal: %w", err)}
		}

		if m.supaClient != nil {
			if err := m.supaClient.UpsertUser(m.cfgUsername); err != nil {
				// non-fatal
			}
		}

		if m.cfg != nil && len(m.cfg.Stack) == 0 {
			m.cfg.Stack = m.signal.Stack
			_ = config.Save(m.cfg)
		}

		return wizardDoneMsg{signal: created}
	}
}

func (m WizardModel) View() string {
	if m.done {
		return "\n" + CardStyle.Width(52).Render(
			SuccessStyle.Render("✓ Signal created successfully!")+"\n\n"+
				CaptionStyle.Render("  Your signal is now live in the feed."),
		)
	}

	// ── Progress bar ───────────────────────────────────────────────────────
	currentStep := int(m.step) + 1
	progressBar := RenderProgressBar(currentStep, 9, 25, Primary)
	progressLabel := CaptionStyle.Render(fmt.Sprintf(" Step %d of 9", currentStep))
	progress := lipgloss.JoinHorizontal(lipgloss.Left, progressBar, progressLabel)

	// ── Step content ───────────────────────────────────────────────────────
	var content string

	switch m.step {
	case StepType:
		content = m.renderSelect("Select signal type:", []string{
			"contributor — Looking for contributors to an open source project",
			"beginner — Beginner-friendly, looking to learn together",
			"vibe_coder — Casual collaboration, no pressure",
			"hiring — Looking to hire someone",
			"showcase — Show off what you're building",
		})

	case StepProject:
		content = fmt.Sprintf("%s\n\n%s", LabelStyle.Render("Project name:"), m.textInputs[0].View())

	case StepStack:
		content = fmt.Sprintf("%s\n\n%s", LabelStyle.Render("Stack tags (comma separated):"), m.textInputs[1].View())

	case StepNeeds:
		content = fmt.Sprintf("%s\n\n%s", LabelStyle.Render("What do you need? (comma separated):"), m.textInputs[2].View())

	case StepDifficulty:
		content = m.renderSelect("Difficulty level:", []string{
			"beginner",
			"intermediate",
			"advanced",
		})

	case StepCommitment:
		content = m.renderSelect("Commitment level:", []string{
			"casual — 1-5 hr/week",
			"part-time — 5-20 hr/week",
			"full-time",
		})

	case StepContact:
		content = m.renderSelect("Preferred contact method:", []string{
			"github",
			"discord",
			"email",
			"matrix",
		})

	case StepContactURL:
		content = fmt.Sprintf("%s\n\n%s", LabelStyle.Render("Contact URL:"), m.textInputs[0].View())

	case StepConfirm:
		content = m.renderConfirm()
	}

	help := CaptionStyle.Render(fmt.Sprintf("  %s navigate  %s select  %s cancel", RenderKeyHint("↑", "↓"), RenderKeyHint("enter"), RenderKeyHint("esc")))

	errBlock := ""
	if m.err != nil {
		errBlock = "\n" + ErrorStyle.Render("  "+m.err.Error()) + "\n"
	}

	return CardStyle.Width(52).Render(lipgloss.JoinVertical(lipgloss.Left,
		H1Style.Render("Create Signal"),
		progress,
		"",
		content,
		errBlock,
		"",
		help,
	))
}

func (m WizardModel) renderSelect(prompt string, options []string) string {
	var b strings.Builder
	b.WriteString(LabelStyle.Render(prompt) + "\n\n")
	for i, opt := range options {
		prefix := "  "
		style := CaptionStyle
		if i == m.cursor {
			prefix = "▸ "
			style = HighlightStyle.Copy().Padding(0, 1)
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(opt)))
	}
	return b.String()
}

func (m WizardModel) renderConfirm() string {
	var b strings.Builder
	b.WriteString(CardHeaderStyle.Render("Review your signal") + "\n")

	fields := []struct{ label, value string }{
		{"Type", string(m.signal.Type)},
		{"Project", m.signal.Project},
		{"Stack", strings.Join(m.signal.Stack, ", ")},
		{"Needs", strings.Join(m.signal.Needs, ", ")},
		{"Difficulty", string(m.signal.Difficulty)},
		{"Commitment", string(m.signal.Commitment)},
		{"Contact", fmt.Sprintf("%s (%s)", m.signal.Contact, m.signal.ContactURL)},
	}

	for _, f := range fields {
		label := lipgloss.NewStyle().Foreground(Muted).Width(12).Render(f.label)
		value := lipgloss.NewStyle().Foreground(Foreground).Render(f.value)
		b.WriteString(fmt.Sprintf("  %s %s\n", label, value))
	}

	b.WriteString("\n" + CaptionStyle.Render(fmt.Sprintf("  %s submit  %s cancel", RenderKeyHint("enter"), RenderKeyHint("esc"))))
	return b.String()
}

type wizardDoneMsg struct {
	signal *models.Signal
}
