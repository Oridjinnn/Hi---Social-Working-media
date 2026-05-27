package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/models"
)

type SignalVisit struct {
	SignalID  int64     `json:"signal_id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	GitHubURL string    `json:"github_url"`
	ViewedAt  time.Time `json:"viewed_at"`
}

type ChatSession struct {
	SignalID  int64     `json:"signal_id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	GitHubURL string    `json:"github_url"`
	OpenedAt  time.Time `json:"opened_at"`
}

type GroupEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Kind      string    `json:"kind"`
}

type History struct {
	SignalVisits []SignalVisit `json:"signal_visits"`
	ChatSessions []ChatSession `json:"chat_sessions"`
	GroupEvents  []GroupEvent  `json:"group_events"`
}

func historyPath() string {
	return filepath.Join(config.ConfigDir(), "history.json")
}

func Load() (*History, error) {
	path := historyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &History{}, nil
		}
		return nil, fmt.Errorf("reading history: %w", err)
	}

	var h History
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parsing history: %w", err)
	}
	return &h, nil
}

func Save(h *History) error {
	if h == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(historyPath()), 0700); err != nil {
		return fmt.Errorf("creating history dir: %w", err)
	}
	out, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding history: %w", err)
	}
	if err := os.WriteFile(historyPath(), out, 0600); err != nil {
		return fmt.Errorf("writing history file: %w", err)
	}
	return nil
}

func (h *History) LogSignalVisit(signal *models.Signal) {
	if h == nil || signal == nil {
		return
	}
	visit := SignalVisit{
		SignalID:  signal.ID,
		Title:     signal.Title,
		Author:    signal.Author.GitHubUsername,
		GitHubURL: signal.GitHubURL,
		ViewedAt:  time.Now(),
	}
	h.SignalVisits = append([]SignalVisit{visit}, filterSignalVisits(h.SignalVisits, visit)...)
	if len(h.SignalVisits) > 30 {
		h.SignalVisits = h.SignalVisits[:30]
	}
	_ = Save(h)
}

func filterSignalVisits(list []SignalVisit, newVisit SignalVisit) []SignalVisit {
	filtered := make([]SignalVisit, 0, len(list))
	for _, existing := range list {
		if existing.SignalID == newVisit.SignalID {
			continue
		}
		filtered = append(filtered, existing)
	}
	return filtered
}

func (h *History) LogChatSession(signal *models.Signal) {
	if h == nil || signal == nil {
		return
	}
	session := ChatSession{
		SignalID:  signal.ID,
		Title:     signal.Title,
		Author:    signal.Author.GitHubUsername,
		GitHubURL: signal.GitHubURL,
		OpenedAt:  time.Now(),
	}
	h.ChatSessions = append([]ChatSession{session}, filterChatSessions(h.ChatSessions, session)...)
	if len(h.ChatSessions) > 30 {
		h.ChatSessions = h.ChatSessions[:30]
	}
	_ = Save(h)
}

func filterChatSessions(list []ChatSession, session ChatSession) []ChatSession {
	filtered := make([]ChatSession, 0, len(list))
	for _, existing := range list {
		if existing.SignalID == session.SignalID {
			continue
		}
		filtered = append(filtered, existing)
	}
	return filtered
}

func (h *History) LogGroupEvent(eventText string) {
	if h == nil || eventText == "" {
		return
	}
	entry := GroupEvent{
		Timestamp: time.Now(),
		Event:     eventText,
		Kind:      "grouphouse",
	}
	h.GroupEvents = append(h.GroupEvents, entry)
	if len(h.GroupEvents) > 60 {
		sort.SliceStable(h.GroupEvents, func(i, j int) bool {
			return h.GroupEvents[i].Timestamp.After(h.GroupEvents[j].Timestamp)
		})
		h.GroupEvents = h.GroupEvents[:60]
	}
	_ = Save(h)
}

func (h *History) RecentSignalVisits() []SignalVisit {
	if h == nil {
		return nil
	}
	return h.SignalVisits
}

func (h *History) RecentChatSessions() []ChatSession {
	if h == nil {
		return nil
	}
	return h.ChatSessions
}

func (h *History) RecentGroupEvents() []GroupEvent {
	if h == nil {
		return nil
	}
	return h.GroupEvents
}
