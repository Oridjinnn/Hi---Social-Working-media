package history

import (
	"os"
	"testing"
	"time"

	"github.com/Oridjinnn/hi/models"
)

func TestLogSignalVisitAndLoad(t *testing.T) {
	tmp := t.TempDir()
	// point config dir to temp home
	_ = os.Setenv("HOME", tmp)

	h, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(h.SignalVisits) != 0 {
		t.Fatalf("expected empty history, got %d", len(h.SignalVisits))
	}

	sig := &models.Signal{ID: 42, Title: "Hello World", GitHubURL: "https://github.com/example/repo", Author: models.User{GitHubUsername: "alice"}}
	h.LogSignalVisit(sig)

	h2, err := Load()
	if err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}
	if len(h2.SignalVisits) != 1 {
		t.Fatalf("expected 1 signal visit, got %d", len(h2.SignalVisits))
	}
	v := h2.SignalVisits[0]
	if v.SignalID != 42 || v.Title != "Hello World" || v.Author != "alice" {
		t.Fatalf("unexpected visit data: %+v", v)
	}
}

func TestLogChatSession(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)

	h, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	sig := &models.Signal{ID: 7, Title: "ChatMe", GitHubURL: "https://github.com/x/y", Author: models.User{GitHubUsername: "bob"}}
	h.LogChatSession(sig)

	h2, err := Load()
	if err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}
	if len(h2.ChatSessions) != 1 {
		t.Fatalf("expected 1 chat session, got %d", len(h2.ChatSessions))
	}
	s := h2.ChatSessions[0]
	if s.SignalID != 7 || s.Title != "ChatMe" || s.Author != "bob" {
		t.Fatalf("unexpected chat data: %+v", s)
	}
}

func TestLogGroupEvent(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)

	h, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	h.LogGroupEvent("agent joined")

	h2, err := Load()
	if err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}
	if len(h2.GroupEvents) != 1 {
		t.Fatalf("expected 1 group event, got %d", len(h2.GroupEvents))
	}
	e := h2.GroupEvents[0]
	if e.Event != "agent joined" || e.Kind != "grouphouse" {
		t.Fatalf("unexpected group event: %+v", e)
	}
	// timestamp should be recent
	if time.Since(e.Timestamp) > time.Minute {
		t.Fatalf("event timestamp too old: %v", e.Timestamp)
	}
}
