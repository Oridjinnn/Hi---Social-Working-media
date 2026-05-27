package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
)

// ── Waiting messages ──────────────────────────────────────────────────────

var waitingMessages = []string{
	"bribing GitHub with open source contributions...",
	"compiling your vibe...",
	"git push origin trust...",
	"downloading more RAM for your dreams...",
	"asking the internet very nicely...",
	"sudo give me access...",
	"your code is being judged by senior devs...",
	"checking if you remembered to save...",
	"rm -rf doubt/",
	"forking the universe...",
	"merging your soul into main...",
	"fixing merge conflicts with destiny...",
	"running npm install on your career...",
	"closing 47 browser tabs to focus...",
	"debugging the authentication flow of life...",
	"your GitHub is being summoned...",
	"404: patience not found, trying again...",
	"sending smoke signals to GitHub servers...",
	"turning coffee into tokens...",
	"reticulating splines...",
}

// ── Client ID Prompt Model ─────────────────────────────────────────────────

type ClientIDResult struct {
	ClientID string
	Err      error
}

type ClientIDPromptModel struct {
	textInput textinput.Model
	err       error
	done      bool
	submitted bool
}

func NewClientIDPromptModel() ClientIDPromptModel {
	ti := textinput.New()
	ti.Placeholder = "Iv1.xxxxxxxxxxxxxxxxxxxx"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 48
	ti.PromptStyle = lipgloss.NewStyle().Foreground(Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	return ClientIDPromptModel{
		textInput: ti,
	}
}

func (m ClientIDPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ClientIDPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.err = fmt.Errorf("setup cancelled")
			m.done = true
			return m, tea.Quit
		case "enter":
			val := strings.TrimSpace(m.textInput.Value())
			if val == "" {
				m.err = fmt.Errorf("client ID cannot be empty")
				m.done = true
				return m, tea.Quit
			}
			m.submitted = true
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m ClientIDPromptModel) View() string {
	if m.done {
		return ""
	}

	stepStyle := lipgloss.NewStyle().Foreground(Foreground)
	labelStyle := lipgloss.NewStyle().Foreground(Muted)
	urlStyle := lipgloss.NewStyle().Foreground(Accent).Underline(true)
	inputLabel := lipgloss.NewStyle().Foreground(Primary).Bold(true)

	content := SectionTitleStyle.Render("🔑 GitHub OAuth Setup Required") + "\n\n" +
		labelStyle.Render("  You need a GitHub OAuth App to use hi.") + "\n" +
		labelStyle.Render("  Create one free in 30 seconds:") + "\n\n" +
		stepStyle.Render("  1. Open ") + urlStyle.Render("github.com/settings/developers") + "\n" +
		stepStyle.Render("  2. Click ") + lipgloss.NewStyle().Foreground(Success).Render("\"New OAuth App\"") + "\n" +
		stepStyle.Render("  3. Fill in:") + "\n" +
		labelStyle.Render("       Application name:") + " hi-cli\n" +
		labelStyle.Render("       Homepage URL:     ") + " https://github.com\n" +
		labelStyle.Render("       Callback URL:     ") + " http://localhost\n" +
		stepStyle.Render("  4. Click ") + lipgloss.NewStyle().Foreground(Success).Render("\"Register application\"") + "\n" +
		stepStyle.Render("  5. Copy the ") + lipgloss.NewStyle().Foreground(Warning).Bold(true).Render("Client ID") + "\n\n" +
		RenderDivider(56, Muted) + "\n\n" +
		inputLabel.Render("  ? Paste your Client ID:") + "\n" +
		"  " + m.textInput.View() + "\n\n" +
		CaptionStyle.Render("  enter to confirm · ctrl+c to cancel")

	return "\n" + CardStyle.Width(60).Render(content) + "\n"
}

func (m ClientIDPromptModel) Result() ClientIDResult {
	return ClientIDResult{
		ClientID: strings.TrimSpace(m.textInput.Value()),
		Err:      m.err,
	}
}

// ── Already Authenticated Model ────────────────────────────────────────────

type AlreadyAuthedModel struct {
	username string
}

func NewAlreadyAuthedModel(username string) AlreadyAuthedModel {
	return AlreadyAuthedModel{username: username}
}

func (m AlreadyAuthedModel) Init() tea.Cmd { return nil }

func (m AlreadyAuthedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m AlreadyAuthedModel) View() string {
	content := SectionTitleStyle.Render("✓ already authenticated") + "\n\n" +
		SuccessStyle.Render(fmt.Sprintf("  ⚡ gm, @%s", m.username)) + "\n\n" +
		CaptionStyle.Render("  run 'hi' to open the feed") + "\n" +
		CaptionStyle.Render("  run 'hi auth login --force' to re-auth")

	return "\n" + CardStyle.Width(52).BorderForeground(Success).Render(content) + "\n"
}

// ── Waiting for Device Auth ─────────────────────────────────────────────────

type AuthModel struct {
	spinner   spinner.Model
	code      string
	verifyURL string
	msgIndex  int
	tick      int
	done      bool
	username  string
	err       error
}

type AuthSuccessMsg struct{ Username string }
type AuthErrMsg struct{ Err error }
type authTickMsg struct{}

func NewAuthModel(code, verifyURL string) AuthModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(Primary)

	return AuthModel{
		spinner:   sp,
		code:      code,
		verifyURL: verifyURL,
	}
}

func (m AuthModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickEvery(4*time.Second),
	)
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return authTickMsg{}
	})
}

func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case authTickMsg:
		m.msgIndex = (m.msgIndex + 1) % len(waitingMessages)
		m.tick++
		return m, tickEvery(4 * time.Second)

	case AuthSuccessMsg:
		m.done = true
		m.username = msg.Username
		return m, tea.Quit

	case AuthErrMsg:
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

func (m AuthModel) View() string {
	if m.done {
		return renderSuccess(m.username)
	}
	if m.err != nil {
		return renderAuthError(m.err)
	}
	return renderAuthWaiting(m.spinner, m.code, m.verifyURL, waitingMessages[m.msgIndex])
}

// ── Renderers ──────────────────────────────────────────────────────────────

func renderAuthWaiting(sp spinner.Model, code, verifyURL, waitMsg string) string {
	labelStyle := lipgloss.NewStyle().Foreground(Muted)
	codeStyle := lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Primary).
		Bold(true).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1)
	urlStyle := lipgloss.NewStyle().Foreground(Success).Underline(true)
	waitStyle := lipgloss.NewStyle().Foreground(Muted).Italic(true)
	clipboardStyle := lipgloss.NewStyle().Foreground(Warning).Italic(true)

	content := SectionTitleStyle.Render("Login with GitHub") + "\n\n" +
		labelStyle.Render("Step 1 — Open this URL in your browser:") + "\n" +
		urlStyle.Render("  "+verifyURL) + "\n\n" +
		labelStyle.Render("Step 2 — Enter this code:") + "\n" +
		codeStyle.Render(code) + "\n\n" +
		clipboardStyle.Render("  ✓ Copied to clipboard!") + "\n\n" +
		RenderDivider(48, Muted) + "\n\n" +
		sp.View() + " " + waitStyle.Render(waitMsg) + "\n\n" +
		CaptionStyle.Render("ctrl+c to cancel")

	return "\n" + CardStyle.Width(56).Render(content) + "\n"
}

func renderSuccess(username string) string {
	content := SectionTitleStyle.Render("✓ Authenticated") + "\n\n" +
		CaptionStyle.Render(fmt.Sprintf("  ⚡ gm, @%s", username)) + "\n\n" +
		BodyStyle.Render("  Next steps:") + "\n" +
		fmt.Sprintf("   1. Run → %s\n", RenderKeyHint("hi")) +
		fmt.Sprintf("   2. Press → %s to create your first signal\n", RenderKeyHint("n")) +
		CaptionStyle.Render("   3. Describe what you're building") + "\n\n" +
		CaptionStyle.Render("  run 'hi' to open the feed")

	return "\n" + CardStyle.Width(52).BorderForeground(Success).Render(content) + "\n"
}

func renderAuthError(err error) string {
	safe := config.RedactSecrets(err.Error())
	content := SectionTitleStyle.Render("✗ Authentication Failed") + "\n\n" +
		CaptionStyle.Render(fmt.Sprintf("  %s", safe)) + "\n\n" +
		CaptionStyle.Render("  run 'hi auth login' to try again")

	return "\n" + CardStyle.Width(52).BorderForeground(Danger).Render(content) + "\n"
}
