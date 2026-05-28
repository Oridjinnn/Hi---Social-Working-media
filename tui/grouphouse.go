package tui

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/grouphouse"
	"github.com/Oridjinnn/hi/history"
	"github.com/Oridjinnn/hi/utils"
)

// ── Group House styles ──────────────────────────────────────────────────────

var (
	ghAgentOnlineStyle = lipgloss.NewStyle().
				Foreground(Success).Bold(true)

	ghAgentIdleStyle = lipgloss.NewStyle().
				Foreground(Muted)

	ghLogTimestampStyle = lipgloss.NewStyle().Foreground(MutedLight)
	ghLogSenderStyle    = lipgloss.NewStyle().Foreground(PrimaryLight).Bold(true)
	ghLogTextStyle      = lipgloss.NewStyle().Foreground(Foreground)
)

// ── Messages ─────────────────────────────────────────────────────────────────

type ghLogEntry struct {
	Timestamp time.Time
	Sender    string
	Text      string
	MsgType   string
}

type ghUpdateMsg struct {
	agents    []grouphouse.AgentInfo
	log       ghLogEntry
	houseName string
	workspace string
}

// ── GroupHouseModel ──────────────────────────────────────────────────────────

type GroupHouseModel struct {
	cfg             *config.Config
	server          *grouphouse.Server
	isRunning       bool
	houseName       string
	workspace       string
	port            int
	agents          []grouphouse.AgentInfo
	log             []ghLogEntry
	inputActive     bool
	inputText       string
	recommendations []string
	pulling         bool
	availableModels []string
	selectedModel   string
	showModelMenu   bool
	logOffset       int
	history         *history.History
	historyMode     bool
	err             error
	width           int
}

type ghPullDoneMsg struct {
	modelName string
	err       error
}

func NewGroupHouseModel(cfg *config.Config) GroupHouseModel {
	historyData, _ := history.Load()
	return GroupHouseModel{
		cfg:             cfg,
		port:            9753,
		agents:          []grouphouse.AgentInfo{},
		log:             []ghLogEntry{},
		recommendations: getHardwareRecommendations(),
		history:         historyData,
	}
}

func detectTotalMemoryMB() uint64 {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 8000
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				var mem uint64
				_, _ = fmt.Sscanf(line, "MemTotal: %d kB", &mem)
				return mem / 1024
			}
		}
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 8000
		}
		var mem uint64
		_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &mem)
		return mem / 1024 / 1024
	}
	return 8000
}

func getHardwareRecommendations() []string {
	ramMB := detectTotalMemoryMB()
	_, err := exec.LookPath("ollama")

	var models []string
	if ramMB < 6000 {
		models = []string{
			"Phi-3-Mini (3.8B) — Fast & tiny",
			"TinyLlama-1.1B — Ultra lightweight",
		}
	} else if ramMB < 12000 {
		models = []string{
			"Qwen-2.5-3B — Modern coding model",
			"Phi-3-Mini (3.8B) — Fast on 8GB RAM",
			"Llama-3-8B (Q4_0) — Good all-rounder",
		}
	} else {
		models = []string{
			"Mistral-7B-v0.3 — High performance",
			"Gemma-2-9B — State-of-the-art logic",
		}
	}

	if err != nil {
		for i, m := range models {
			models[i] = m + " (Install Ollama to run)"
		}
	}

	return models
}

type ghModelsLoadedMsg []string

func (m GroupHouseModel) fetchLocalModels() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("ollama", "list").Output()
		if err != nil {
			return ghModelsLoadedMsg{}
		}
		lines := strings.Split(string(out), "\n")
		var models []string
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) > 0 {
				name := strings.TrimSuffix(fields[0], ":latest")
				models = append(models, name)
			}
		}
		return ghModelsLoadedMsg(models)
	}
}

func (m GroupHouseModel) isInstalled(rec string) bool {
	name := strings.ToLower(strings.Split(rec, " ")[0])
	name = strings.ReplaceAll(name, "-", "")
	for _, am := range m.availableModels {
		amNorm := strings.ReplaceAll(strings.ToLower(am), ":", "")
		if strings.Contains(amNorm, name) || strings.Contains(name, amNorm) {
			return true
		}
	}
	return false
}

func (m GroupHouseModel) pullModelCmd(modelName string) tea.Cmd {
	return func() tea.Msg {
		name := strings.ToLower(strings.Split(modelName, " ")[0])
		tag := name
		if strings.Contains(name, "phi") {
			tag = "phi3:mini"
		}
		if strings.Contains(name, "qwen") {
			tag = "qwen2.5:3b"
		}
		if strings.Contains(name, "llama") {
			tag = "llama3"
		}
		err := exec.Command("ollama", "pull", tag).Run()
		return ghPullDoneMsg{modelName: tag, err: err}
	}
}

func (m GroupHouseModel) spawnLocalAgentCmd(modelName string) tea.Cmd {
	return func() tea.Msg {
		host := fmt.Sprintf("ws://localhost:%d", m.port)
		agentID := fmt.Sprintf("local-%s-%d", modelName, time.Now().Unix())
		client := grouphouse.NewClient(host, modelName, grouphouse.KindAgent, agentID)

		client.OnMessage = func(msg grouphouse.Message) {
			if msg.Type == grouphouse.MsgBroadcast && msg.Sender != modelName {
				prompt := fmt.Sprintf("Context: You are an AI agent in a shared workspace. User says: %s", msg.Payload.(map[string]interface{})["text"])
				out, err := exec.Command("ollama", "run", modelName, prompt).Output()
				if err == nil {
					_ = client.Broadcast(strings.TrimSpace(string(out)))
				}
			}
		}

		if err := client.Connect(); err != nil {
			return ghUpdateMsg{
				log: ghLogEntry{
					Timestamp: time.Now(),
					Sender:    "system",
					Text:      fmt.Sprintf("Failed to connect local agent: %v", err),
					MsgType:   "error",
				},
			}
		}

		return ghUpdateMsg{
			log: ghLogEntry{
				Timestamp: time.Now(),
				Sender:    "system",
				Text:      fmt.Sprintf("Local agent '%s' joined the house", modelName),
				MsgType:   "join",
			},
		}
	}
}

func (m GroupHouseModel) Init() tea.Cmd {
	return m.fetchLocalModels()
}

type ghServerStartedMsg struct {
	server    *grouphouse.Server
	houseName string
	workspace string
	port      int
	err       error
}

func findFreePort(start int) int {
	for port := start; port < start+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = ln.Close()
			return port
		}
	}
	return start
}

func (m GroupHouseModel) startServer() tea.Cmd {
	return func() tea.Msg {
		name := m.houseName
		if name == "" || name == "-house" {
			if name == "-house" {
				name = "my-house"
			}
		}

		port := findFreePort(m.port)
		if port != m.port {
			m.port = port
		}

		configDir := config.ConfigDir()
		ws, err := grouphouse.NewWorkspace(configDir, name)
		if err != nil {
			return ghServerStartedMsg{err: fmt.Errorf("workspace: %w", err)}
		}

		server := grouphouse.NewServer(name, port, ws)

		go func() {
			if err := server.Start(); err != nil {
				fmt.Printf("Server error: %v\n", err)
			}
		}()

		return ghServerStartedMsg{
			server:    server,
			houseName: name,
			workspace: ws.Path,
			port:      port,
		}
	}
}

func (m GroupHouseModel) stopServer() tea.Cmd {
	return func() tea.Msg {
		if m.server != nil {
			m.server.Stop()
		}
		return nil
	}
}

func (m GroupHouseModel) Update(msg tea.Msg) (GroupHouseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		// Pass width down to recommendations and other components
		return m, nil

	case ghServerStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.server = msg.server
		m.houseName = msg.houseName
		m.workspace = msg.workspace
		if msg.port > 0 {
			m.port = msg.port
		}
		m.isRunning = true
		logEntry := ghLogEntry{
			Timestamp: time.Now(),
			Sender:    "system",
			Text:      fmt.Sprintf("House started on port %d", m.port),
			MsgType:   "join",
		}
		m.log = append(m.log, logEntry)
		if m.history != nil {
			m.history.LogGroupEvent(logEntry.Text)
		}
		return m, m.fetchLocalModels()

	case ghModelsLoadedMsg:
		m.availableModels = msg
		return m, nil

	case ghPullDoneMsg:
		m.pulling = false
		if msg.err != nil {
			m.err = fmt.Errorf("pull failed: %w", msg.err)
		}
		return m, m.fetchLocalModels()

	case ghUpdateMsg:
		if msg.houseName != "" {
			m.houseName = msg.houseName
		}
		if msg.workspace != "" {
			m.workspace = msg.workspace
		}
		if msg.agents != nil {
			m.agents = msg.agents
		}
		if msg.log.Timestamp != (time.Time{}) {
			m.log = append(m.log, msg.log)
			if len(m.log) > 100 {
				m.log = m.log[len(m.log)-100:]
			}
			if m.history != nil {
				m.history.LogGroupEvent(msg.log.Text)
			}
		}
		return m, nil

	case tea.MouseMsg:
		if !m.isRunning || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if msg.Y > 20 {
			if m.showModelMenu {
				m.showModelMenu = false
			} else if msg.X < 15 {
				m.showModelMenu = true
			}
		}

	case tea.KeyMsg:
		if !m.isRunning {
			switch msg.String() {
			case "p":
				if !m.pulling && len(m.recommendations) > 0 {
					for _, rec := range m.recommendations {
						if !m.isInstalled(rec) {
							m.pulling = true
							return m, m.pullModelCmd(rec)
						}
					}
				}
			case "enter", " ":
				m.houseName = m.cfg.GitHubUsername + "-house"
				return m, m.startServer()
			}
			return m, nil
		}

		if m.showModelMenu {
			switch msg.String() {
			case "enter":
				if len(m.availableModels) > 0 {
					m.selectedModel = m.availableModels[0]
					m.showModelMenu = false
					return m, m.spawnLocalAgentCmd(m.selectedModel)
				}
			case "esc", "q":
				m.showModelMenu = false
				return m, nil
			}
			return m, nil
		}

		if m.inputActive {
			switch msg.String() {
			case "enter":
				text := strings.TrimSpace(m.inputText)
				if text != "" {
					entry := ghLogEntry{
						Timestamp: time.Now(),
						Sender:    m.cfg.GitHubUsername,
						Text:      text,
						MsgType:   "broadcast",
					}
					m.log = append(m.log, entry)
					if m.history != nil {
						m.history.LogGroupEvent(entry.Text)
					}
				}
				m.inputText = ""
				m.inputActive = false
				return m, nil
			case "esc":
				m.inputText = ""
				m.inputActive = false
				return m, nil
			case "backspace":
				if len(m.inputText) > 0 {
					m.inputText = m.inputText[:len(m.inputText)-1]
				}
				return m, nil
			default:
				if s := msg.String(); len(s) == 1 {
					m.inputText += s
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "h":
			m.historyMode = !m.historyMode
			return m, nil
		case "s":
			if !m.isRunning {
				return m, m.startServer()
			}
		case "q":
			if m.isRunning {
				return m, m.stopServer()
			}
		case "i", "enter":
			m.inputActive = true
			m.inputText = ""
			return m, nil
		case "r":
			if m.server != nil {
				return m, m.fetchLocalModels()
			}
		case "up", "k":
			if m.logOffset < len(m.log)-1 {
				m.logOffset++
			}
		case "down", "j":
			if m.logOffset > 0 {
				m.logOffset--
			}
		}
	}

	return m, nil
}

func (m GroupHouseModel) View() string {
	var b strings.Builder

	if !m.isRunning {
		return m.renderStartPrompt()
	}

	if m.historyMode {
		return m.renderGroupHistory()
	}

	// Header with shared styles
	statusColor := Success
	statusText := "● RUNNING"
	if !m.isRunning {
		statusColor = Muted
		statusText = "○ STOPPED"
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		CaptionStyle.Render("GROUPS / "),
		H1Style.Render(m.houseName),
		" ",
		lipgloss.NewStyle().Foreground(statusColor).Render(statusText),
		"  ",
		CaptionStyle.Render(fmt.Sprintf("listening on :%d", m.port)),
	)
	b.WriteString("\n " + header + "\n")

	// Agent panel + Log panel side by side
	agentPanel := m.renderAgentPanel()
	logPanel := m.renderLogPanel()
	panels := lipgloss.JoinHorizontal(lipgloss.Top, agentPanel, "  ", logPanel)
	b.WriteString(panels + "\n")

	if m.showModelMenu {
		b.WriteString(m.renderModelMenu())
	}

	inputBar := m.renderInputBar()
	b.WriteString("\n" + inputBar)

	help := HelpStyle.Render(fmt.Sprintf("  %s send message  %s stop house  %s scroll log  %s history  %s close input",
		RenderKeyHint("i"),
		RenderKeyHint("q"),
		RenderKeyHint("↑", "↓"),
		RenderKeyHint("h"),
		RenderKeyHint("esc"),
	))
	b.WriteString("\n" + help)

	return b.String()
}

func (m GroupHouseModel) renderStartPrompt() string {
	w := cardWidth(&m, 52)
	if w < 30 {
		w = 52
	}

	recView := ""
	if len(m.recommendations) > 0 {
		recView = "\n" + H2Style.Render("Recommended for your hardware:") + "\n"
		for i, rec := range m.recommendations {
			statusText := "[missing]"
			if m.isInstalled(rec) {
				statusText = "[installed]"
			}
			recView += CaptionStyle.Render(fmt.Sprintf(" • %s %s", rec, statusText)) + "\n"
			if i == 0 && !m.isInstalled(rec) && !m.pulling {
				recView += HelpStyle.Render("   (Press 'p' to pull)") + "\n"
			}
		}
		if m.pulling {
			recView += "\n" + CaptionStyle.Render(" ⏳ Pulling model... Please wait.") + "\n"
		}
		recView += "\n"
	}

	var errView string
	if m.err != nil {
		errView = ErrorStyle.Render("Error: "+config.RedactSecrets(m.err.Error())) + "\n\n"
	}

	box := CardStyle.Width(w).Render(
		H1Style.Render("🏠 Group House") + "\n\n" +
			BodyStyle.Render("Shared workspace for you, your team,\nand AI agents.") + "\n\n" +
			CaptionStyle.Render("Start building with your friend, ai agent\nand even your local model!") + "\n\n" +
			recView +
			errView +
			SuccessStyle.Render("Press Enter to start your house") + "\n\n" +
			CaptionStyle.Render("Or run: hi grouphouse start --port 9753") + "\n" +
			CaptionStyle.Render("Then: hi grouphouse join --host ws://localhost:9753 --name <agent>"),
	)

	return "\n  " + box + "\n\n" + HelpStyle.Render("  tab: switch tabs · 1/2/3: jump")
}

func (m GroupHouseModel) renderGroupHistory() string {
	header := CardHeaderStyle.Render("⏪ Rewind — Group House history") + "\n\n"
	if m.history == nil {
		return CardStyle.Width(m.width).Render(header + CaptionStyle.Render("  No group history available."))
	}

	var lines []string
	events := m.history.RecentGroupEvents()
	for i := len(events) - 1; i >= 0 && i >= len(events)-15; i-- {
		event := events[i]
		lines = append(lines, fmt.Sprintf("  • [%s] %s", utils.TimeAgo(event.Timestamp), event.Event))
	}
	if len(lines) == 0 {
		lines = append(lines, "  No group history recorded yet.")
	}

	content := header + strings.Join(lines, "\n") + "\n\n" + HelpStyle.Render(fmt.Sprintf("  %s close", RenderKeyHint("h")))
	return CardStyle.Width(m.width).Render(content)
}

func cardWidth(m *GroupHouseModel, hardcoded int) int {
	if m.width == 0 {
		return hardcoded
	}
	return m.width - 4
}

// panelWidths returns agent and log panel widths based on container width
func (m *GroupHouseModel) panelWidths() (agentW, logW int) {
	totalW := m.width - 4
	if totalW <= 0 {
		return 28, 50 // fallback defaults
	}
	agentW = int(float64(totalW) * 0.32) // ~30% for agents
	logW = totalW - agentW - 2           // rest for log (minus gap)
	if agentW < 20 {
		agentW = 20
	}
	if logW < 30 {
		logW = totalW - 22
		agentW = 20
	}
	return
}

func (m GroupHouseModel) renderAgentPanel() string {
	agentW, _ := m.panelWidths()
	style := CardStyle.Copy().Width(agentW)

	var items strings.Builder
	items.WriteString(CardHeaderStyle.Render("👥 Agents") + "\n\n")

	if len(m.agents) == 0 {
		items.WriteString(CaptionStyle.Render("  No agents connected\n") + "\n")
		items.WriteString(CaptionStyle.Render("  Agents can join with:\n"))
		items.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#A3E635")).Render(fmt.Sprintf("  ws://localhost:%d\n", m.port)))
	} else {
		for _, a := range m.agents {
			status := ghAgentOnlineStyle.Render("●")
			name := ghAgentIdleStyle.Render(a.Name)
			kind := CaptionStyle.Render(string(a.Kind))
			items.WriteString(fmt.Sprintf("  %s %s (%s)\n", status, name, kind))
		}
	}

	items.WriteString("\n")
	items.WriteString(CaptionStyle.Render("📁 workspace"))
	items.WriteString(fmt.Sprintf("\n%s", lipgloss.NewStyle().Foreground(lipgloss.Color("#A3E635")).Render(m.workspace)))

	return style.Render(items.String())
}

func (m GroupHouseModel) renderModelMenu() string {
	if len(m.availableModels) == 0 {
		return ""
	}
	style := CardActiveStyle.Copy().
		MarginLeft(2).
		Background(lipgloss.Color("#1A1A24"))

	var menu strings.Builder
	menu.WriteString(CardHeaderStyle.Render("Select Local Model") + "\n")
	for _, model := range m.availableModels {
		prefix := "  "
		if model == m.selectedModel {
			prefix = "• "
		}
		menu.WriteString(prefix + model + "\n")
	}
	return style.Render(menu.String())
}

func (m GroupHouseModel) renderLogPanel() string {
	_, logW := m.panelWidths()
	style := CardStyle.Copy().Width(logW)

	var items strings.Builder
	items.WriteString(CardHeaderStyle.Render("📋 Activity") + "\n\n")

	if len(m.log) == 0 {
		items.WriteString(CaptionStyle.Render("  Waiting for activity...\n"))
	} else {
		pageSize := 15
		start := len(m.log) - pageSize - m.logOffset
		if start < 0 {
			start = 0
		}
		end := start + pageSize
		if end > len(m.log) {
			end = len(m.log)
		}
		for i := start; i < end; i++ {
			entry := m.log[i]
			t := ghLogTimestampStyle.Render(entry.Timestamp.Format("15:04:05"))
			icon := "💬"
			switch entry.MsgType {
			case "file":
				icon = "📝"
			case "run":
				icon = "⚡"
			case "join":
				icon = "👋"
			case "leave":
				icon = "👋"
			}
			line := fmt.Sprintf("  %s %s %s: %s", icon, t, ghLogSenderStyle.Render(entry.Sender), ghLogTextStyle.Render(entry.Text))
			if len(line) > 48 {
				line = line[:48] + "…"
			}
			items.WriteString(line + "\n")
		}
	}
	return style.Render(items.String())
}

func (m GroupHouseModel) renderInputBar() string {
	modelName := m.selectedModel
	if modelName == "" {
		modelName = "No Model"
	}

	modelSelector := lipgloss.NewStyle().Foreground(Accent).Render(fmt.Sprintf("[%s ▼] ", modelName))
	prompt := modelSelector + "> "

	inputSty := InputStyle
	if m.inputActive {
		inputSty = FocusedStyle
	}

	text := m.inputText
	if !m.inputActive {
		text = "Press 'i' to send a message to all agents"
	}

	return inputSty.Render(prompt + text)
}
