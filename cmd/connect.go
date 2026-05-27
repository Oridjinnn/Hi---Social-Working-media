package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/supabase"
	"github.com/Oridjinnn/hi/utils"
)

var connectCmd = &cobra.Command{
	Use:   "connect <signal-id>",
	Short: "Connect to a signal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not authenticated. Run: hi auth login")
		}

		issueNumber, err := parseIssueNumber(args[0])
		if err != nil {
			return fmt.Errorf("invalid signal ID: %w", err)
		}

		ghClient := github.New(cfg)
		supaClient := supabase.New(cfg.SupabaseURL, cfg.SupabaseAnonKey)

		// Fetch signal details
		signal, err := ghClient.GetSignal(issueNumber)
		if err != nil {
			return fmt.Errorf("fetching signal: %w", err)
		}

		// Log connection event to Supabase
		if cfg.SupabaseURL != "" {
			event := &models.ConnectionEvent{
				SignalID:      signal.ID,
				SignalAuthor:  signal.Author.GitHubUsername,
				ActorUsername: cfg.GitHubUsername,
				EventType:     models.EventTypeConnect,
				CreatedAt:     time.Now(),
			}
			if err := supaClient.LogConnectionEvent(event); err != nil {
				fmt.Printf("(Warning: could not log connection: %v)\n", err)
			}
		}

		// Add comment to GitHub Issue
		if err := ghClient.AddConnectionComment(issueNumber, cfg.GitHubUsername); err != nil {
			fmt.Printf("(Warning: could not add comment: %v)\n", err)
		}

		fmt.Printf("✓ Connected to signal #%d!\n", issueNumber)
		fmt.Printf("  Opening @%s's contact...\n", signal.Author.GitHubUsername)

		// Open contact URL in browser
		contactURL := signal.ContactURL
		if contactURL == "" {
			contactURL = signal.Author.GitHubURL
		}
		if err := utils.OpenURL(contactURL); err != nil {
			fmt.Printf("(Warning: could not open browser: %v)\n", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}