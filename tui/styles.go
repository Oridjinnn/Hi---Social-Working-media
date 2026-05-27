package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Color Palette ──────────────────────────────────────────────────────────
var (
	Primary      = lipgloss.Color("#A78BFA") // Softer Purple
	PrimaryLight = lipgloss.Color("#C4B5FD")
	Secondary    = lipgloss.Color("#22D3EE")
	Success      = lipgloss.Color("#34D399")
	Warning      = lipgloss.Color("#FBBF24")
	Danger       = lipgloss.Color("#F87171")
	Muted        = lipgloss.Color("#4B5563")
	MutedLight   = lipgloss.Color("#9CA3AF")
	Accent       = lipgloss.Color("#60A5FA")

	Background   = lipgloss.Color("#0F172A") // Deeper Dark
	Surface      = lipgloss.Color("#1E293B")
	SurfaceAlt   = lipgloss.Color("#334155")
	Foreground   = lipgloss.Color("#F3F4F6")
	BorderColor  = lipgloss.Color("#334155") // Muted borders
	ActiveBorder = lipgloss.Color("#4C1D95")
)

// ── Typographic Scale ──────────────────────────────────────────────────────

var (
	H1Style = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Padding(0, 1)

	H2Style = lipgloss.NewStyle().
		Foreground(Foreground).
		Bold(true).
		Padding(0, 1)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(MutedLight).
			Italic(true).
			MarginBottom(1)

	BodyStyle = lipgloss.NewStyle().
			Foreground(Foreground).
			Padding(0, 1)

	CaptionStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 1)

	LabelStyle = lipgloss.NewStyle().
			Foreground(MutedLight).
			Padding(0, 1)

	BadgeStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Muted).
			Padding(0, 1)
)

// ── Card Primitives ────────────────────────────────────────────────────────

var (
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(BorderColor).
			Padding(0, 2)

	CardActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(ActiveBorder).
			Padding(0, 2)

	CardHeaderStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	CardBodyStyle = lipgloss.NewStyle().
			Foreground(Foreground).
			Padding(0, 1)
)

// ── Status & Signal Type Badges ───────────────────────────────────────────

var (
	StatusStyle        = lipgloss.NewStyle().Padding(0, 1)
	OpenStatusStyle    = StatusStyle.Copy().Foreground(Success).Bold(true)
	FilledStatusStyle  = StatusStyle.Copy().Foreground(Muted)
	ExpiredStatusStyle = StatusStyle.Copy().Foreground(Danger)

	SignalTypeStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true).
			Padding(0, 1)

	StackStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3E635")).
			Padding(0, 1)
)

// ── General UI Styles ─────────────────────────────────────────────────────

var (
	DetailStyle = lipgloss.NewStyle().
			Padding(1, 2)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(1, 0)

	SubtleHelpStyle = lipgloss.NewStyle().
			Foreground(MutedLight).
			Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true).
			Padding(0, 1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true).
			Padding(0, 1)
)

// ── Toast ──────────────────────────────────────────────────────────────────

var (
	ToastStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			Background(Surface).
			Foreground(Foreground).
			Padding(0, 1).
			Margin(1, 0)

	ToastSuccessStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Success).
				Background(Surface).
				Foreground(Foreground).
				Padding(0, 1).
				Margin(1, 0)

	ToastWarningStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Warning).
				Background(Surface).
				Foreground(Foreground).
				Padding(0, 1).
				Margin(1, 0)
)

// ── Logo & Branding ────────────────────────────────────────────────────────

var (
	LogoStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginBottom(1)

	ProfileStatStyle = lipgloss.NewStyle().
				Foreground(Accent).
				Bold(true)
)

// ── Input Styles ───────────────────────────────────────────────────────────

var (
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Muted).
			Background(lipgloss.Color("#111827")).
			Foreground(Foreground).
			Padding(0, 1)

	FocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(Primary).
			Background(Surface).
			Padding(0, 1)

	BlurredStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Muted).
			Background(lipgloss.Color("#111827")).
			Padding(0, 1)
)

// ── Sidebar ────────────────────────────────────────────────────────────────

var (
	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(BorderColor).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingRight(1)

	SidebarItemActive = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				PaddingLeft(2).
				PaddingRight(2).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(Primary)

	SidebarItemInactive = lipgloss.NewStyle().
				Foreground(MutedLight).
				PaddingLeft(3).
				PaddingRight(2)
)

// ── Button Styles ─────────────────────────────────────────────────────────

var (
	ButtonStyle = lipgloss.NewStyle().
			Background(Primary).
			Foreground(Foreground).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			Bold(true)

	ButtonSecondary = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Foreground(Primary).
			Padding(0, 1)

	ButtonDanger = lipgloss.NewStyle().
			Background(Danger).
			Foreground(Foreground).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			Bold(true)

	HighlightStyle = lipgloss.NewStyle().
			Background(SurfaceAlt).
			Foreground(Foreground).
			Padding(0, 1)
)

// ── Divider ─────────────────────────────────────────────────────────────────

func RenderDivider(width int, color lipgloss.TerminalColor) string {
	if width <= 0 {
		width = 60
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Render(strings.Repeat("─", width))
}

// ── Progress Bar ───────────────────────────────────────────────────────────

func RenderProgressBar(current, total int, width int, color lipgloss.TerminalColor) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	filled := int(float64(current) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

// ── Key Hint Helper ────────────────────────────────────────────────────────

var KeyHintStyle = lipgloss.NewStyle().
	Foreground(PrimaryLight).
	Bold(true)

func RenderKeyHint(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	styled := make([]string, len(keys))
	for i, k := range keys {
		styled[i] = KeyHintStyle.Render(k)
	}
	return strings.Join(styled, " · ")
}

// ── Status Color Helper ────────────────────────────────────────────────────

func StatusColor(status string) lipgloss.Style {
	switch status {
	case "open":
		return OpenStatusStyle.Copy().Foreground(Success).Bold(true)
	case "filled":
		return FilledStatusStyle.Copy().Foreground(Muted)
	case "expired":
		return ExpiredStatusStyle.Copy().Foreground(Danger)
	default:
		return StatusStyle.Copy().Foreground(Warning)
	}
}

// ── Tab Bar ─────────────────────────────────────────────────────────────────

var (
	TabActiveStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 2).
			Background(SurfaceAlt).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(Primary)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(MutedLight).
				Padding(0, 2)
)

// ── Legacy Aliases ─────────────────────────────────────────────────────────

var (
	TitleStyle   = lipgloss.NewStyle().Foreground(Primary).Bold(true).Padding(0, 1).MarginBottom(1)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)
	AccentStyle  = lipgloss.NewStyle().Foreground(Accent)
)
