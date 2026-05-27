package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os/exec"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/market"
	"github.com/Oridjinnn/hi/utils"
)

// ── Market-specific styles (complementing shared styles) ─────────────────────

var (
	marketLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A3E635")).Bold(true)

	marketMutedStyle = lipgloss.NewStyle().
				Foreground(Muted)

	marketStarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
	marketHNStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
)

// ── Sparkline ────────────────────────────────────────────────────────────────

func sparkline(values []int, width int) string {
	if len(values) == 0 {
		return ""
	}
	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	maxV := 1
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	out := ""
	for _, v := range values {
		idx := int(float64(v) / float64(maxV) * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		out += bars[idx]
	}
	return lipgloss.NewStyle().Foreground(Accent).Render(out)
}

func topicSparkValues(intel market.TopicIntel) []int {
	return []int{intel.NewRepos, intel.HNHits / 2, intel.TotalStars / 5000}
}

// ── MarketModel ───────────────────────────────────────────────────────────────

type marketLoadedMsg struct{ report *market.MarketReport }
type marketErrMsg struct{ err error }

type marketArea int

const (
	areaTrends marketArea = iota
	areaOpportunities
)

type Opportunity struct {
	Target    string
	Problem   string
	Potential string
	Adoption  string
	Sentiment string
	Velocity  string
}

type MarketModel struct {
	report          *market.MarketReport
	suggestions     []Opportunity
	availableModels []string
	loading         bool
	pulling         bool
	analyzing       bool
	analysis        string
	err             error
	width           int
	cursor          int
	oppCursor       int
	focus           marketArea
	exported        bool
	loaded          bool
}

func NewMarketModel() MarketModel {
	return MarketModel{width: 80}
}

func (m *MarketModel) Init() tea.Cmd {
	// Only check ollama models on init — market data fetches lazily
	return m.fetchModelsCmd()
}

type marketAnalysisMsg struct{ text string }
type marketPullDoneMsg struct{ err error }
type marketModelsLoadedMsg []string

func (m MarketModel) fetchModelsCmd() tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("ollama"); err != nil {
			return marketModelsLoadedMsg{}
		}
		out, err := exec.Command("ollama", "list").Output()
		if err != nil {
			return marketModelsLoadedMsg{}
		}
		lines := strings.Split(string(out), "\n")
		var models []string
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) > 0 {
				models = append(models, fields[0])
			}
		}
		return marketModelsLoadedMsg(models)
	}
}

func fetchMarketCmd() tea.Cmd {
	return func() tea.Msg {
		report, err := market.Fetch()
		if err != nil {
			return marketErrMsg{err: err}
		}
		return marketLoadedMsg{report}
	}
}

func (m MarketModel) pullModelCmd(modelName string) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("ollama"); err != nil {
			return marketPullDoneMsg{err: fmt.Errorf("ollama not found: %w", err)}
		}
		err := exec.Command("ollama", "pull", modelName).Run()
		return marketPullDoneMsg{err: err}
	}
}

func (m MarketModel) analyzeSentimentCmd() tea.Cmd {
	return func() tea.Msg {
		_, err := exec.LookPath("ollama")
		if err != nil {
			return marketAnalysisMsg{text: "Install Ollama to enable local sentiment analysis."}
		}

		if m.report == nil || len(m.report.Topics) == 0 {
			return nil
		}

		if m.cursor < 0 || m.cursor >= len(m.report.Topics) {
			return marketAnalysisMsg{text: "No topic selected."}
		}
		topic := m.report.Topics[m.cursor].Topic.Label

		selectedModel := "phi3:mini"
		found := false
		for _, model := range m.availableModels {
			if strings.Contains(model, "phi3") || strings.Contains(model, "qwen2.5") || strings.Contains(model, "llama3") {
				selectedModel = model
				found = true
				break
			}
		}

		prompt := fmt.Sprintf(
			"Act as a technology market analyst. Summarize developer adoption and general 'vibe' for '%s' in 2 sentences.",
			topic,
		)

		cmd := exec.Command("ollama", "run", selectedModel, prompt)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()

		if err != nil {
			errMsg := fmt.Sprintf("Local model analysis failed: %v", err)
			if !found || strings.Contains(stderr.String(), "not found") {
				errMsg = fmt.Sprintf("No suitable model found for analysis. Press 'p' to pull %s.", selectedModel)
			}
			return marketAnalysisMsg{text: errMsg}
		}

		return marketAnalysisMsg{text: strings.TrimSpace(string(out))}
	}
}

func (m MarketModel) Update(msg tea.Msg) (MarketModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case marketModelsLoadedMsg:
		m.availableModels = msg
		if m.report != nil && m.analysis == "" && !m.analyzing {
			m.analyzing = true
			return m, m.analyzeSentimentCmd()
		}

	case marketLoadedMsg:
		m.report = msg.report
		m.loading = false
		m.loaded = true
		m.suggestions = []Opportunity{
			{Target: "AI Dev", Problem: "Agent Hallucination Eval", Potential: "🚀 High Search Vol", Adoption: "NPM: +15% growth", Sentiment: "Vibe: Critical for prod", Velocity: "PR Velocity: Low"},
			{Target: "Coder", Problem: "TUI Layout Engine for Go", Potential: "📈 Growing niche", Adoption: "GoProxy: 5k/wk", Sentiment: "Vibe: Enthusiastic but early", Velocity: "PR Velocity: High"},
			{Target: "Vibe Coder", Problem: "Natural Language Schema Gen", Potential: "🔥 Viral on Socials", Adoption: "Low supply", Sentiment: "Vibe: High friction", Velocity: "PR Velocity: N/A"},
		}
		if len(m.availableModels) > 0 {
			m.analyzing = true
			return m, m.analyzeSentimentCmd()
		}

	case marketErrMsg:
		m.err = msg.err
		m.loading = false

	case marketAnalysisMsg:
		m.analysis = msg.text
		m.analyzing = false

	case marketPullDoneMsg:
		m.pulling = false
		if msg.err != nil {
			m.analysis = "Pull failed: " + msg.err.Error()
		} else {
			m.analysis = "Model pulled successfully! Retrying analysis..."
			m.analyzing = true
			return m, m.analyzeSentimentCmd()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "ctrl+n":
			if m.focus == areaTrends {
				m.focus = areaOpportunities
			} else {
				m.focus = areaTrends
			}
		case "up", "k":
			if m.focus == areaTrends {
				if m.cursor > 0 {
					m.cursor--
				}
			} else {
				if m.oppCursor > 0 {
					m.oppCursor--
				}
			}
		case "down", "j":
			if m.focus == areaTrends {
				if m.report != nil && m.cursor < len(m.report.Topics)-1 {
					m.cursor++
					m.analyzing = true
					m.analysis = ""
					return m, m.analyzeSentimentCmd()
				}
			} else {
				if m.oppCursor < len(m.suggestions)-1 {
					m.oppCursor++
				}
			}
		case "e":
			if m.report != nil {
				path, err := exportHTML(m.report)
				if err == nil {
					m.exported = true
					_ = utils.OpenURL("file://" + path)
				}
			}
		case "p":
			if strings.Contains(m.analysis, "Press 'p' to pull") && !m.pulling {
				m.pulling = true
				m.analysis = "Pulling phi3:mini... (this may take a few minutes)"
				return m, m.pullModelCmd("phi3:mini")
			}
		case "r":
			m.loading = true
			m.err = nil
			m.exported = false
			return m, fetchMarketCmd()
		}
	}
	return m, nil
}

func (m MarketModel) View() string {
	var b strings.Builder

	// Default width if not set
	w := m.width
	if w < 40 {
		w = 80 // fallback for uninitialized state only
	}

	b.WriteString(SectionTitleStyle.Render("⚡ AI Market Intel") + " " + CaptionStyle.Render(time.Now().Format("15:04")) + "\n")

	if m.loading {
		b.WriteString(CaptionStyle.Faint(true).Render("  ⚡ Fetching global sentiment...") + "\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+config.RedactSecrets(m.err.Error())) + "\n")
		b.WriteString(CaptionStyle.Render("  Tip: Set GITHUB_TOKEN for higher API limits\n"))
		return b.String()
	}

	if m.report == nil {
		b.WriteString(CaptionStyle.Render("  Market intel data unavailable.\n") +
			CaptionStyle.Render("  Set GITHUB_TOKEN or check your network connection.") + "\n")
		return b.String()
	}

	b.WriteString(CaptionStyle.Render("  "+m.report.Summary) + "\n\n")

	// Analysis panel
	analysisColor := BorderColor
	if m.focus == areaTrends {
		analysisColor = ActiveBorder
	}
	analysisStyle := CardStyle.Copy().Width(w - 4).BorderForeground(analysisColor)

	analysisText := m.analysis
	if m.analyzing {
		analysisText = CaptionStyle.Render("Analyzing market sentiment with local Phi-3...")
	}
	if analysisText != "" {
		title := CardHeaderStyle.Render("🤖 STRATEGIC INTELLIGENCE")
		b.WriteString(analysisStyle.Render(title+"\n"+analysisText) + "\n\n")
	} else {
		b.WriteString("\n")
	}

	// Topics list
	var topicsList strings.Builder

	for i, t := range m.report.Topics {
		cursor := "  "
		labelStyle := marketLabelStyle
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(Accent).Render("▶ ")
			labelStyle = marketLabelStyle.Copy().Underline(true)
		}

		spark := sparkline(topicSparkValues(t), 8)
		stars := marketStarStyle.Render(fmt.Sprintf("★ %s", fmtNum(t.TotalStars)))
		hn := marketHNStyle.Render(fmt.Sprintf("HN:%d", t.HNHits))
		newR := marketMutedStyle.Render(fmt.Sprintf("+%d new/30d", t.NewRepos))

		label := labelStyle.Render(fmt.Sprintf("%-16s", t.Topic.Label))
		momentum := fmt.Sprintf("%-12s", t.Momentum)

		row := fmt.Sprintf("%s%s  %s  %s  %s  %s  %s",
			cursor, label, momentum, spark, stars, hn, newR)

		repoNames := []string{}
		for _, r := range t.TopRepos {
			repoNames = append(repoNames, r.Name)
			if len(repoNames) >= 3 {
				break
			}
		}
		if len(repoNames) > 0 {
			row += "\n  " + marketMutedStyle.Render("top: "+strings.Join(repoNames, " · "))
		}
		topicsList.WriteString(row + "\n\n")
	}

	topicsBorder := BorderColor
	if m.focus == areaTrends {
		topicsBorder = ActiveBorder
	}
	topicsContainer := CardStyle.Copy().
		Width(w - 4).
		BorderForeground(topicsBorder).
		Render(topicsList.String())

	b.WriteString(topicsContainer + "\n")

	// Opportunity cards
	if len(m.suggestions) > 0 {
		b.WriteString("\n" + H2Style.Render("💡 High-Intent Opportunities") + "\n")
		for i, opt := range m.suggestions {
			cardSty := CardStyle.Copy()
			if m.focus == areaOpportunities && i == m.oppCursor {
				cardSty = CardActiveStyle.Copy().Background(lipgloss.Color("#1A1C16"))
			}

			content := fmt.Sprintf("%s: %s\n%s · %s\n%s · %s",
				marketLabelStyle.Render(opt.Target),
				opt.Problem,
				lipgloss.NewStyle().Foreground(Success).Render(opt.Potential),
				lipgloss.NewStyle().Foreground(Accent).Render(opt.Adoption),
				lipgloss.NewStyle().Foreground(Muted).Render(opt.Sentiment),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Render(opt.Velocity))

			b.WriteString(cardSty.Render(content) + "\n")
		}
	}

	if m.exported {
		b.WriteString("\n" + SuccessStyle.Render("  ✓ Report opened in browser"))
	}

	return b.String()
}

// ── HTML Export ───────────────────────────────────────────────────────────────

func exportHTML(report *market.MarketReport) (string, error) {
	dir, err := os.MkdirTemp("", "hi-market-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "ai-market-intel.html")

	html := buildHTML(report)
	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func fmtNum(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func momentumColor(m string) string {
	switch {
	case strings.HasPrefix(m, "🚀"):
		return "#22C55E"
	case strings.HasPrefix(m, "📈"):
		return "#A3E635"
	case strings.HasPrefix(m, "➡"):
		return "#6B7280"
	default:
		return "#EF4444"
	}
}

func buildHTML(report *market.MarketReport) string {
	var rows strings.Builder
	for _, t := range report.Topics {
		repoCards := ""
		for _, r := range t.TopRepos {
			lang := r.Language
			if lang == "" {
				lang = "—"
			}
			repoCards += fmt.Sprintf(`
        <a class="repo" href="%s" target="_blank">
          <span class="repo-name">%s</span>
          <span class="repo-meta">★ %s &nbsp;·&nbsp; %s</span>
        </a>`, r.HTMLURL, r.FullName, fmtNum(r.Stars), lang)
		}

		rows.WriteString(fmt.Sprintf(`
    <div class="card">
      <div class="card-header">
        <span class="label">%s</span>
        <span class="momentum" style="color:%s">%s</span>
      </div>
      <div class="stats">
        <div class="stat"><span class="stat-val">%s</span><span class="stat-key">Total Stars</span></div>
        <div class="stat"><span class="stat-val">%d</span><span class="stat-key">HN Stories (7d)</span></div>
        <div class="stat"><span class="stat-val">+%d</span><span class="stat-key">New Repos (30d)</span></div>
      </div>
      <div class="repos">%s</div>
    </div>`, t.Topic.Label, momentumColor(t.Momentum), t.Momentum,
			fmtNum(t.TotalStars), t.HNHits, t.NewRepos, repoCards))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>HI — AI Market Intel</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Inter', system-ui, sans-serif; background: #0F0F13; color: #E2E2E8; padding: 2rem; }
  h1 { font-size: 1.5rem; color: #534AB7; margin-bottom: .25rem; }
  .meta { color: #6B7280; font-size: .85rem; margin-bottom: 2rem; }
  .summary { background: #1A1A24; border-left: 3px solid #534AB7; padding: .75rem 1rem; border-radius: 4px; margin-bottom: 2rem; color: #A3A3B8; font-size: .9rem; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1.25rem; }
  .card { background: #1A1A24; border: 1px solid #2A2A38; border-radius: 10px; padding: 1.25rem; }
  .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
  .label { font-weight: 600; font-size: 1rem; color: #E2E2E8; }
  .momentum { font-size: .85rem; }
  .stats { display: flex; gap: 1rem; margin-bottom: 1rem; }
  .stat { display: flex; flex-direction: column; }
  .stat-val { font-size: 1.2rem; font-weight: 700; color: #534AB7; }
  .stat-key { font-size: .72rem; color: #6B7280; text-transform: uppercase; letter-spacing: .05em; }
  .repos { display: flex; flex-direction: column; gap: .5rem; }
  .repo { display: flex; justify-content: space-between; align-items: center; background: #0F0F13; border-radius: 6px; padding: .5rem .75rem; text-decoration: none; }
  .repo:hover { background: #1E1E2E; }
  .repo-name { font-size: .82rem; color: #A3E635; font-family: monospace; }
  .repo-meta { font-size: .75rem; color: #6B7280; }
  footer { margin-top: 2rem; color: #3A3A50; font-size: .75rem; text-align: center; }
</style>
</head>
<body>
<h1>⚡ HI — AI Market Intel</h1>
<p class="meta">Generated %s · Data from GitHub API + Hacker News</p>
<div class="summary">%s</div>
<div class="grid">%s</div>
<footer>Built with HI · github.com/Oridjinnn/hi</footer>
</body>
</html>`,
		report.GeneratedAt.Format("02 Jan 2006 15:04 MST"),
		report.Summary,
		rows.String(),
	)
}
