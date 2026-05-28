package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
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

	scrollIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).Italic(true)
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

// Models known to work well for sentiment/analysis — ordered by preference.
var preferredAnalysisModels = []string{
	"phi3", "phi4", "llama3.2", "llama3", "qwen2.5", "mistral", "gemma2", "gemma", "codellama", "tinyllama",
}

// ── MarketModel ───────────────────────────────────────────────────────────────

type marketLoadedMsg struct{ report *market.MarketReport }
type marketErrMsg struct{ err error }

type marketArea int

const (
	areaTrends marketArea = iota
	areaOpportunities
	areaAnalysis
)

type Opportunity struct {
	Target    string
	Problem   string
	Potential string
	Adoption  string
	Sentiment string
	Velocity  string
}

// Proper struct message types to avoid type assertion issues.
type marketModelsLoadedMsg struct{ models []string }
type marketAnalysisMsg struct{ text string }
type marketPullDoneMsg struct{ err error }
type marketPullProgressMsg struct{ text string }

var suitableModelsForAnalysis = []string{
	"phi3", "phi4", "llama3.2", "llama3", "qwen2.5", "mistral", "gemma2", "gemma",
}

type MarketModel struct {
	report          *market.MarketReport
	suggestions     []Opportunity
	availableModels []string
	loading         bool
	pulling         bool
	analyzing       bool
	analysis        string
	rawAnalysis     string // the full analysis text (before potential wrapping)
	err             error
	width           int
	height          int
	cursor          int
	oppCursor       int
	focus           marketArea
	exported        bool
	loaded          bool
	// Scroll support for analysis pane
	analysisVP        viewport.Model
	analysisReady     bool
	analysisViewportH int // height allocated for analysis viewport
}

func NewMarketModel() MarketModel {
	return MarketModel{
		width: 80,
	}
}

func (m *MarketModel) Init() tea.Cmd {
	// Only check ollama models on init — market data fetches lazily
	return m.fetchModelsCmd()
}

func (m MarketModel) fetchModelsCmd() tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("ollama"); err != nil {
			return marketModelsLoadedMsg{}
		}
		out, err := exec.Command("ollama", "list").Output()
		if err != nil {
			return marketModelsLoadedMsg{}
		}
		raw := string(out)
		lines := strings.Split(raw, "\n")
		var models []string
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			// Clean Windows \r characters and split by whitespace
			line = strings.TrimRight(line, "\r")
			fields := strings.Fields(line)
			if len(fields) > 0 {
				name := strings.TrimSpace(fields[0])
				if name != "" {
					models = append(models, name)
				}
			}
		}
		return marketModelsLoadedMsg{models: models}
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
		if err != nil {
			return marketPullDoneMsg{err: fmt.Errorf("pull %s failed: %w", modelName, err)}
		}
		return marketPullDoneMsg{}
	}
}

// fallbackInsight generates a data-driven analysis summary from the report
// without needing an LLM. Guaranteed to always return non-empty text.
func fallbackInsight(report *market.MarketReport) string {
	if report == nil || len(report.Topics) == 0 {
		return "No market data available yet."
	}

	// Find hottest by total stars
	sorted := make([]market.TopicIntel, len(report.Topics))
	copy(sorted, report.Topics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TotalStars > sorted[j].TotalStars
	})

	hot := sorted[0]

	// Count surging/rising
	var surging, rising []string
	for _, t := range report.Topics {
		if strings.Contains(t.Momentum, "🚀") {
			surging = append(surging, t.Topic.Label)
		} else if strings.Contains(t.Momentum, "📈") {
			rising = append(rising, t.Topic.Label)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔥 Hottest: %s — ★%s across top repos",
		hot.Topic.Label, marketFmtNum(hot.TotalStars)))

	if len(surging) > 0 {
		b.WriteString(fmt.Sprintf(" | Surging: %s", strings.Join(surging, ", ")))
	}
	if len(rising) > 0 {
		b.WriteString(fmt.Sprintf(" | Rising: %s", strings.Join(rising, ", ")))
	}

	// Total ecosystem stats
	var totalStars, totalHN, totalNew int
	for _, t := range report.Topics {
		totalStars += t.TotalStars
		totalHN += t.HNHits
		totalNew += t.NewRepos
	}
	b.WriteString(fmt.Sprintf("\n📊 Ecosystem: ★%s total stars · %d HN stories (7d) · +%d new repos (30d)",
		marketFmtNum(totalStars), totalHN, totalNew))

	return b.String()
}

// selectBestModel picks the best available model for analysis.
// Returns "" if no suitable model is found.
func selectBestModel(available []string) (string, bool) {
	// First, check preferred models in order
	for _, target := range suitableModelsForAnalysis {
		for _, model := range available {
			if strings.HasPrefix(strings.ToLower(model), target) ||
				strings.Contains(strings.ToLower(model), target) {
				return model, true
			}
		}
	}
	// Fallback: return any model at all
	if len(available) > 0 {
		return available[0], false
	}
	return "", false
}

func (m MarketModel) analyzeSentimentCmd() tea.Cmd {
	return func() tea.Msg {
		// Always generate fallback insight first
		fallback := fallbackInsight(m.report)

		_, ollamaErr := exec.LookPath("ollama")
		if ollamaErr != nil {
			return marketAnalysisMsg{text: fallback}
		}

		if m.report == nil || len(m.report.Topics) == 0 {
			return marketAnalysisMsg{text: fallback}
		}

		if m.cursor < 0 || m.cursor >= len(m.report.Topics) {
			return marketAnalysisMsg{text: fallback}
		}
		topic := m.report.Topics[m.cursor]

		selectedModel, found := selectBestModel(m.availableModels)
		if selectedModel == "" {
			// No model at all — return fallback + suggest pulling
			return marketAnalysisMsg{
				text: fallback + "\n\n💡 No Ollama model found. Press 'p' to install a model for AI-powered insights.",
			}
		}

		prompt := fmt.Sprintf(
			"Act as a technology market analyst. Summarize developer adoption and 'vibe' for '%s' in 1-2 concise sentences. Focus on actionable insight.",
			topic.Topic.Label,
		)

		cmd := exec.Command("ollama", "run", selectedModel, prompt)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()

		if err != nil {
			msg := fallback
			if !found {
				msg += fmt.Sprintf("\n\n⚠️  Model '%s' not found locally. Press 'p' to pull it.", selectedModel)
			} else if strings.Contains(stderr.String(), "not found") {
				msg += fmt.Sprintf("\n\n⚠️  Model '%s' not found locally. Press 'p' to pull it.", selectedModel)
			} else {
				msg += fmt.Sprintf("\n\n⚠️  AI analysis unavailable (%s)", config.RedactSecrets(err.Error()))
			}
			return marketAnalysisMsg{text: msg}
		}

		aiText := strings.TrimSpace(string(out))
		if aiText == "" {
			return marketAnalysisMsg{text: fallback}
		}

		// Combine: fallback (data) + AI insight
		combined := fallback + "\n\n🤖 " + aiText
		return marketAnalysisMsg{text: combined}
	}
}

func (m MarketModel) Update(msg tea.Msg) (MarketModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case marketModelsLoadedMsg:
		m.availableModels = msg.models
		if m.report != nil && m.rawAnalysis == "" && !m.analyzing {
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
		// Always generate analysis — fallback + AI if available
		m.analyzing = true
		return m, m.analyzeSentimentCmd()

	case marketErrMsg:
		m.err = msg.err
		m.loading = false

	case marketAnalysisMsg:
		m.rawAnalysis = msg.text
		m.analyzing = false
		// Update viewport content
		m.analysis = formatAnalysisContent(msg.text, m.width)
		if !m.analysisReady {
			availH := 10
			if m.height > 20 {
				availH = m.height / 3
				if availH < 6 {
					availH = 6
				}
			}
			m.initAnalysisViewport(m.width, availH)
		}
		m.analysisVP.SetContent(m.analysis)
		m.analysisVP.GotoTop()

	case marketPullDoneMsg:
		m.pulling = false
		if msg.err != nil {
			m.rawAnalysis = fmt.Sprintf("Pull failed: %s", msg.err.Error())
			m.analysis = formatAnalysisContent(m.rawAnalysis, m.width)
		} else {
			m.rawAnalysis = "Model pulled successfully! Running analysis..."
			m.analysis = formatAnalysisContent(m.rawAnalysis, m.width)
			m.analyzing = true
			return m, m.analyzeSentimentCmd()
		}

	case marketPullProgressMsg:
		m.rawAnalysis = msg.text
		m.analysis = formatAnalysisContent(msg.text, m.width)

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "ctrl+n":
			// Cycle through focus areas: Trends -> Analysis -> Opportunities -> back
			switch m.focus {
			case areaTrends:
				m.focus = areaAnalysis
			case areaAnalysis:
				m.focus = areaOpportunities
			case areaOpportunities:
				m.focus = areaTrends
			}

		case "up", "k":
			if m.focus == areaAnalysis && m.analysisReady {
				m.analysisVP.LineUp(1)
				return m, nil
			}
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
			if m.focus == areaAnalysis && m.analysisReady {
				m.analysisVP.LineDown(1)
				return m, nil
			}
			if m.focus == areaTrends {
				if m.report != nil && m.cursor < len(m.report.Topics)-1 {
					m.cursor++
					m.analyzing = true
					m.rawAnalysis = ""
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
			if strings.Contains(m.rawAnalysis, "Press 'p' to") && !m.pulling {
				m.pulling = true
				targetModel := "phi3:mini"
				if selected, found := selectBestModel(m.availableModels); found || len(m.availableModels) == 0 {
					if found {
						targetModel = selected
					}
				}
				m.rawAnalysis = fmt.Sprintf("Pulling %s... (this may take a few minutes)", targetModel)
				m.analysis = formatAnalysisContent(m.rawAnalysis, m.width)
				return m, m.pullModelCmd(targetModel)
			}

		case "r":
			m.loading = true
			m.err = nil
			m.exported = false
			m.rawAnalysis = ""
			m.analysis = ""
			return m, fetchMarketCmd()
		}
	}

	// If analysis viewport is active and ready, pass resize/key msgs to it
	if m.analysisReady {
		var vpCmd tea.Cmd
		m.analysisVP, vpCmd = m.analysisVP.Update(msg)
		if vpCmd != nil {
			return m, vpCmd
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

	// Analysis panel with scroll support
	analysisColor := BorderColor
	if m.focus == areaAnalysis {
		analysisColor = ActiveBorder
	}
	analysisStyle := CardStyle.Copy().Width(w - 4).BorderForeground(analysisColor)

	analysisText := m.analysis
	if m.analyzing {
		analysisText = CaptionStyle.Render("Analyzing market sentiment with local Phi-3...")
	}
	if analysisText != "" {
		title := CardHeaderStyle.Render("🤖 STRATEGIC INTELLIGENCE")
		// Use viewport for analysis if content is large enough
		if m.analysisReady && !m.analyzing {
			// Check if content exceeds viewport height
			availH := 10
			if m.height > 20 {
				availH = m.height / 3
				if availH < 6 {
					availH = 6
				}
			}
			m.analysisVP.Width = w - 8
			m.analysisVP.Height = availH
			content := title + "\n" + m.analysis
			m.analysisVP.SetContent(content)
			rendered := m.analysisVP.View()
			// Add scroll indicator if needed
			if m.analysisVP.TotalLineCount() > m.analysisVP.VisibleLineCount() {
				pct := int(float64(m.analysisVP.YOffset) / float64(m.analysisVP.TotalLineCount()-m.analysisVP.VisibleLineCount()) * 100)
				rendered += "\n" + scrollIndicatorStyle.Render(fmt.Sprintf("  ↑/↓ scroll · %d%%", pct))
			}
			b.WriteString(analysisStyle.Render(rendered) + "\n\n")
		} else {
			b.WriteString(analysisStyle.Render(title+"\n"+analysisText) + "\n\n")
		}
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

// ── Helpers ──────────────────────────────────────────────────────────────────

// formatAnalysisContent wraps analysis text for display (pass-through for now,
// could add wrapping/truncation logic later).
func formatAnalysisContent(text string, width int) string {
	return text
}

// marketFmtNum formats a number with k suffix (same as fmtNum but public name).
func marketFmtNum(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// initAnalysisViewport sets up a scrollable viewport for the analysis pane.
func (m *MarketModel) initAnalysisViewport(width, height int) {
	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	m.analysisVP = vp
	m.analysisReady = true
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
