package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search signals by tag or text",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not authenticated. Run: hi auth login")
		}

		client := github.New(cfg)
		query := strings.Join(args, " ")

		// Parse query into label filters + text
		var labelFilters []string
		var textParts []string

		knownPrefixes := []string{"type:", "status:", "commitment:", "difficulty:", "stack:", "need:"}
		isLabel := false

		for _, word := range strings.Fields(query) {
			isLabel = false
			for _, prefix := range knownPrefixes {
				if strings.HasPrefix(word, prefix) {
					labelFilters = append(labelFilters, word)
					isLabel = true
					break
				}
			}
			if !isLabel {
				textParts = append(textParts, word)
			}
		}

		textQuery := strings.Join(textParts, " ")
		if textQuery == query {
			// No label filters, try to interpret as a direct search
			labelFilters = nil
		}

		signals, err := client.SearchSignals(textQuery, labelFilters)
		if err != nil {
			return fmt.Errorf("searching signals: %w", err)
		}

		if len(signals) == 0 {
			fmt.Println("No signals found matching your search.")
			return nil
		}

		fmt.Printf("Found %d signals:\n\n", len(signals))
		for _, s := range signals {
			stackStr := strings.Join(s.Stack, ", ")
			fmt.Printf("#%d [%s] [%s] %s\n", s.ID, s.Type, s.Status, s.Title)
			fmt.Printf("  by @%s | %s | %s\n", s.Author.GitHubUsername, stackStr, s.Commitment)
			fmt.Printf("  %s\n", s.GitHubURL)
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
