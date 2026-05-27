package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
)

// ── Message bubble styles ────────────────────────────────────────────────────

var (
	userBubbleStyle = lipgloss.NewStyle().
			Background(Surface).
			Padding(0, 1, 0, 1)

	otherBubbleStyle = lipgloss.NewStyle().
				Background(SurfaceAlt).
				Padding(0, 1, 0, 1)

	systemMsgStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)
)

type ChatModel struct {
	signal     *models.Signal
	messages   []github.ChatMessage
	input      textinput.Model
	client     *github.Client
	cfg        *config.Config
	width      int
	height     int
	quitting   bool
	loading    bool
	msgCount   int
	limitHit   bool
	lastPolled time.Time
}

func NewChatModel(signal *models.Signal, client *github.Client, cfg *config.Config) *ChatModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.Width = 80
	return &ChatModel{
		signal:     signal,
		input:      ti,
		client:     client,
		cfg:        cfg,
		loading:    true,
		msgCount:   loadConnectionCount(signal.ID),
		lastPolled: time.Now().Add(-1 * time.Hour),
	}
}

type chatTickMsg time.Time

func tickChat(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return chatTickMsg(t)
	})
}

func (m *ChatModel) Init() tea.Cmd {
	interval := time.Duration(m.cfg.PollInterval()) * time.Second
	return tea.Batch(m.loadComments(), tickChat(interval))
}

func (m *ChatModel) loadComments() tea.Cmd {
	return func() tea.Msg {
		msgs, err := m.client.PollComments(int64(m.signal.ID), m.lastPolled)
		if err != nil {
			return errorMsg{err: err}
		}
		return chatMessagesMsg{messages: msgs, posted: false}
	}
}

func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case chatMessagesMsg:
		if len(msg.messages) > 0 {
			m.messages = append(m.messages, msg.messages...)
		}
		m.lastPolled = time.Now()
		m.loading = false
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.quitting = true
			return m, nil
		case "enter":
			body := m.input.Value()
			if body == "" {
				return m, nil
			}

			limit := m.cfg.MsgLimit()
			if m.msgCount >= limit {
				m.limitHit = true
				return m, nil
			}

			m.input.SetValue("")
			m.msgCount++
			saveConnectionCount(m.signal.ID, m.msgCount)

			return m, func() tea.Msg {
				err := m.client.PostComment(int64(m.signal.ID), body)
				if err != nil {
					return errorMsg{err: err}
				}
				msgs, _ := m.client.PollComments(int64(m.signal.ID), m.lastPolled)
				return chatMessagesMsg{messages: msgs, posted: true}
			}
		}
	case chatTickMsg:
		interval := time.Duration(m.cfg.PollInterval()) * time.Second
		return m, tea.Batch(m.loadComments(), tickChat(interval))
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func connectionCountPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hi", "connections.json")
}

func loadConnectionCount(signalID int64) int {
	data, err := os.ReadFile(connectionCountPath())
	if err != nil {
		return 0
	}
	var counts map[string]int
	if err := json.Unmarshal(data, &counts); err != nil {
		return 0
	}
	return counts[fmt.Sprintf("%d", signalID)]
}

func saveConnectionCount(signalID int64, count int) {
	path := connectionCountPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := os.ReadFile(path)
	var counts map[string]int
	_ = json.Unmarshal(data, &counts)
	if counts == nil {
		counts = make(map[string]int)
	}
	counts[fmt.Sprintf("%d", signalID)] = count
	out, _ := json.Marshal(counts)
	_ = os.WriteFile(path, out, 0644)
}

func (m *ChatModel) View() string {
	var b strings.Builder

	// ── Header ─────────────────────────────────────────────────────────────
	header := CardHeaderStyle.Render(fmt.Sprintf("💬 %s", m.signal.Title))
	b.WriteString(header + "\n\n")

	// ── Messages area ──────────────────────────────────────────────────────
	if m.loading && len(m.messages) == 0 {
		b.WriteString("  Loading messages...")
	} else if len(m.messages) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(Muted).Render("  No messages yet. Say hi!") + "\n")
	} else {
		username := ""
		if m.cfg != nil {
			username = m.cfg.GitHubUsername
		}
		for _, msg := range m.messages {
			isOwn := strings.EqualFold(msg.User, username)

			// User label
			userLabel := lipgloss.NewStyle().Foreground(Primary).Bold(true).Render(msg.User)
			if isOwn {
				userLabel = lipgloss.NewStyle().Foreground(Secondary).Bold(true).Render(msg.User)
			}

			// Message body
			bubbleStyle := userBubbleStyle
			if !isOwn {
				bubbleStyle = otherBubbleStyle
			}
			b.WriteString(fmt.Sprintf("  %s\n", userLabel))
			b.WriteString(fmt.Sprintf("    %s\n\n", bubbleStyle.Render(msg.Body)))
		}
	}

	// ── Footer with input bar ──────────────────────────────────────────────
	limit := m.cfg.MsgLimit()
	footer := fmt.Sprintf("\n%s\n", InputStyle.Render(m.input.View()))
	footer += CaptionStyle.Render(fmt.Sprintf("  [%d/%d messages  •  tier: %s]", m.msgCount, limit, m.cfg.GetTier()))

	if m.limitHit {
		footer += "\n" + ErrorStyle.Render(fmt.Sprintf("  ⚠ Message limit reached (tier: %s · %d msgs max)", m.cfg.GetTier(), limit))
		footer += "\n" + CaptionStyle.Render("  Upgrade to Pro for 1,000 msgs: hi upgrade")
	} else {
		footer += "\n" + CaptionStyle.Render(fmt.Sprintf("  %s back  %s send", RenderKeyHint("esc"), RenderKeyHint("enter")))
	}

	return CardStyle.Width(m.width).Render(lipgloss.JoinVertical(lipgloss.Left, b.String(), footer))
}

type chatMessagesMsg struct {
	messages []github.ChatMessage
	posted   bool
}
