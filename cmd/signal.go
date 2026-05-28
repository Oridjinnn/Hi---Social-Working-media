package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/supabase"
)

var (
	signalType       string
	signalProject    string
	signalStack      string
	signalNeeds      string
	signalDifficulty string
	signalCommitment string
	signalContact    string
	signalContactURL string
	signalStatus     string
	signalMine       bool
	signalReason     string
)

var signalCmd = &cobra.Command{
	Use:   "signal",
	Short: "Manage signals",
}

var signalCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new signal",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not authenticated. Run: hi auth login")
		}

		// Check if any relevant flags were provided (non-interactive mode)
		relevant := []string{"type", "project", "stack", "needs", "difficulty", "commitment", "contact", "contact-url"}
		hasFlags := false
		for _, f := range relevant {
			if cmd.Flags().Changed(f) {
				hasFlags = true
				break
			}
		}

		if !hasFlags {
			// Launch TUI wizard
			fmt.Println("Launching signal creation wizard... (requires TUI)")
			return nil
		}

		// Non-interactive mode from flags
		client := github.New(cfg)

		// Parse stack and needs from comma-separated strings
		stack := []string{}
		if signalStack != "" {
			for _, s := range strings.Split(signalStack, ",") {
				stack = append(stack, strings.TrimSpace(s))
			}
		}
		needs := []string{}
		if signalNeeds != "" {
			for _, n := range strings.Split(signalNeeds, ",") {
				needs = append(needs, strings.TrimSpace(n))
			}
		}

		s := models.Signal{
			Title:      fmt.Sprintf("[%s] %s", signalType, signalProject),
			Body:       "",
			Type:       models.SignalType(signalType),
			Status:     models.SignalStatusOpen,
			Project:    signalProject,
			Stack:      stack,
			Needs:      needs,
			Difficulty: models.DifficultyLevel(signalDifficulty),
			Commitment: models.CommitmentLevel(signalCommitment),
			Contact:    models.ContactMethod(signalContact),
			ContactURL: signalContactURL,
		}

		if err := client.CheckSignalLimit(cfg.GitHubUsername, cfg.SignalLimit(), cfg.GetTier()); err != nil {
			return err
		}

		created, err := client.CreateSignal(&s)
		if err != nil {
			return fmt.Errorf("creating signal: %w", err)
		}

		fmt.Printf("✓ Signal created: #%d %s\n", created.ID, created.GitHubURL)
		return nil
	},
}

var signalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List signals",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not authenticated. Run: hi auth login")
		}

		client := github.New(cfg)

		var labels []string
		if signalStatus != "" {
			labels = append(labels, "status:"+signalStatus)
		}
		if signalMine {
			labels = append(labels, fmt.Sprintf("author:%s", cfg.GitHubUsername))
		}

		signals, err := client.ListSignals(labels, 1)
		if err != nil {
			return fmt.Errorf("listing signals: %w", err)
		}

		if len(signals) == 0 {
			fmt.Println("No signals found.")
			return nil
		}

		for _, s := range signals {
			stackStr := strings.Join(s.Stack, ", ")
			fmt.Printf("#%d [%s] %s by @%s (stack: %s)\n",
				s.ID, s.Status, s.Title, s.Author.GitHubUsername, stackStr)
		}
		return nil
	},
}

var signalCloseCmd = &cobra.Command{
	Use:   "close <signal-id>",
	Short: "Close a signal",
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

		reason := models.SignalStatus(signalReason)
		if reason == "" {
			reason = models.SignalStatusFilled
		}

		client := github.New(cfg)
		if err := client.CloseSignal(issueNumber, reason); err != nil {
			return fmt.Errorf("closing signal: %w", err)
		}

		fmt.Printf("✓ Signal #%d closed (%s)\n", issueNumber, reason)
		return nil
	},
}

var signalUpdateCmd = &cobra.Command{
	Use:   "update <signal-id>",
	Short: "Update a signal",
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

		client := github.New(cfg)

		// Get current signal
		signal, err := client.GetSignal(issueNumber)
		if err != nil {
			return fmt.Errorf("fetching signal: %w", err)
		}

		if signalStatus != "" {
			newStatus := models.SignalStatus(signalStatus)
			if err := client.UpdateSignalStatus(issueNumber, newStatus); err != nil {
				return fmt.Errorf("updating status: %w", err)
			}
			fmt.Printf("✓ Signal #%d status updated to %s\n", issueNumber, newStatus)
		}

		// If paused, prompt to re-open
		if signal.Status == models.SignalStatusPaused && signalStatus == string(models.SignalStatusOpen) {
			fmt.Println("Signal re-opened.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(signalCmd)
	signalCmd.AddCommand(signalCreateCmd)
	signalCmd.AddCommand(signalListCmd)
	signalCmd.AddCommand(signalCloseCmd)
	signalCmd.AddCommand(signalUpdateCmd)

	signalCreateCmd.Flags().StringVar(&signalType, "type", "contributor", "Signal type (contributor, beginner, vibe_coder, hiring, showcase)")
	signalCreateCmd.Flags().StringVar(&signalProject, "project", "", "Project name")
	signalCreateCmd.Flags().StringVar(&signalStack, "stack", "", "Stack tags (comma separated)")
	signalCreateCmd.Flags().StringVar(&signalNeeds, "needs", "", "What you need (comma separated)")
	signalCreateCmd.Flags().StringVar(&signalDifficulty, "difficulty", "intermediate", "Difficulty level")
	signalCreateCmd.Flags().StringVar(&signalCommitment, "commitment", "casual", "Commitment level")
	signalCreateCmd.Flags().StringVar(&signalContact, "contact", "github", "Contact method")
	signalCreateCmd.Flags().StringVar(&signalContactURL, "contact-url", "", "Contact URL")

	signalListCmd.Flags().BoolVar(&signalMine, "mine", false, "Only show your signals")
	signalListCmd.Flags().StringVar(&signalStatus, "status", "", "Filter by status")

	signalCloseCmd.Flags().StringVar(&signalReason, "reason", "filled", "Reason (filled, expired, paused)")

	signalUpdateCmd.Flags().StringVar(&signalStatus, "status", "", "New status (open, in-progress, filled, paused)")
}

func parseIssueNumber(s string) (int64, error) {
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimSpace(s)
	var num int64
	_, _ = fmt.Sscanf(s, "%d", &num)
	if num == 0 {
		return 0, fmt.Errorf("could not parse issue number from: %s", s)
	}
	return num, nil
}

// Ensure imports are used
var _ = supabase.Client{}
