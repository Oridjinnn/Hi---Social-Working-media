package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/utils"
)

type TrendRepo struct {
	Name        string
	Owner       string
	Language    string
	Stars       int
	OpenIssues  int
	PushedAt    time.Time
	HTMLURL     string
	StarVelocity string
}

type TrendData struct {
	Stack              []string
	TopRepos           []TrendRepo
	WeeklyDelta       string
	SimilarBuilders   []models.Signal
	FetchedAtUnixSecs int64
}

type trendCachePayload struct {
	TrendData      TrendData
	CachedAtUnixTS int64
}

type TrendModel struct {
	data        TrendData
	loading     bool
	err         error
	width       int
	client      *github.Client
	cfg         *config.Config
	cachePath   string
	loadedOnce  bool
}

func NewTrendModel(client *github.Client, cfg *config.Config) *TrendModel {

	stacks := []string{}
	if cfg != nil && len(cfg.Stack) > 0 {
		stacks = cfg.Stack
	} else {
		stacks = []string{"go"}
	}

	cacheDir := filepath.Join(utilsConfigDir(), "trend_cache")
	cachePath := filepath.Join(cacheDir, "trend_cache.json")
	return &TrendModel{
		loading:   true,
		client:    client,
		cfg:       cfg,
		cachePath: cachePath,
		data: TrendData{
			Stack:            stacks,
			TopRepos:         []TrendRepo{},
			WeeklyDelta:      "",
			SimilarBuilders: []models.Signal{},
		},
	}
}

func utilsConfigDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/habel"
	}
	return filepath.Join(home, ".config", "hi")
}

func (m *TrendModel) LoadCmd(signalsFetcher func([]string) ([]models.Signal, error)) tea.Cmd {
	return func() tea.Msg {
		m.LoadAsync(signalsFetcher)
		return trendLoadedMsg{}
	}
}

func (m *TrendModel) LoadAsync(signalsFetcher func([]string) ([]models.Signal, error)) {
	m.loading = true
	defer func() { m.loading = false }()

	const ttl = 6 * time.Hour

	if m.loadedOnce {
		m.loading = false
		return
	}

	var cached trendCachePayload
	zero, ok, err := utils.ReadJSONCache[trendCachePayload](m.cachePath, ttl)
	if err == nil && ok {
		cached = zero
		m.data = cached.TrendData
		m.data.FetchedAtUnixSecs = cached.CachedAtUnixTS
		m.loadedOnce = true
		return
	}

	stacks := []string{"go", "ai", "cli"}
	if m.cfg != nil {
		_ = stacks
	}
	m.data.Stack = stacks

	topRepos := make([]TrendRepo, 0, 5)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, st := range stacks {
		wg.Add(1)
		go func(st string) {
			defer wg.Done()
			lang := st
			if strings.EqualFold(st, "ai") {
				lang = "Go"
			}
			resRepos, err := fetchTrendingByLanguage(m.client, strings.ToLower(lang), 5)
			if err == nil {
				mu.Lock()
				topRepos = append(topRepos, resRepos...)
				mu.Unlock()
			}
		}(st)
	}
	wg.Wait()

	uniq := map[string]TrendRepo{}
	for _, r := range topRepos {
		k := fmt.Sprintf("%s/%s", r.Owner, r.Name)
		if _, ok := uniq[k]; !ok {
			uniq[k] = r
		}
	}
	m.data.TopRepos = nil
	for _, r := range uniq {
		m.data.TopRepos = append(m.data.TopRepos, r)
	}
	sort.Slice(m.data.TopRepos, func(i, j int) bool { return m.data.TopRepos[i].Stars > m.data.TopRepos[j].Stars })
	if len(m.data.TopRepos) > 5 {
		m.data.TopRepos = m.data.TopRepos[:5]
	}

	m.data.WeeklyDelta = fmt.Sprintf("%s projects trending this week", strings.Title(stacks[0]))

	if signalsFetcher != nil {
		sigs, _ := signalsFetcher(stacks)
		m.data.SimilarBuilders = sigs
	}

	payload := trendCachePayload{TrendData: m.data, CachedAtUnixTS: time.Now().Unix()}
	_ = utils.WriteJSONCache(m.cachePath, payload)
	m.loadedOnce = true
}

func fetchTrendingByLanguage(client *github.Client, language string, perPage int) ([]TrendRepo, error) {
	dateLimit := time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	q := fmt.Sprintf("language:%s created:>%s", language, dateLimit)
	apiURL := fmt.Sprintf("%s/search/repositories?q=%s&sort=stars&order=desc&per_page=%d", github.BaseURL, url.QueryEscape(q), perPage)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var res struct {
		Items []struct {
			FullName   string    `json:"full_name"`
			Owner      struct{ Login string } `json:"owner"`
			Name       string    `json:"name"`
			Language   string    `json:"language"`
			Stargazers  int       `json:"stargazers_count"`
			OpenIssues  int      `json:"open_issues_count"`
			PushedAt    time.Time `json:"pushed_at"`
			HTMLURL     string    `json:"html_url"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	out := make([]TrendRepo, 0, len(res.Items))
	for _, it := range res.Items {
		out = append(out, TrendRepo{
			Name:     it.Name,
			Owner:    it.Owner.Login,
			Language: it.Language,
			Stars:    it.Stargazers,
			OpenIssues: it.OpenIssues,
			PushedAt:  it.PushedAt,
			HTMLURL:   it.HTMLURL,
		})
	}
	return out, nil
}

func (m *TrendModel) View() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	if m.loading {
		return CardStyle.Width(width).Render(CaptionStyle.Render("  🔥 Trending projects (loading...)"))
	}
	if m.err != nil {
		return CardStyle.Width(width).Render(lipgloss.NewStyle().Foreground(Danger).Render("  🔥 Trending unavailable"))
	}

	s := ""
	s += CardHeaderStyle.Render("🔥 Trending Projects") + "\n"
	if m.data.WeeklyDelta != "" {
		s += CaptionStyle.Render(fmt.Sprintf("  %s\n", m.data.WeeklyDelta))
	}
	s += "\n"

	for i, r := range m.data.TopRepos {
		if i >= 3 {
			break
		}
		s += fmt.Sprintf("  %s %s (%d ★)\n",
			lipgloss.NewStyle().Foreground(Secondary).Render("•"),
			r.FullName(), r.Stars)
	}

	if len(m.data.SimilarBuilders) > 0 {
		s += "\n"
		s += CardHeaderStyle.Render("People like you are building:") + "\n"
		for _, sig := range m.data.SimilarBuilders {
			if sig.Title == "" {
				continue
			}
			s += fmt.Sprintf("  %s %s (%d ★)\n",
				lipgloss.NewStyle().Foreground(Secondary).Render("•"),
				sig.Title, sig.ViewCount)
		}
	}
	s += "\n"

	if len(m.data.Stack) > 0 {
		s += "\n"
		s += fmt.Sprintf("  Your stack: %s\n", strings.Join(m.data.Stack, ", "))
	}

	return CardStyle.Width(width).Render(lipgloss.NewStyle().Foreground(Primary).Render(strings.TrimRight(s, "\n")))
}

func (r TrendRepo) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type trendLoadedMsg struct{}