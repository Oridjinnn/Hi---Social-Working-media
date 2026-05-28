package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// AuthLevel describes how ready GitHub auth is for HI.
type AuthLevel string

const (
	AuthOK       AuthLevel = "ok"
	AuthMissing  AuthLevel = "missing"
	AuthInvalid  AuthLevel = "invalid"
	AuthDegraded AuthLevel = "degraded"
)

// AuthReport is the result of startup auth validation.
type AuthReport struct {
	Level    AuthLevel
	Issues   []string
	Hints    []string
	Username string
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(ghp_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`(?i)\b(github_pat_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`(?i)\b(gho_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`(?i)\b(ghu_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`(?i)\b(ghs_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`(?i)\b(ghr_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-]+`),
	regexp.MustCompile(`(?i)Authorization:\s*\S+`),
	regexp.MustCompile(`(?i)"github_token"\s*:\s*"[^"]+"`),
}

// RedactSecrets removes likely credentials from strings shown to users/logs.
func RedactSecrets(s string) string {
	out := s
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}

// AuditTokenStorage checks how credentials are stored on disk.
func AuditTokenStorage(cfg *Config) (issues, hints []string) {
	if cfg == nil {
		return []string{"config is nil"}, []string{"run: hi auth login"}
	}

	// Token duplicated in config.json is a hygiene risk.
	if hasTokenInConfigFile() {
		issues = append(issues, "GitHub token found in config.json (should only live in token file)")
		hints = append(hints, "run: hi auth login (HI will migrate token to ~/.config/hi/token)")
	}

	info, err := os.Stat(TokenPath())
	if err != nil {
		if cfg.GitHubToken != "" {
			issues = append(issues, "token file missing while token is loaded in memory")
			hints = append(hints, "run: hi auth login to rewrite secure token storage")
		}
		return issues, hints
	}

	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		issues = append(issues, fmt.Sprintf("token file permissions are too open (%#o)", mode))
		hints = append(hints, "run: chmod 600 "+TokenPath())
	}

	dirInfo, err := os.Stat(ConfigDir())
	if err == nil {
		dirMode := dirInfo.Mode().Perm()
		if dirMode&0007 != 0 {
			issues = append(issues, fmt.Sprintf("config directory permissions are too open (%#o)", dirMode))
			hints = append(hints, "run: chmod 700 "+ConfigDir())
		}
	}

	return issues, hints
}

func hasTokenInConfigFile() bool {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	v, ok := raw["github_token"]
	if !ok {
		return false
	}
	var token string
	if err := json.Unmarshal(v, &token); err != nil {
		return false
	}
	return strings.TrimSpace(token) != ""
}

// MigrateTokenFromConfig moves inline config.json tokens into the restricted token file.
func MigrateTokenFromConfig(cfg *Config) error {
	if cfg == nil || strings.TrimSpace(cfg.GitHubToken) == "" {
		return nil
	}
	if !hasTokenInConfigFile() {
		return nil
	}
	if err := SaveToken(cfg.GitHubToken); err != nil {
		return err
	}
	return Save(cfg)
}

// ValidateAuth checks credential presence, storage hygiene, and token validity.
func ValidateAuth(cfg *Config) AuthReport {
	report := AuthReport{Level: AuthOK}
	if cfg != nil {
		report.Username = cfg.GitHubUsername
	}

	storageIssues, storageHints := AuditTokenStorage(cfg)
	report.Issues = append(report.Issues, storageIssues...)
	report.Hints = append(report.Hints, storageHints...)

	if cfg == nil || strings.TrimSpace(cfg.GitHubToken) == "" {
		report.Level = AuthMissing
		report.Issues = append(report.Issues, "GitHub token not configured")
		report.Hints = append(report.Hints,
			"run: hi auth login",
			"then verify: hi auth status",
		)
		return report
	}

	if err := validateGitHubToken(cfg.GitHubToken); err != nil {
		report.Level = AuthInvalid
		report.Issues = append(report.Issues, RedactSecrets(err.Error()))
		report.Hints = append(report.Hints,
			"run: hi auth login --force",
			"check token scopes include: public_repo, read:user",
		)
		return report
	}

	if len(storageIssues) > 0 {
		report.Level = AuthDegraded
	}
	return report
}

func validateGitHubToken(token string) error {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hi-cli/1.0")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validating GitHub token: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("GitHub rejected the stored token (unauthorized)")
	}
	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("GitHub token lacks required access: %s", RedactSecrets(msg))
		}
		return fmt.Errorf("GitHub token lacks required access")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub auth check failed: %s", resp.Status)
	}
	return nil
}

// FormatAuthHints returns user-facing remediation text.
func (r AuthReport) FormatAuthHints() string {
	if len(r.Hints) == 0 {
		return ""
	}
	var b strings.Builder
	for _, h := range r.Hints {
		b.WriteString("  • ")
		b.WriteString(h)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
