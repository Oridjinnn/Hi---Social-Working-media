package models

import (
	"testing"
	"time"
)

func TestRankSignalsOrdersHighBeforeSpam(t *testing.T) {
	now := mustParseTime("2026-05-27T12:00:00Z")
	old := mustParseTime("2026-01-01T12:00:00Z")

	good := Signal{
		ID:         1,
		Title:      "[contributor] Solid project looking for help",
		Body:       "Detailed body with enough context to evaluate fit and intent for collaboration.",
		CreatedAt:  now,
		Project:    "HI",
		Stack:      []string{"go", "tui"},
		Needs:      []string{"contributor"},
		ContactURL: "https://github.com/example",
		Type:       SignalTypeContributor,
		Status:     SignalStatusOpen,
		Author:     User{ConnectionCount: 10, SuccessCount: 7},
	}

	noise := Signal{
		ID:        2,
		Title:     "help",
		CreatedAt: old,
		Status:    SignalStatusOpen,
	}

	out := RankSignals([]Signal{noise, good})
	if out[0].ID != good.ID {
		t.Fatalf("expected good signal first, got #%d", out[0].ID)
	}
}

func TestScoreSignalTier(t *testing.T) {
	s := Signal{
		Title:      "[contributor] Build CLI tooling",
		Body:       "Looking for a collaborator with Go experience to ship a polished terminal workflow.",
		CreatedAt:  mustParseTime("2026-05-26T12:00:00Z"),
		Project:    "HI",
		Stack:      []string{"go"},
		Needs:      []string{"contributor"},
		ContactURL: "https://github.com/example",
		Status:     SignalStatusOpen,
	}
	r := ScoreSignal(s)
	if r.Tier != TrustHigh && r.Tier != TrustMedium {
		t.Fatalf("expected high/medium tier, got %s (score %.1f)", r.Tier, r.Score)
	}
}

func mustParseTime(v string) (t time.Time) {
	t, _ = time.Parse(time.RFC3339, v)
	return t
}
