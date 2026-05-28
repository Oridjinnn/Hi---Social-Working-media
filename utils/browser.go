package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// OpenURL opens a URL in the default browser.
// Tries multiple methods and falls back to printing instructions.
func OpenURL(url string) error {
	var err error

	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	case "windows":
		// Method 1: start (most reliable on modern Windows)
		if e := exec.Command("cmd", "/c", "start", "", url).Start(); e == nil {
			return nil
		}
		// Method 2: rundll32 (legacy, still works on some configs)
		if e := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start(); e == nil {
			return nil
		}
		err = fmt.Errorf("could not open browser on Windows")
	default: // linux
		err = tryOpenLinux(url)
	}

	if err == nil {
		return nil
	}

	// Fallback: print the URL to stderr (visible even inside TUI alt screen)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Open this URL in your browser:")
	fmt.Fprintf(os.Stderr, "  \033[36m%s\033[0m\n", url)
	fmt.Fprintln(os.Stderr)
	return nil
}

func tryOpenLinux(url string) error {
	// Try xdg-open
	if err := exec.Command("xdg-open", url).Start(); err == nil {
		return nil
	}

	// Try sensible-browser
	if err := exec.Command("sensible-browser", url).Start(); err == nil {
		return nil
	}

	// Try known browsers
	browsers := []string{"brave-browser", "google-chrome", "chromium-browser", "firefox", "opera"}
	for _, b := range browsers {
		if _, err := exec.LookPath(b); err == nil {
			if err := exec.Command(b, url).Start(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no browser found")
}
