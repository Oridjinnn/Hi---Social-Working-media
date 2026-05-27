package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/supabase"
)

var profileCmd = &cobra.Command{
	Use:   "profile [username]",
	Short: "View a user's profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not authenticated. Run: hi auth login")
		}

		client := github.New(cfg)
		supaClient := supabase.New(cfg.SupabaseURL, cfg.SupabaseAnonKey)

		username := cfg.GitHubUsername
		if len(args) > 0 {
			username = strings.TrimPrefix(args[0], "@")
		}

		user, err := client.GetUser(username)
		if err != nil {
			return fmt.Errorf("fetching user: %w", err)
		}

		// HI stats
		if cfg.SupabaseURL != "" {
			connCount, successCount, err := supaClient.GetUserStats(username)
			if err == nil {
				user.ConnectionCount = connCount
				user.SuccessCount = successCount
			}
		}

		isOwn := username == cfg.GitHubUsername
		user.IsSeedUser = false // seed logic not implemented in MVP
		user.IsSupporter = false

		// Recent signals: use GitHub issue list and filter by author.
		// MVP fetches only a single page.
		signals, err := client.ListSignals(nil, 1)
		if err == nil {
			count := 0
			for _, s := range signals {
				if s.Author.GitHubUsername == username {
					count++
				}
			}
			user.SignalCount = count
		}

		fmt.Println()
		fmt.Printf("  @%s", user.GitHubUsername)
		if isOwn {
			fmt.Print(" (you)")
		}
		fmt.Println()

		if user.Bio != "" {
			fmt.Printf("  %s\n", user.Bio)
		}
		fmt.Println()

		fmt.Printf("  📊 %d signals  🤝 %d connections  ✅ %d successes\n",
			user.SignalCount, user.ConnectionCount, user.SuccessCount)
		fmt.Printf("  👥 %d followers  📦 %d public repos\n",
			user.Followers, user.PublicRepos)
		fmt.Printf("  🔗 %s\n", user.GitHubURL)
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
}
