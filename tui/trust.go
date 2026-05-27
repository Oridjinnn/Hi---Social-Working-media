package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/Oridjinnn/hi/models"
)

func renderDetailTrustBadge(rank models.SignalRank) string {
	color := MutedLight
	label := string(rank.Tier)
	switch rank.Tier {
	case models.TrustHigh:
		color = Success
		label = "trusted"
	case models.TrustLow:
		color = Warning
		label = "low signal"
	case models.TrustSpam:
		color = Danger
		label = "noise"
	case models.TrustMedium:
		label = "moderate"
	}
	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("[%s]", label))
}

func renderGhostBadge(isGhost bool) string {
	if !isGhost {
		return ""
	}
	return lipgloss.NewStyle().Foreground(Accent).Bold(true).Render("  [discover]")
}
