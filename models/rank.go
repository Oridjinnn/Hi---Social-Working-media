package models

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type TrustTier string

const (
	TrustHigh   TrustTier = "high"
	TrustMedium TrustTier = "medium"
	TrustLow    TrustTier = "low"
	TrustSpam   TrustTier = "spam"
)

type SignalRank struct {
	Score float64
	Tier  TrustTier
	Hint  string
}

const spamScoreThreshold = 20.0
const lowScoreThreshold = 40.0
const highScoreThreshold = 65.0

func ScoreSignal(s Signal) SignalRank {
	score := 0.0

	age := time.Since(s.CreatedAt)
	switch {
	case age <= 7*24*time.Hour:
		score += 30
	case age <= 14*24*time.Hour:
		score += 24
	case age <= 30*24*time.Hour:
		score += 16
	case age <= 60*24*time.Hour:
		score += 8
	default:
		score += 2
	}

	if strings.TrimSpace(s.Project) != "" {
		score += 8
	}
	if len(s.Stack) > 0 {
		score += 10
	}
	if len(s.Needs) > 0 {
		score += 8
	}
	if strings.TrimSpace(s.ContactURL) != "" {
		score += 6
	}
	if strings.TrimSpace(s.Body) != "" {
		score += 8
	}

	titleLen := len(strings.TrimSpace(s.Title))
	switch {
	case titleLen >= 12 && titleLen <= 90:
		score += 10
	case titleLen >= 8:
		score += 5
	}
	bodyLen := len(strings.TrimSpace(s.Body))
	switch {
	case bodyLen >= 120:
		score += 10
	case bodyLen >= 40:
		score += 6
	case bodyLen >= 10:
		score += 2
	}

	score += authorReputationScore(s.Author)
	if s.ConnectCount > 0 {
		score += 3
	}
	if s.ViewCount > 0 && s.ConnectCount > 0 {
		rate := float64(s.ConnectCount) / float64(s.ViewCount)
		if rate >= 0.15 {
			score += 4
		} else if rate >= 0.05 {
			score += 2
		}
	}

	if s.IsGhost {
		score -= 15
	}
	if string(s.Status) == "expired" || string(s.Status) == "filled" {
		score -= 20
	}
	if titleLen < 6 {
		score -= 8
	}
	if len(s.Stack) == 0 && len(s.Needs) == 0 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	tier := trustTier(score)
	return SignalRank{
		Score: score,
		Tier:  tier,
		Hint:  reputationHint(s, tier),
	}
}

func authorReputationScore(u User) float64 {
	score := 0.0
	if u.ConnectionCount > 0 && u.SuccessCount > 0 {
		rate := float64(u.SuccessCount) / float64(u.ConnectionCount)
		score += rate * 6
	}
	if u.SignalCount >= 3 {
		score += 2
	}
	if u.IsSupporter || u.IsSeedUser {
		score += 2
	}
	if score > 10 {
		score = 10
	}
	return score
}

func trustTier(score float64) TrustTier {
	switch {
	case score < spamScoreThreshold:
		return TrustSpam
	case score < lowScoreThreshold:
		return TrustLow
	case score < highScoreThreshold:
		return TrustMedium
	default:
		return TrustHigh
	}
}

func reputationHint(s Signal, tier TrustTier) string {
	if s.Author.ConnectionCount > 0 {
		rate := int(float64(s.Author.SuccessCount) / float64(s.Author.ConnectionCount) * 100)
		if rate > 100 {
			rate = 100
		}
		return fmt.Sprintf("rep %d%% success", rate)
	}
	if s.ConnectCount >= 2 {
		return fmt.Sprintf("active · %d connects", s.ConnectCount)
	}
	switch tier {
	case TrustHigh:
		return "clear intent"
	case TrustMedium:
		return "moderate detail"
	case TrustLow:
		return "low completeness"
	default:
		return "likely noise"
	}
}

func RankSignals(signals []Signal) []Signal {
	if len(signals) <= 1 {
		return signals
	}

	type ranked struct {
		signal Signal
		rank   SignalRank
	}

	rankedList := make([]ranked, len(signals))
	for i, s := range signals {
		rankedList[i] = ranked{signal: s, rank: ScoreSignal(s)}
	}

	sort.SliceStable(rankedList, func(i, j int) bool {
		pi := tierPriority(rankedList[i].rank.Tier)
		pj := tierPriority(rankedList[j].rank.Tier)
		if pi != pj {
			return pi > pj
		}
		if rankedList[i].rank.Score != rankedList[j].rank.Score {
			return rankedList[i].rank.Score > rankedList[j].rank.Score
		}
		return rankedList[i].signal.CreatedAt.After(rankedList[j].signal.CreatedAt)
	})

	out := make([]Signal, len(signals))
	for i, r := range rankedList {
		out[i] = r.signal
	}
	return out
}

func tierPriority(t TrustTier) int {
	switch t {
	case TrustHigh:
		return 4
	case TrustMedium:
		return 3
	case TrustLow:
		return 2
	case TrustSpam:
		return 1
	default:
		return 0
	}
}
