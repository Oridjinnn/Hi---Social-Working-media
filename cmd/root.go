package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/github"
	"github.com/Oridjinnn/hi/supabase"
	"github.com/Oridjinnn/hi/tui"
)

var (
	cfg     *config.Config
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "hi",
	Short: "HI — Connect developers via structured intent signals",
	Long: `HI is a terminal-native CLI tool that connects developers via 
structured intent signals. Backed by GitHub Issues and Supabase.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		report := config.ValidateAuth(cfg)
		switch report.Level {
		case config.AuthMissing, config.AuthInvalid:
			return runAuthGate(report)
		case config.AuthDegraded:
			fmt.Fprintln(os.Stderr, "Warning: credential storage is not fully hardened.")
			for _, issue := range report.Issues {
				fmt.Fprintf(os.Stderr, "  • %s\n", issue)
			}
			if hints := report.FormatAuthHints(); hints != "" {
				fmt.Fprintln(os.Stderr, hints)
			}
		}
		return runTUI()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/hi/config.json)")
}

func initConfig() {
	if cfgFile != "" {
		return
	}
}

func runAuthGate(report config.AuthReport) error {
	p := tea.NewProgram(tui.NewAuthGateModel(report), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return config.SanitizedError(err)
	}
	return nil
}

func runTUI() error {
	// Build dependencies
	ghClient := github.New(cfg)
	supaClient := supabase.New(cfg.SupabaseURL, cfg.SupabaseAnonKey)

	app := tui.NewAppModel(ghClient, supaClient, cfg)

	// Enable mouse support for clickable model selectors
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
