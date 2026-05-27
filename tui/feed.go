package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/notify"
	"github.com/Oridjinnn/hi/supabase"
	"github.com/Oridjinnn/hi/utils"
)

// ── Delegate (custom item rendering) ──────────────────────────────────────────

type signalDelegate struct{}

func (d signalDelegate) Height() int                               { return 2 }
func (d signalDelegate) Spacing() int                              { return 0 }
func (d signalDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d signalDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	sig, ok := listItem.(signalItem)
	if !ok {
		return
	}
	s := sig.signal

	isSelected := index == m.Index()

	// ── Title row ──────────────────────────────────────────────────────────
	title := s.Title

	titleStyle := lipgloss.NewStyle().Foreground(Foreground).Bold(true)
	titleStr := titleStyle.Render(utils.Truncate(title, 50))

	// Difficulty badge
	difficultyColors := map[string]lipgloss.Color{
		"beginner":     Success,
		"intermediate": Warning,
		"advanced":     Danger,
	}
	difColor := difficultyColors[string(s.Difficulty)]
	if difColor == "" {
		difColor = Muted
	}
	difBadge := lipgloss.NewStyle().
		Foreground(difColor).
		Padding(0, 1).
		Render("[" + string(s.Difficulty) + "]")

	// Bookmark indicator
	bookmarkStar := ""
	if sig.bookmarked {
		bookmarkStar = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Render(" ★")
	}

	discoverTag := ""
	if s.IsGhost {
		discoverTag = lipgloss.NewStyle().Foreground(Accent).Bold(true).Render(" [discover]")
	}

	trustBadge := renderTrustBadge(sig.rank)

	titleRow := fmt.Sprintf("%s%s %s%s%s", titleStr, discoverTag, difBadge, trustBadge, bookmarkStar)

	// ── Description row ─────────────────────────────────────────────────────
	var descParts []string

	// Status as colored dot + label
	statusColor := Success
	switch string(s.Status) {
	case "filled":
		statusColor = Muted
	case "expired":
		statusColor = Danger
	}
	statusDot := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render("●")
	statusLbl := lipgloss.NewStyle().Foreground(statusColor).Render(string(s.Status))
	descParts = append(descParts, fmt.Sprintf("%s %s", statusDot, statusLbl))

	// Signal type badge
	typeBadge := SignalTypeStyle.Render(string(s.Type))
	descParts = append(descParts, typeBadge)

	// Author
	authorStr := lipgloss.NewStyle().Foreground(MutedLight).Render("@" + s.Author.GitHubUsername)
	descParts = append(descParts, authorStr)

	if sig.rank.Hint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(Muted)
		if sig.rank.Tier == models.TrustHigh {
			hintStyle = lipgloss.NewStyle().Foreground(Success)
		} else if sig.rank.Tier == models.TrustSpam || sig.rank.Tier == models.TrustLow {
			hintStyle = lipgloss.NewStyle().Foreground(Warning)
		}
		descParts = append(descParts, hintStyle.Render(sig.rank.Hint))
	}

	// Time ago (right-aligned)
	timeStr := lipgloss.NewStyle().Foreground(Muted).Render(utils.TimeAgo(s.CreatedAt))

	descLeft := strings.Join(descParts, "  ")
	descRow := fmt.Sprintf("%s%s", descLeft, lipgloss.NewStyle().Width(20).Align(lipgloss.Right).Render(timeStr))

	// ── Stack tags row ─────────────────────────────────────────────────────
	stackTags := ""
	if len(s.Stack) > 0 {
		tags := make([]string, len(s.Stack))
		for i, st := range s.Stack {
			tags[i] = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A3E635")).
				Render(st)
		}
		stackTags = strings.Join(tags, " · ")
	}

	// Combine description + stack tags into one row
	combinedDesc := descRow
	if stackTags != "" {
		combinedDesc += "  " + lipgloss.NewStyle().Foreground(Muted).Render(stackTags)
	}

	// ── Assemble (2 rows) ─────────────────────────────────────────────────
	var rendered string
	if isSelected {
		sel := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(Primary).
			Background(SurfaceAlt).
			Padding(0, 1).
			Width(m.Width() - 4)
		rendered = sel.Render(
			titleRow + "\n" + combinedDesc,
		)
	} else {
		style := lipgloss.NewStyle().
			Padding(0, 2).
			Width(m.Width() - 4)
		rendered = style.Render(
			titleRow + "\n" + combinedDesc,
		)
	}

	fmt.Fprint(w, rendered)
}

func renderTrustBadge(rank models.SignalRank) string {
	if rank.Tier == models.TrustMedium {
		return ""
	}
	label := ""
	color := Muted
	switch rank.Tier {
	case models.TrustHigh:
		label = "trusted"
		color = Success
	case models.TrustLow:
		label = "low"
		color = Warning
	case models.TrustSpam:
		label = "noise"
		color = Danger
	}
	if label == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(color).Render(" [" + label + "]")
}

func (m FeedModel) buildSignalItems(signals []models.Signal) []list.Item {
	items := make([]list.Item, len(signals))
	for i, s := range signals {
		bookmarked := false
		if m.bookmarks != nil {
			bookmarked = m.bookmarks[s.ID]
		}
		items[i] = signalItem{
			signal:     s,
			bookmarked: bookmarked,
			rank:       models.ScoreSignal(s),
		}
	}
	return items
}

type FeedModel struct {
	list             list.Model
	signals          []models.Signal
	loading          bool
	trend            *TrendModel
	trendLoaded      bool
	toast            *ToastModel
	client           *github.Client
	supaClient       *supabase.Client
	cfg              *config.Config
	spinner          spinner.Model
	err              error
	width            int
	height           int
	retryCount       int
	realtimeBridge   *RealtimeBridge
	realtimeEvents   <-chan models.ConnectionEvent
	realtimeStatuses <-chan realtimeBridgeStatusMsg
	realtimeState    RealtimeConnState
	realtimeLastSync time.Time
	realtimeErr      string
	// Wizard mode
	wizard     *WizardModel
	wizardDone bool
	// Detail mode
	detail *DetailModel
	// Chat mode
	chat *ChatModel
	// Filter mode
	filterActive bool
	filterLabel  string
	// Active filter options
	filterOptions []string
	filterCursor  int
	// Search mode
	searchActive bool
	searchInput  textinput.Model
	searchQuery  string
	// Notification center
	notifications     []models.ConnectionEvent
	showNotifications bool
	notifCursor       int
	// Bookmarks
	bookmarks      map[int64]bool
	showBookmarked bool
}

type signalItem struct {
	signal     models.Signal
	bookmarked bool
	rank       models.SignalRank
}

func (i signalItem) Title() string {
	return i.signal.Title
}

func (i signalItem) Description() string {
	return ""
}

func (i signalItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", i.signal.Title, i.signal.Type, strings.Join(i.signal.Stack, " "), i.signal.Project)
}

func NewFeedModel(client *github.Client, supaClient *supabase.Client, cfg *config.Config) FeedModel {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(PrimaryLight)

	items := []list.Item{}
	delegate := signalDelegate{}

	l := list.New(items, delegate, 80, 30)
	l.Title = "HI — Signal Feed"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = TitleStyle
	l.Styles.PaginationStyle = HelpStyle
	l.Styles.HelpStyle = HelpStyle
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	l.SetShowHelp(false)
	l.SetShowFilter(false)

	ti := textinput.New()
	ti.Placeholder = "Search signals by title, stack, or author..."
	ti.PromptStyle = lipgloss.NewStyle().Foreground(PrimaryLight)
	ti.TextStyle = lipgloss.NewStyle().Foreground(Foreground)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(MutedLight)
	ti.Width = 60

	return FeedModel{
		list:          l,
		signals:       nil,
		loading:       true,
		trend:         NewTrendModel(client, cfg),
		client:        client,
		supaClient:    supaClient,
		cfg:           cfg,
		spinner:       sp,
		filterOptions: []string{"all", "difficulty:beginner", "difficulty:intermediate", "difficulty:advanced", "type:contributor", "type:beginner", "type:showcase"},
		searchInput:   ti,
	}
}

func (m FeedModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)
	cmds = append(cmds, m.loadSignals())

	if m.trend != nil {
		cmds = append(cmds, m.trend.LoadCmd(nil))
	}

	if m.supaClient != nil && m.cfg != nil && m.cfg.GitHubUsername != "" {
		bridge := NewRealtimeBridge(m.supaClient, m.cfg.GitHubUsername)
		m.realtimeBridge = bridge
		m.realtimeEvents, m.realtimeStatuses = bridge.Start()
		cmds = append(cmds, waitRealtimeBridgeEvent(m.realtimeEvents))
		cmds = append(cmds, waitRealtimeBridgeStatus(m.realtimeStatuses))
	}

	return tea.Batch(cmds...)
}

func (m FeedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.wizard != nil && !m.wizardDone {
		return m.updateWizard(msg)
	}

	if m.chat != nil {
		return m.updateChat(msg)
	}

	if connectMsg, ok := msg.(ConnectMsg); ok && connectMsg.Signal != nil {
		m.detail = nil
		c := NewChatModel(connectMsg.Signal, m.client, m.cfg)
		m.chat = c
		return m, m.chat.Init()
	}

	if m.detail != nil {
		return m.updateDetail(msg)
	}

	if m.searchActive {
		return m.updateSearch(msg)
	}

	if m.filterActive {
		return m.updateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-10)
		if m.trend != nil {
			m.trend.width = msg.Width
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if m.showNotifications {
			switch msg.String() {
			case "up", "k":
				if m.notifCursor > 0 {
					m.notifCursor--
				}
				return m, tea.Batch(cmds...)
			case "down", "j":
				if m.notifCursor < len(m.notifications)-1 {
					m.notifCursor++
				}
				return m, tea.Batch(cmds...)
			case "esc", "q", "N":
				m.showNotifications = false
				return m, tea.Batch(cmds...)
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "right", "enter":
			if len(m.signals) > 0 {
				idx := m.list.Index()
				sig := m.signals[idx]
				m.detail = NewDetailModel(&sig)
			}
			return m, tea.Batch(cmds...)

		case "/":
			m.searchActive = true
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			m.searchQuery = ""
			return m, tea.Batch(append(cmds, textinput.Blink)...)

		case "n":
			username := ""
			if m.cfg != nil {
				username = m.cfg.GitHubUsername
			}
			wiz := NewWizard(m.client, m.supaClient, username, m.cfg)
			m.wizard = &wiz
			m.wizardDone = false
			return m, tea.Batch(append(cmds, m.wizard.Init())...)
		case "f":
			m.filterActive = true
			m.filterCursor = 0
			return m, tea.Batch(cmds...)
		case "c":
			if len(m.signals) > 0 {
				idx := m.list.Index()
				sig := m.signals[idx]

				if m.cfg == nil || m.cfg.GitHubToken == "" {
					t := NewToast(models.ConnectionEvent{
						ActorUsername: "system",
						EventType:     "auth_required",
					})
					m.toast = &t
					m.err = fmt.Errorf("authentication required — run: hi auth login")
					return m, tea.Batch(cmds...)
				}
				return m, tea.Batch(append(cmds, m.connectCmd(sig))...)
			}

		case "o":
			if len(m.signals) > 0 {
				idx := m.list.Index()
				signal := m.signals[idx]
				utils.OpenURL(signal.GitHubURL)
			}
		case "r":
			m.loading = true
			cmds = []tea.Cmd{m.loadSignals()}
			if m.trend != nil {
				cmds = append(cmds, m.trend.LoadCmd(nil))
			}
			return m, tea.Batch(cmds...)

		case "N":
			m.showNotifications = !m.showNotifications
			if m.showNotifications {
				m.notifCursor = 0
			}
			return m, nil

		case "s":
			if len(m.signals) > 0 {
				idx := m.list.Index()
				sig := m.signals[idx]
				if m.bookmarks == nil {
					m.bookmarks = make(map[int64]bool)
				}
				m.bookmarks[sig.ID] = !m.bookmarks[sig.ID]
				cmd := m.list.SetItems(m.buildSignalItems(m.signals))
				return m, tea.Batch(append(cmds, cmd)...)
			}

		case "S":
			m.showBookmarked = !m.showBookmarked
			if m.showBookmarked && m.bookmarks != nil {
				var filtered []models.Signal
				for _, s := range m.signals {
					if m.bookmarks[s.ID] {
						filtered = append(filtered, s)
					}
				}
				items := m.buildSignalItems(filtered)
				for i := range items {
					if si, ok := items[i].(signalItem); ok {
						si.bookmarked = true
						items[i] = si
					}
				}
				cmd := m.list.SetItems(items)
				return m, tea.Batch(append(cmds, cmd)...)
			}
			if m.showBookmarked {
				cmd := m.list.SetItems(nil)
				return m, tea.Batch(append(cmds, cmd)...)
			}
			cmd := m.list.SetItems(m.buildSignalItems(m.signals))
			return m, tea.Batch(append(cmds, cmd)...)
		}

	case signalsLoadedMsg:
		m.loading = false
		m.signals = msg.signals
		m.filterLabel = msg.filter
		return m, tea.Batch(append(cmds, m.list.SetItems(m.buildSignalItems(msg.signals)))...)

	case spinner.TickMsg:
		if m.loading {
			var spinnerCmd tea.Cmd
			m.spinner, spinnerCmd = m.spinner.Update(msg)
			return m, tea.Batch(append(cmds, spinnerCmd)...)
		}

	case errorMsg:
		if m.retryCount < 3 {
			m.retryCount++
			return m, tea.Batch(append(cmds, m.loadSignals())...)
		}
		m.loading = false
		m.err = msg.err

	case toastMsg:
		m.toast = msg.toast
		return m, tea.Batch(cmds...)

	case realtimeBridgeEventMsg:
		m.notifications = append([]models.ConnectionEvent{msg.event}, m.notifications...)
		if len(m.notifications) > 50 {
			m.notifications = m.notifications[:50]
		}

		t := NewToast(msg.event)
		m.toast = &t
		_ = notify.FireOS("HI — New Connection", fmt.Sprintf("@%s connected to your signal #%d", msg.event.ActorUsername, msg.event.SignalID), "")

		if m.realtimeEvents != nil {
			return m, tea.Batch(append(cmds, waitRealtimeBridgeEvent(m.realtimeEvents))...)
		}
		return m, tea.Batch(cmds...)

	case realtimeBridgeStatusMsg:
		m.realtimeState = msg.state
		m.realtimeLastSync = msg.lastSync
		if msg.err != nil {
			m.realtimeErr = msg.err.Error()
		} else {
			m.realtimeErr = ""
		}
		if m.realtimeStatuses != nil {
			return m, tea.Batch(append(cmds, waitRealtimeBridgeStatus(m.realtimeStatuses))...)
		}
		return m, tea.Batch(cmds...)

	case toastDismissMsg:
		m.toast = nil

	case ConnectMsg:
		if msg.Signal != nil {
			c := NewChatModel(msg.Signal, m.client, m.cfg)
			m.chat = c
			if m.detail != nil {
				m.detail = nil
			}
			cmds = append(cmds, m.chat.Init(), emitAppSyncCmd("feed", "connect_opened_chat"))
			return m, tea.Batch(cmds...)
		}

	case trendLoadedMsg:
		return m, tea.Batch(cmds...)

	case wizardDoneMsg:
		m.wizard = nil
		m.wizardDone = false
		m.loading = true
		cmds = append(cmds, m.loadSignals(), emitAppSyncCmd("feed", "signal_created"))
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, tea.Batch(append(cmds, cmd)...)
}

func (m FeedModel) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.searchActive = false
			m.searchQuery = ""
			m.searchInput.SetValue("")
			cmd := m.list.SetItems(m.buildSignalItems(m.signals))
			return m, cmd
		case "enter":
			query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
			m.searchQuery = query
			if query == "" {
				m.searchActive = false
				cmd := m.list.SetItems(m.buildSignalItems(m.signals))
				return m, cmd
			}

			var filtered []models.Signal
			for _, s := range m.signals {
				if strings.Contains(strings.ToLower(s.Title), query) ||
					strings.Contains(strings.ToLower(s.Author.GitHubUsername), query) ||
					strings.Contains(strings.ToLower(string(s.Type)), query) ||
					containsAny(strings.ToLower(strings.Join(s.Stack, " ")), query) {
					filtered = append(filtered, s)
				}
			}
			cmd := m.list.SetItems(m.buildSignalItems(filtered))
			return m, cmd
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func containsAny(s, substr string) bool {
	parts := strings.Fields(s)
	for _, p := range parts {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}

func (m FeedModel) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.filterCursor > 0 {
				m.filterCursor--
			}
		case "down":
			if m.filterCursor < len(m.filterOptions)-1 {
				m.filterCursor++
			}
		case "enter":
			selected := m.filterOptions[m.filterCursor]
			m.filterActive = false
			if selected == "all" {
				m.filterLabel = ""
			} else {
				m.filterLabel = selected
			}
			m.loading = true
			return m, m.loadSignals()
		case "esc", "q":
			m.filterActive = false
			return m, nil
		}
	}
	return m, nil
}

func (m FeedModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if connectMsg, ok := msg.(ConnectMsg); ok {
		m.detail = nil
		c := NewChatModel(connectMsg.Signal, m.client, m.cfg)
		m.chat = c
		return m, c.Init()
	}

	if m.detail != nil {
		updated, cmd := m.detail.Update(msg)
		if d, ok := updated.(*DetailModel); ok {
			if d.quitting {
				m.detail = nil
				return m, cmd
			}
			return m, cmd
		}
	}
	return m, nil
}

func (m FeedModel) updateChat(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.chat == nil {
		return m, nil
	}
	updated, cmd := m.chat.Update(msg)
	if c, ok := updated.(*ChatModel); ok {
		if c.quitting {
			m.chat = nil
			return m, cmd
		}
		m.chat = c
		if chatMsg, ok := msg.(chatMessagesMsg); ok && chatMsg.posted {
			return m, tea.Batch(cmd, emitAppSyncCmd("feed", "chat_posted"))
		}
		return m, cmd
	}
	return m, nil
}

func (m FeedModel) updateWizard(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.wizard == nil {
		return m, nil
	}
	updated, cmd := m.wizard.Update(msg)
	if wiz, ok := updated.(*WizardModel); ok {
		m.wizard = wiz
	}
	return m, cmd
}

func (m FeedModel) renderNotificationCenter() string {
	width := 56

	notifCount := len(m.notifications)
	title := fmt.Sprintf("🔔 Notifications (%d)", notifCount)

	if notifCount == 0 {
		content := CardHeaderStyle.Render(title) + "\n\n" +
			HelpStyle.Render("  No notifications yet.\n  When someone connects to your signal,\n  it will appear here.")
		return "\n" + CardStyle.Width(width).Render(content) + "\n"
	}

	var items strings.Builder
	items.WriteString("\n")
	start := m.notifCursor
	end := start + 10
	if end > notifCount {
		end = notifCount
	}
	for i := start; i < end; i++ {
		e := m.notifications[i]
		prefix := "  "
		style := CaptionStyle
		if i == m.notifCursor {
			prefix = "▸ "
			style = HighlightStyle.Copy().Padding(0, 1)
		}
		eventType := "connected"
		if e.EventType == "view" {
			eventType = "viewed"
		}
		label := fmt.Sprintf("@%s %s signal #%d", e.ActorUsername, eventType, e.SignalID)
		items.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(label)))
	}

	help := HelpStyle.Render(fmt.Sprintf("  %s · page %d-%d of %d", RenderKeyHint("↑", "↓", "scroll"), start+1, end, notifCount))

	content := CardHeaderStyle.Render(title) + "\n" +
		items.String() + "\n" +
		help

	return "\n" + CardStyle.Width(width).Render(content) + "\n"
}

func (m FeedModel) View() string {
	if m.wizard != nil {
		return m.wizard.View()
	}

	if m.chat != nil {
		return m.chat.View()
	}

	if m.detail != nil {
		return m.detail.View()
	}

	// Build base feed content first
	baseContent := m.renderBaseFeed()

	// Overlay search, filter, or notifications on top of base feed
	if m.searchActive {
		searchBar := InputStyle.Render("  / " + m.searchInput.View() + "\n")
		searchHint := HelpStyle.Render("  " + RenderKeyHint("enter") + " commit  " + RenderKeyHint("esc") + " cancel")
		return lipgloss.JoinVertical(lipgloss.Left, searchBar, searchHint, "", baseContent)
	}

	if m.filterActive {
		return lipgloss.JoinVertical(lipgloss.Left, m.renderFilterMenu(), "", baseContent)
	}

	if m.showNotifications {
		return lipgloss.JoinVertical(lipgloss.Left, m.renderNotificationCenter(), "", baseContent)
	}

	return baseContent
}

func (m FeedModel) renderBaseFeed() string {
	if m.loading {
		return lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render(m.spinner.View() + " Loading signals...")
	}

	if m.err != nil {
		return lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render(ErrorStyle.Render(fmt.Sprintf("Error: %s", config.RedactSecrets(m.err.Error()))))
	}

	if len(m.signals) == 0 && !m.loading {
		return renderWelcome()
	}

	var feedView strings.Builder

	feedView.WriteString(m.list.View())

	// Condensed help bar with grouped key hints
	help := HelpStyle.Render("  ↑/↓/j/k nav  n new  → detail  c connect  s ★  f filter  / search/N notif/r ref/q quit")
	feedView.WriteString("\n")
	feedView.WriteString(help)

	// Toast overlay - appended at bottom without truncation
	if m.toast != nil && m.toast.visible {
		toastView := m.toast.View()
		feedView.WriteString("\n\n" + toastView + "\n")
	}

	return feedView.String()
}

func (m FeedModel) renderFilterMenu() string {
	width := 52

	var items strings.Builder
	items.WriteString("\n")
	for i, opt := range m.filterOptions {
		prefix := "  "
		style := CaptionStyle
		if i == m.filterCursor {
			prefix = "▸ "
			style = HighlightStyle.Copy().Padding(0, 1)
		}
		items.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(opt)))
	}

	help := HelpStyle.Render("  " + RenderKeyHint("↑", "↓") + " navigate  " + RenderKeyHint("enter") + " apply  " + RenderKeyHint("esc") + " cancel")

	content := CardHeaderStyle.Render("Filter Signals") + "\n" +
		items.String() + "\n" +
		help

	return "\n" + CardStyle.Width(width).Render(content) + "\n"
}

func renderWelcome() string {
	width := 52

	subStyle := lipgloss.NewStyle().Foreground(MutedLight)
	stepStyle := lipgloss.NewStyle().Foreground(Foreground)

	content := H1Style.Render("👋 Welcome to HI!") + "\n\n" +
		subStyle.Render("  Your feed is empty — let's fix that!") + "\n\n" +
		stepStyle.Render("  1. Press ") + RenderKeyHint("n") + stepStyle.Render(" to create your first signal") + "\n" +
		HelpStyle.Render("     Describe your project, stack, and what you need") + "\n\n" +
		stepStyle.Render("  2. Other devs will find you") + "\n" +
		HelpStyle.Render("     They'll connect and you'll build together") + "\n\n" +
		RenderDivider(width-2, Muted) + "\n\n" +
		HelpStyle.Render("  "+RenderKeyHint("↑", "↓")+" navigate  "+RenderKeyHint("n")+" new signal  "+RenderKeyHint("→")+" details  "+RenderKeyHint("f")+" filter  "+RenderKeyHint("q")+" quit")

	return "\n" + CardStyle.Width(width).Render(content) + "\n"
}

func (m FeedModel) connectCmd(s models.Signal) tea.Cmd {
	return func() tea.Msg {
		return ConnectMsg{SignalID: s.ID, Signal: &s}
	}
}

func (m FeedModel) loadSignals() tea.Cmd {
	return func() tea.Msg {
		var labels []string
		if m.filterLabel != "" {
			labels = append(labels, m.filterLabel)
		}
		signals, err := m.client.ListSignals(labels, 1)
		if err != nil {
			return errorMsg{err: err}
		}
		signals = models.RankSignals(signals)

		return signalsLoadedMsg{signals: signals, filter: m.filterLabel}
	}
}

type signalsLoadedMsg struct {
	signals []models.Signal
	filter  string
}

type errorMsg struct {
	err error
}

type OpenWizardMsg struct{}

type ConnectMsg struct {
	SignalID int64
	Signal   *models.Signal
}

type toastMsg struct {
	toast *ToastModel
}

type toastDismissMsg struct{}

var _ = models.Signal{}
