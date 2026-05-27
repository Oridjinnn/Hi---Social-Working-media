package grouphouse

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)


// Cmd wraps an OS command execution for the group house.
type Cmd struct {
	Command   string
	WorkDir   string
	Timeout   time.Duration
}

func NewCmd(command, workDir string) *Cmd {
	return &Cmd{
		Command: command,
		WorkDir: workDir,
	}
}

func (c *Cmd) Run() (string, string, int, error) {
	// Parse the command into shell and args
	var cmd *exec.Cmd
	shell := "sh"
	shellFlag := "-c"

	// Use bash if available
	if hasBash() {
		shell = "bash"
	}

	cmd = exec.Command(shell, shellFlag, c.Command)
	cmd.Dir = c.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()

	if c.Timeout > 0 {
		timer := time.AfterFunc(c.Timeout, func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
		defer timer.Stop()
	}

	err := cmd.Run()
	_ = time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start
			return "", fmt.Sprintf("error: %v", err), 1, err
		}
	}

	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())

	return out, errOut, exitCode, nil
}

// FormatDuration returns a human-readable duration string.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		secs := int(d.Seconds())
		return fmt.Sprintf("%ds", secs)
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", mins, secs)
}

func hasBash() bool {
	_, err := exec.LookPath("bash")
	return err == nil
}