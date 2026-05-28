package market

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Oridjinnn/hi/utils"
)

// AITopic defines a tracked AI market segment.
type AITopic struct {
	Label   string
	Query   string // GitHub search query
	HNQuery string // Hacker News search term
}

var TrackedTopics = []AITopic{
	{Label: "LLM / Foundational Models", Query: "llm language-model ai", HNQuery: "llm"},
	{Label: "AI Agents", Query: "ai-agent autonomous-agent llm-agent", HNQuery: "ai agent"},
	{Label: "RAG / Vector Search", Query: "rag retrieval-augmented vector-search", HNQuery: "rag vector"},
	{Label: "AI CLI Tools", Query: "ai cli terminal llm", HNQuery: "ai cli"},
	{Label: "Code Generation", Query: "code-generation copilot llm-code", HNQuery: "code generation ai"},
	{Label: "Multimodal AI", Query: "multimodal vision-language image-text ai", HNQuery: "multimodal"},
}

// RepoSnapshot is a single repo data point.
type RepoSnapshot struct {
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	HTMLURL     string    `json:"html_url"`
	Stars       int       `json:"stargazers_count"`
	Forks       int       `json:"forks_count"`
	Language    string    `json:"language"`
	Description string    `json:"description"`
	PushedAt    time.Time `json:"pushed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// TopicIntel holds aggregated market data for one AI topic.
type TopicIntel struct {
	Topic      AITopic
	TopRepos   []RepoSnapshot
	TotalStars int
	NewRepos   int    // repos created in last 30 days
	HNHits     int    // HN stories mentioning this topic in last 7 days
	Momentum   string // "🚀 surging", "📈 rising", "➡ stable", "📉 cooling"
}

// MarketReport is the full compiled AI market snapshot.
type MarketReport struct {
	Topics      []TopicIntel
	GeneratedAt time.Time
	Summary     string
}

type ghSearchResponse struct {
	TotalCount int            `json:"total_count"`
	Items      []RepoSnapshot `json:"items"`
}

type hnStory struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Points      int    `json:"points"`
	NumComments int    `json:"num_comments"`
}

type hnSearchResponse struct {
	Hits []hnStory `json:"hits"`
}

func cachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "hi", "market_intel.json")
}

// Fetch returns cached market data if fresh, otherwise fetches and caches.
func Fetch() (*MarketReport, error) {
	// Try cache first (6-hour TTL)
	var cached MarketReport
	if hit, err := func() (bool, error) {
		v, ok, err := utils.ReadJSONCache[MarketReport](cachePath(), 6*time.Hour)
		if err != nil || !ok {
			return false, err
		}
		cached = v
		return true, nil
	}(); hit && err == nil {
		return &cached, nil
	}

	// Cache miss — fetch live
	report, err := fetchLive()
	if err != nil {
		return nil, err
	}

	// Write to cache (best-effort)
	_ = utils.WriteJSONCache(cachePath(), report)
	return report, nil
}

// fetchLive compiles a full MarketReport. Uses GitHub token if available.
func fetchLive() (*MarketReport, error) {
	token := os.Getenv("GITHUB_TOKEN")
	report := &MarketReport{GeneratedAt: time.Now()}

	for _, topic := range TrackedTopics {
		intel, err := fetchTopic(topic, token)
		if err != nil {
			// Partial failure: include empty entry, don't abort
			report.Topics = append(report.Topics, TopicIntel{Topic: topic, Momentum: "❓ unavailable"})
			continue
		}
		report.Topics = append(report.Topics, *intel)
	}

	report.Summary = compileSummary(report.Topics)
	return report, nil
}

func fetchTopic(topic AITopic, token string) (*TopicIntel, error) {
	intel := &TopicIntel{Topic: topic}

	// --- GitHub: top repos by stars ---
	q := url.QueryEscape(topic.Query + " stars:>50")
	apiURL := fmt.Sprintf(
		"https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=5",
		q,
	)
	req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API rate limit hit — set GITHUB_TOKEN for higher limits")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API: %s", resp.Status)
	}

	var ghResp ghSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&ghResp); err != nil {
		return nil, err
	}
	intel.TopRepos = ghResp.Items

	// Aggregate
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	for _, r := range ghResp.Items {
		intel.TotalStars += r.Stars
		if r.CreatedAt.After(thirtyDaysAgo) {
			intel.NewRepos++
		}
	}

	// --- Hacker News: story count last 7 days ---
	hnURL := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?query=%s&tags=story&numericFilters=created_at_i>%d",
		url.QueryEscape(topic.HNQuery),
		now.AddDate(0, 0, -7).Unix(),
	)
	hnResp, err := http.Get(hnURL)
	if err == nil {
		defer func() {
			_ = hnResp.Body.Close()
		}()
		if hnResp.StatusCode == 200 {
			var hn hnSearchResponse
			if err := json.NewDecoder(hnResp.Body).Decode(&hn); err == nil {
				intel.HNHits = len(hn.Hits)
			}
		}
	}

	// Momentum based on new repos + HN hits
	intel.Momentum = calcMomentum(intel.NewRepos, intel.HNHits)
	return intel, nil
}

func calcMomentum(newRepos, hnHits int) string {
	score := newRepos*2 + hnHits
	switch {
	case score >= 20:
		return "🚀 surging"
	case score >= 10:
		return "📈 rising"
	case score >= 4:
		return "➡  stable"
	default:
		return "📉 cooling"
	}
}

func compileSummary(topics []TopicIntel) string {
	// Find hottest topic
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].TotalStars > topics[j].TotalStars
	})
	if len(topics) == 0 {
		return "No data available."
	}
	hot := topics[0]
	surging := []string{}
	for _, t := range topics {
		if strings.HasPrefix(t.Momentum, "🚀") {
			surging = append(surging, t.Topic.Label)
		}
	}
	s := fmt.Sprintf("Hottest segment: %s (%s stars across top repos).", hot.Topic.Label, fmtNum(hot.TotalStars))
	if len(surging) > 0 {
		s += fmt.Sprintf(" Surging now: %s.", strings.Join(surging, ", "))
	}
	return s
}

func fmtNum(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
