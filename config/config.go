package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	GitHubToken     string   `json:"github_token"`
	GitHubUsername  string   `json:"github_username"`
	GitHubClientID  string   `json:"github_client_id"`
	SupabaseURL     string   `json:"supabase_url"`
	SupabaseAnonKey string   `json:"supabase_anon_key"`
	HIRepoOwner     string   `json:"hi_repo_owner"`
	HIRepoName      string   `json:"hi_repo_name"`
	Stack           []string `json:"stack"`
	Tier            string   `json:"tier"`
}

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "hi")
	}
	return filepath.Join(home, ".config", "hi")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func TokenPath() string {
	return filepath.Join(ConfigDir(), "token")
}

func Load() (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Prefer restricted token file over inline config value.
	tokenData, err := os.ReadFile(TokenPath())
	if err == nil && len(tokenData) > 0 {
		cfg.GitHubToken = strings.TrimSpace(string(tokenData))
	}

	// Migrate legacy inline tokens into token file.
	_ = MigrateTokenFromConfig(cfg)

	return cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Save token separately with restricted permissions
	if cfg.GitHubToken != "" {
		if err := os.WriteFile(TokenPath(), []byte(cfg.GitHubToken), 0600); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}
	}

	// Save config without token (token in separate file)
	cfgCopy := *cfg
	cfgCopy.GitHubToken = ""
	data, err := json.MarshalIndent(cfgCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// SanitizedError returns an error safe to print in UI/logs.
func SanitizedError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", RedactSecrets(err.Error()))
}

func SaveClientID(clientID string) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}
	cfg.GitHubClientID = clientID
	return Save(cfg)
}

func LoadToken() (string, error) {
	data, err := os.ReadFile(TokenPath())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SaveToken(token string) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	return os.WriteFile(TokenPath(), []byte(strings.TrimSpace(token)), 0600)
}

// GetTier returns the user's subscription tier, defaulting to "free".
func (c *Config) GetTier() string {
	if c.Tier == "" {
		return "free"
	}
	return c.Tier
}

// MsgLimit returns the message limit based on the user's tier.
func (c *Config) MsgLimit() int {
	switch c.GetTier() {
	case "pro":
		return 1000
	case "enterprise":
		return 5000
	default:
		return 100
	}
}

// SignalLimit returns the active signal limit based on the user's tier.
func (c *Config) SignalLimit() int {
	switch c.GetTier() {
	case "pro":
		return 50
	case "enterprise":
		return 200
	default:
		return 20
	}
}

// PollInterval returns the polling interval in seconds based on the user's tier.
func (c *Config) PollInterval() int {
	if c.GetTier() == "free" {
		return 30
	}
	return 15
}
