package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
// Tries platform-specific methods, then falls back silently.
func CopyToClipboard(text string) error {
	r := strings.NewReader(text)

	// Windows: PowerShell Set-Clipboard
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("powershell"); err == nil {
			cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard")
			cmd.Stdin = r
			if cmd.Run() == nil {
				return nil
			}
		}
		// Fallback: clip.exe (legacy)
		if _, err := exec.LookPath("clip"); err == nil {
			cmd := exec.Command("clip")
			cmd.Stdin = r
			if cmd.Run() == nil {
				return nil
			}
		}
		return fmt.Errorf("no clipboard tool found on Windows")
	}

	// macOS: pbcopy
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("pbcopy"); err == nil {
			cmd := exec.Command("pbcopy")
			cmd.Stdin = r
			if cmd.Run() == nil {
				return nil
			}
		}
		return fmt.Errorf("pbcopy not found on macOS")
	}

	// Linux: X11 / Wayland / termux
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
		_ = cmd.Run() // best-effort
	}

	return nil // silently ignore — clipboard is best-effort
}
