package utils

import (
	"os/exec"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
// Tries xclip, xsel, wl-copy (Wayland), then falls back silently.
func CopyToClipboard(text string) error {
	r := strings.NewReader(text)

	// Try xclip (X11)
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = r
		if cmd.Run() == nil {
			return nil
		}
	}

	// Try xsel (X11 alternative)
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "--clipboard", "--input")
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return nil
		}
	}

	// Try wl-copy (Wayland)
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return nil
		}
	}

	// Try termux-clipboard-set (Android)
	if _, err := exec.LookPath("termux-clipboard-set"); err == nil {
		cmd := exec.Command("termux-clipboard-set")
		cmd.Stdin = strings.NewReader(text)
		cmd.Run() // best-effort
	}

	return nil // silently ignore — clipboard is best-effort
}
