package main

import (
	"fmt"
	"os"

	"github.com/Oridjinnn/hi/cmd"
)

// Build-time variables (set via -ldflags -X)
var (
	GitHubClientID  string
	SupabaseURL     string
	SupabaseAnonKey string
	SignalRepoOwner string
	SignalRepoName  string
	Version         = "0.1.0"
)

func main() {
	// Set environment variables from build-time values if not already set
	if v := os.Getenv("HI_GITHUB_CLIENT_ID"); v == "" && GitHubClientID != "" {
		_ = os.Setenv("HI_GITHUB_CLIENT_ID", GitHubClientID)
	}
	if v := os.Getenv("HI_SUPABASE_URL"); v == "" && SupabaseURL != "" {
		_ = os.Setenv("HI_SUPABASE_URL", SupabaseURL)
	}
	if v := os.Getenv("HI_SUPABASE_ANON_KEY"); v == "" && SupabaseAnonKey != "" {
		_ = os.Setenv("HI_SUPABASE_ANON_KEY", SupabaseAnonKey)
	}
	if v := os.Getenv("HI_SIGNAL_REPO_OWNER"); v == "" && SignalRepoOwner != "" {
		_ = os.Setenv("HI_SIGNAL_REPO_OWNER", SignalRepoOwner)
	}
	if v := os.Getenv("HI_SIGNAL_REPO_NAME"); v == "" && SignalRepoName != "" {
		_ = os.Setenv("HI_SIGNAL_REPO_NAME", SignalRepoName)
	}

	fmt.Printf("HI v%s — Connect developers via structured intent signals\n\n", Version)

	cmd.Execute()
}
