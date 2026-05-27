package utils

import (
	"fmt"
	"strings"
	"time"
)

func TimeAgo(t time.Time) string {
	diff := time.Since(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		m := int(diff.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	}
	if diff < 24*time.Hour {
		h := int(diff.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	}
	if diff < 7*24*time.Hour {
		d := int(diff.Hours() / 24)
		if d == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", d)
	}
	return t.Format("Jan 2, 2006")
}

func ParseLabels(body string) []string {
	var labels []string
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "label:") || strings.HasPrefix(line, "labels:") {
			parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(line, "label:"), "labels:"), ",")
			for _, p := range parts {
				labels = append(labels, strings.TrimSpace(p))
			}
		}
	}
	return labels
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func LabelColor(label string) string {
	if strings.HasPrefix(label, "type:") {
		switch label {
		case "type:contributor":
			return "#534AB7"
		case "type:beginner":
			return "#1D9E75"
		case "type:vibe-coder":
			return "#EF9F27"
		case "type:hiring":
			return "#D85A30"
		case "type:showcase":
			return "#3B82F6"
		}
	}
	if strings.HasPrefix(label, "status:") {
		switch label {
		case "status:open":
			return "#22C55E"
		case "status:in-progress":
			return "#EAB308"
		case "status:filled":
			return "#6B7280"
		case "status:paused":
			return "#F59E0B"
		case "status:expired":
			return "#EF4444"
		}
	}
	if strings.HasPrefix(label, "commitment:") {
		return "#A3E635"
	}
	return "#0369A1"
}
