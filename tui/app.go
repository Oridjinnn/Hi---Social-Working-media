package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/supabase"
)

func sanitizeErr(err error) error {
	return config.SanitizedError(err)
}

// ── Tabs ─────────────────────────────────────────────────────────────────────

type Tab int

const (
	TabHome Tab = iota
	TabFeed
	TabGroup
	TabMarket
)

var tabLabels = []string{"Home", "Feeds", "Grouphouse", "Market Intel"}
var tabBadgeCounts = []int{0, 0, 0, 0}

// ── AppModel ──────────────────────────────────────────────────────────────────

type AppModel struct {
	activeTab   Tab
	profile     ProfileModel
	feed        FeedModel
	group       GroupHouseModel
	market      MarketModel
	lastUpdated map[Tab]time.Time
	width       int
	height      int
	cfg         *config.Config
	lastError   error
}

// sidebarWidth is the condensed sidebar width (down from 20)
const sidebarWidth = 12

// contentWidth returns the usable width for tab content
func (m AppModel) contentWidth() int {
	cw := m.width - sidebarWidth - 2 // 1 sidebar border + 1 gap
	if cw < 30 {
		cw = 30
	}
	return cw
}

func NewAppModel(ghClient *github.Client, supaClient *supabase.Client, cfg *config.Config) AppModel {
	return AppModel{
		activeTab: TabHome,
		profile:   NewProfileModel(nil, nil, true, cfg.GitHubUsername, cfg),
		feed:      NewFeedModel(ghClient, supaClient, cfg),
		group:     NewGroupHouseModel(cfg),
		market:    NewMarketModel(),
		lastUpdated: map[Tab]time.Time{
			TabHome:   time.Now(),
			TabFeed:   time.Now(),
			TabGroup:  time.Now(),
			TabMarket: time.Now(),
		},
		cfg: cfg,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.profile.Init(),
		m.feed.Init(),
		m.group.Init(),
		m.market.Init(),
		fetchMarketCmd(),
	)
}

func (m AppModel) Update(msg tea.Msg) (resModel tea.Model, resCmd tea.Cmd) {
	var cmds []tea.Cmd

	// Guardrail: System-wide panic recovery.
	defer func() {
		if r := recover(); r != nil {
			m.lastError = sanitizeErr(fmt.Errorf("HI recovered from a critical failure: %v", r))
			resModel = m
			resCmd = nil
		}
	}()

	switch msg := msg.(type) {
	case appSyncMsg:
		m.applyAppSync(msg)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cw := m.contentWidth()

		// Pass content width to sub-models
		adjustedMsg := tea.WindowSizeMsg{Width: cw, Height: m.height}

		profileUpdated, pCmd := m.profile.Update(adjustedMsg)
		m.profile = profileUpdated.(ProfileModel)
		cmds = append(cmds, pCmd)

		feedUpdated, fCmd := m.feed.Update(adjustedMsg)
		if fm, ok := feedUpdated.(FeedModel); ok {
			m.feed = fm
		}
		cmds = append(cmds, fCmd)

		groupUpdated, gCmd := m.group.Update(adjustedMsg)
		m.group = groupUpdated
		cmds = append(cmds, gCmd)

		marketUpdated, mCmd := m.market.Update(adjustedMsg)
		m.market = marketUpdated
		cmds = append(cmds, mCmd)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "1":
			m.activeTab = TabHome
			return m, nil
		case "2":
			m.activeTab = TabFeed
			return m, nil
		case "3":
			m.activeTab = TabGroup
			return m, nil
		case "4":
			m.activeTab = TabMarket
			return m, nil
		case "tab":
			if m.lastError != nil {
				m.lastError = nil
				return m, m.Init()
			}
			m.activeTab = (m.activeTab + 1) % Tab(len(tabLabels))
			return m, nil
		}
	}

	// Sync notification badge count from feed notifications
	tabBadgeCounts[TabFeed] = len(m.feed.notifications)

	// Route msgs to owning model only
	switch msg.(type) {
	case signalsLoadedMsg, trendLoadedMsg, errorMsg, toastMsg, toastDismissMsg, chatMessagesMsg, chatTickMsg, realtimeBridgeEventMsg, realtimeBridgeStatusMsg:
		fUpdated, fCmd := m.feed.Update(msg)
		if fm, ok := fUpdated.(FeedModel); ok {
			m.feed = fm
		}
		if _, ok := msg.(signalsLoadedMsg); ok {
			m.touchTab(TabFeed, time.Now())
			m.profile.signals = m.feed.signals
			m.touchTab(TabHome, time.Now())
		}
		if _, ok := msg.(chatMessagesMsg); ok {
			m.touchTab(TabFeed, time.Now())
		}
		if _, ok := msg.(realtimeBridgeStatusMsg); ok {
			m.touchTab(TabFeed, time.Now())
		}
		cmds = append(cmds, fCmd)
	case marketLoadedMsg, marketErrMsg:
		marketUpdated, mCmd := m.market.Update(msg)
		m.market = marketUpdated
		m.touchTab(TabMarket, time.Now())
		cmds = append(cmds, mCmd)
	case ghUpdateMsg:
		groupUpdated, gCmd := m.group.Update(msg)
		m.group = groupUpdated
		m.touchTab(TabGroup, time.Now())
		cmds = append(cmds, gCmd)
	default:
		switch m.activeTab {
		case TabHome:
			profileUpdated, c := m.profile.Update(msg)
			m.profile = profileUpdated.(ProfileModel)
			m.touchTab(TabHome, time.Now())
			cmds = append(cmds, c)
		case TabFeed:
			feedUpdated, c := m.feed.Update(msg)
			if fm, ok := feedUpdated.(FeedModel); ok {
				m.feed = fm
			}
			m.touchTab(TabFeed, time.Now())
			cmds = append(cmds, c)
		case TabGroup:
			groupUpdated, c := m.group.Update(msg)
			m.group = groupUpdated
			m.touchTab(TabGroup, time.Now())
			cmds = append(cmds, c)
		case TabMarket:
			marketUpdated, c := m.market.Update(msg)
			m.market = marketUpdated
			m.touchTab(TabMarket, time.Now())
			cmds = append(cmds, c)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	// Safety Net: If a critical error occurred, show the Recovery Screen
	if m.lastError != nil {
		return m.renderRecoveryScreen()
	}

	var content string
	switch m.activeTab {
	case TabHome:
		content = m.profile.View()
	case TabFeed:
		content = m.feed.View()
	case TabGroup:
		content = m.group.View()
	case TabMarket:
		content = m.market.View()
	}

	// Build the full layout: sidebar + content area with tabs
	sidebar := m.renderSidebar()
	tabBar := m.renderTabBar()
	fullContent := lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, fullContent)
}

func (m AppModel) renderRecoveryScreen() string {
	title := lipgloss.NewStyle().Foreground(Danger).Bold(true).Render("CRITICAL RECOVERY MODE")
	errText := lipgloss.NewStyle().Foreground(Warning).Render(config.RedactSecrets(m.lastError.Error()))
	hint := lipgloss.NewStyle().Foreground(Muted).Render("HI has detected an internal inconsistency and protected your session.\nPress 'tab' to attempt a self-fix/reload.")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Danger).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", errText, "", hint))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m AppModel) renderSidebar() string {
	var items []string

	// Condensed logo — just "HI" in small caps
	logo := lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Margin(0, 0, 1, 0).
		Width(sidebarWidth - 2).
		Align(lipgloss.Center).
		Render("HI")
	items = append(items, logo)

	for i, label := range tabLabels {
		badgeCount := tabBadgeCounts[Tab(i)]
		labelText := label
		if badgeCount > 0 {
			badge := lipgloss.NewStyle().
				Foreground(Danger).
				Bold(true).
				Render(fmt.Sprintf(" %d", badgeCount))
			labelText += badge
		}

		// Use compact icons instead of full emoji labels
		icon := ""
		switch Tab(i) {
		case TabHome:
			icon = "●"
		case TabFeed:
			icon = "◆"
		case TabGroup:
			icon = "▣"
		case TabMarket:
			icon = "◈"
		}
		itemText := fmt.Sprintf("%s %s", icon, labelText)

		if Tab(i) == m.activeTab {
			items = append(items, SidebarItemActive.Copy().Width(sidebarWidth-2).Render(itemText))
		} else {
			items = append(items, SidebarItemInactive.Copy().Width(sidebarWidth-2).Render(itemText))
		}
	}

	// Version / build info + realtime status
	versionStr := lipgloss.NewStyle().
		Foreground(Muted).
		Width(sidebarWidth - 2).
		Align(lipgloss.Center).
		Render("v0.1.0")
	rtStateColor := Muted
	rtLabel := "RT --"
	switch m.feed.realtimeState {
	case RealtimeStateConnected:
		rtStateColor = Success
		rtLabel = "RT ON"
	case RealtimeStateConnecting:
		rtStateColor = Accent
		rtLabel = "RT CONN"
	case RealtimeStateReconnecting:
		rtStateColor = Warning
		rtLabel = "RT RETRY"
	case RealtimeStateDegraded:
		rtStateColor = Warning
		rtLabel = "RT DEG"
	case RealtimeStateOffline:
		rtStateColor = Danger
		rtLabel = "RT OFF"
	}
	rtStr := lipgloss.NewStyle().
		Foreground(rtStateColor).
		Width(sidebarWidth - 2).
		Align(lipgloss.Center).
		Render(rtLabel)
	syncLabel := "SYNC --"
	if !m.feed.realtimeLastSync.IsZero() {
		syncLabel = "SYNC " + humanizeAge(time.Since(m.feed.realtimeLastSync))
	}
	syncStr := lipgloss.NewStyle().
		Foreground(MutedLight).
		Width(sidebarWidth - 2).
		Align(lipgloss.Center).
		Render(syncLabel)
	items = append(items, "", versionStr, rtStr, syncStr)

	sideContent := strings.Join(items, "\n")
	return SidebarStyle.Copy().Width(sidebarWidth).Height(m.height).Render(sideContent)
}

// renderTabBar shows a horizontal tab bar at the top of the content area
func (m AppModel) renderTabBar() string {
	var tabStrs []string
	for i, label := range tabLabels {
		badgeCount := tabBadgeCounts[Tab(i)]
		display := strings.ToUpper(label)
		if badgeCount > 0 {
			badge := lipgloss.NewStyle().
				Foreground(Danger).
				Bold(true).
				Render(fmt.Sprintf(" %d", badgeCount))
			display += badge
		}

		if Tab(i) == m.activeTab {
			tabStrs = append(tabStrs, TabActiveStyle.Render(display))
		} else {
			tabStrs = append(tabStrs, TabInactiveStyle.Render(display))
		}
	}
	tabLine := strings.Join(tabStrs, " ")
	contextHint := m.renderContextHint()
	return lipgloss.NewStyle().Padding(1, 1, 0, 0).Render(tabLine + "\n" + contextHint)
}

func (m AppModel) renderContextHint() string {
	baseHint := "1-4 switch  tab cycle  ctrl+c quit"
	freshness := ""
	if t, ok := m.lastUpdated[m.activeTab]; ok && !t.IsZero() {
		freshness = "  updated " + humanizeAge(time.Since(t)) + " ago"
	}
	switch m.activeTab {
	case TabMarket:
		return HelpStyle.Render("  " + baseHint + "  ↑/↓ nav  tab focus area  r refresh  e export  p pull model" + freshness)
	case TabFeed:
		return HelpStyle.Render("  " + baseHint + "  / search  f filter  n new signal  c connect" + freshness)
	case TabGroup:
		return HelpStyle.Render("  " + baseHint + "  grouphouse controls in view" + freshness)
	case TabHome:
		fallthrough
	default:
		return HelpStyle.Render("  " + baseHint + freshness)
	}
}

func statBox(label, value string) string {
	val := lipgloss.NewStyle().Foreground(Primary).Bold(true).Render(value)
	lbl := lipgloss.NewStyle().Foreground(Muted).Render(label)
	return lipgloss.NewStyle().
		Padding(0, 2).
		MarginRight(2).
		Render(val + " " + lbl)
}

func humanizeAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func (m *AppModel) touchTab(tab Tab, at time.Time) {
	if m.lastUpdated == nil {
		m.lastUpdated = make(map[Tab]time.Time)
	}
	if at.IsZero() {
		at = time.Now()
	}
	m.lastUpdated[tab] = at
}

func (m *AppModel) applyAppSync(msg appSyncMsg) {
	at := msg.at
	if at.IsZero() {
		at = time.Now()
	}
	switch msg.source {
	case "feed":
		m.touchTab(TabFeed, at)
		m.profile.signals = m.feed.signals
		m.touchTab(TabHome, at)
	}
}
