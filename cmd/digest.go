package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/supabase"
)

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Show a weekly digest of activity",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not authenticated. Run: hi auth login")
		}

		ghClient := github.New(cfg)
		supaClient := supabase.New(cfg.SupabaseURL, cfg.SupabaseAnonKey)

		fmt.Println()
		fmt.Println("═══════════════════════════════════════")
		fmt.Println("  HI — Weekly Digest")
		fmt.Printf("  %s\n", time.Now().Format("Jan 2, 2006"))
		fmt.Println("═══════════════════════════════════════")
		fmt.Println()

		// 1. New signals in the last 7 days (MVP: fetch a few pages)
		fmt.Println("📡 New Signals")
		fmt.Println("───────────────────────────────────────")

		var allSignals []models.Signal
		for page := 1; page <= 3; page++ { // MVP: cover reasonable weekly volume
			signals, err := ghClient.ListSignals(nil, page)
			if err != nil {
				fmt.Printf("  (Could not fetch signals page %d: %v)\n", page, err)
				break
			}
			allSignals = append(allSignals, signals...)
		}

		newCount := 0
		for _, s := range allSignals {
			if time.Since(s.CreatedAt) < 7*24*time.Hour {
				fmt.Printf("  #%d [%s] %s by @%s\n", s.ID, s.Type, s.Title, s.Author.GitHubUsername)
				newCount++
			}
		}
		if newCount == 0 {
			fmt.Println("  No new signals this week.")
		}
		fmt.Println()

		// 2. Your signals' activity (Supabase events)
		fmt.Println("👀 Your Signal Activity")
		fmt.Println("───────────────────────────────────────")
		if cfg.SupabaseURL != "" {
			events, err := supaClient.GetPendingNotifications(cfg.GitHubUsername)
			if err != nil {
				fmt.Printf("  (Could not fetch activity: %v)\n", err)
			} else {
				weekEvents := 0
				for _, e := range events {
					if time.Since(e.CreatedAt) < 7*24*time.Hour {
						fmt.Printf("  @%s %s signal #%d\n", e.ActorUsername, e.EventType, e.SignalID)
						weekEvents++
					}
				}
				if weekEvents == 0 {
					fmt.Println("  No activity on your signals this week.")
				} else {
					fmt.Printf("  Total: %d events this week\n", weekEvents)
				}
			}
		} else {
			fmt.Println("  (Supabase not configured)")
		}
		fmt.Println()

		// 3. Stats
		fmt.Println("📊 Your Stats")
		fmt.Println("───────────────────────────────────────")
		user, _ := ghClient.GetCurrentUser()
		if user != nil {
			var connCount, successCount int
			if cfg.SupabaseURL != "" {
				connCount, successCount, _ = supaClient.GetUserStats(cfg.GitHubUsername)
			}

			// Count user's own signals
			ownCount := 0
			for _, s := range allSignals {
				if s.Author.GitHubUsername == cfg.GitHubUsername {
					ownCount++
				}
			}

			fmt.Printf("  Your signals:  %d\n", ownCount)
			fmt.Printf("  Connections:   %d\n", connCount)
			fmt.Printf("  Successes:     %d\n", successCount)
		}
		fmt.Println()
		fmt.Println("───────────────────────────────────────")
		fmt.Println("  Run 'hi' to open the full TUI feed.")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(digestCmd)
}

// Ensure models import is used (referenced in events loop)
var _ = models.ConnectionEvent{}

