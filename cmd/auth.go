package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/tui"
	"github.com/Oridjinnn/hi/utils"
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

type githubUserResponse struct {
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
	Bio       string `json:"bio"`
}

var forceReauth bool

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with GitHub via OAuth device flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		return loginGitHub()
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = os.Remove(config.TokenPath())
		_ = os.Remove(config.ConfigPath())
		fmt.Println("Logged out.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showAuthStatus()
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authLoginCmd.Flags().BoolVarP(&forceReauth, "force", "f", false, "Force re-authentication")
}

func showAuthStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return config.SanitizedError(fmt.Errorf("loading config: %w", err))
	}

	report := config.ValidateAuth(cfg)
	switch report.Level {
	case config.AuthOK:
		fmt.Printf("Authenticated as @%s\n", cfg.GitHubUsername)
		fmt.Printf("Config: %s\n", config.ConfigPath())
		fmt.Printf("Token file: %s\n", config.TokenPath())
		fmt.Println("Storage: hardened")
	case config.AuthDegraded:
		fmt.Printf("Authenticated as @%s (degraded)\n", cfg.GitHubUsername)
		for _, issue := range report.Issues {
			fmt.Printf("  • %s\n", issue)
		}
		if hints := report.FormatAuthHints(); hints != "" {
			fmt.Println(hints)
		}
	default:
		fmt.Println("Not authenticated.")
		for _, issue := range report.Issues {
			fmt.Printf("  • %s\n", issue)
		}
		if hints := report.FormatAuthHints(); hints != "" {
			fmt.Println(hints)
		}
	}
	return nil
}

func loginGitHub() error {
	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	// If already authenticated and not forced, show already-authed view
	if !forceReauth && cfg.GitHubUsername != "" && cfg.GitHubToken != "" {
		p := tea.NewProgram(tui.NewAlreadyAuthedModel(cfg.GitHubUsername))
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	}

	// Get client ID — from config, env var, or prompt
	clientID := cfg.GitHubClientID
	if clientID == "" {
		clientID = os.Getenv("HI_GITHUB_CLIENT_ID")
	}
	if clientID == "" {
		// Interactive prompt via TUI
		clientID, err = promptForClientID()
		if err != nil {
			return err
		}
		// Save to config
		if err := config.SaveClientID(clientID); err != nil {
			return fmt.Errorf("saving client ID: %w", err)
		}
	}

	if strings.Contains(clientID, "PLACEHOLDER") {
		return fmt.Errorf("client ID looks like a placeholder — create a real OAuth App at https://github.com/settings/developers")
	}

	// Step 1: Request device code
	deviceReq, err := http.NewRequest("POST", "https://github.com/login/device/code",
		strings.NewReader(url.Values{
			"client_id": []string{clientID},
			"scope":     []string{"public_repo read:user"},
		}.Encode()))
	if err != nil {
		return fmt.Errorf("creating device code request: %w", err)
	}
	deviceReq.Header.Set("Accept", "application/json")
	deviceReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(deviceReq)
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var deviceCode deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCode); err != nil {
		return fmt.Errorf("decoding device code response: %w", err)
	}

	// ── Browser confirmation prompt ──────────────────────────────────────
	//
	// Print the prompt directly to the terminal (before TUI starts) so it
	// is guaranteed to be visible on any platform, including Windows.
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "╭──────────────────────────────────────────────────────────╮")
	fmt.Fprintln(os.Stderr, "│                                                          │")
	fmt.Fprintln(os.Stderr, "│   🔑 GitHub Authentication                               │")
	fmt.Fprintln(os.Stderr, "│                                                          │")
	fmt.Fprintln(os.Stderr, "│   HI needs to open your browser and copy a code to       │")
	fmt.Fprintln(os.Stderr, "│   your clipboard to authenticate with GitHub.            │")
	fmt.Fprintln(os.Stderr, "│                                                          │")
	fmt.Fprintln(os.Stderr, "│   Allow browser + clipboard access?                      │")
	fmt.Fprintln(os.Stderr, "│                                                          │")
	fmt.Fprintln(os.Stderr, "│   • y / yes → open browser + copy code automatically     │")
	fmt.Fprintln(os.Stderr, "│   • n / no  → show URL + code, you open manually         │")
	fmt.Fprintln(os.Stderr, "│   • s / skip → skip this prompt next time (save pref)    │")
	fmt.Fprintln(os.Stderr, "│                                                          │")
	fmt.Fprint(os.Stderr, "   Choose (y/n/s): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanned := scanner.Scan()
	input := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if !scanned || input == "n" || input == "no" {
		// User chose manual mode — skip browser/clipboard, print URL+code
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  ─────────────────────────────────────────────────")
		fmt.Fprintln(os.Stderr, "  Manual authentication — follow these steps:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "  1. Open:  \033[36m%s\033[0m\n", deviceCode.VerificationURI)
		fmt.Fprintf(os.Stderr, "  2. Code:  \033[33m%s\033[0m\n", deviceCode.UserCode)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  ─────────────────────────────────────────────────")
		fmt.Fprintln(os.Stderr)
	} else if input == "s" || input == "skip" {
		// Save preference to allow browser access
		_ = config.SaveClientID(clientID) // best-effort
		_ = utils.CopyToClipboard(deviceCode.UserCode)
		_ = utils.OpenURL(deviceCode.VerificationURI)
	} else {
		// User allowed browser + clipboard access
		_ = utils.CopyToClipboard(deviceCode.UserCode)
		_ = utils.OpenURL(deviceCode.VerificationURI)
	}

	// Step 2: Poll for access token via TUI
	token, err := pollForToken(clientID, deviceCode)
	if err != nil {
		return err
	}

	// Step 3: Save token
	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	// Step 4: Fetch user info
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hi-cli/1.0")

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}
	defer func() {
		_ = resp2.Body.Close()
	}()

	var ghUser githubUserResponse
	if err := json.NewDecoder(resp2.Body).Decode(&ghUser); err != nil {
		return fmt.Errorf("decoding user response: %w", err)
	}

	// Step 5: Save config (preserve existing client ID if set)
	if cfg.GitHubClientID == "" {
		cfg.GitHubClientID = clientID
	}
	cfg.GitHubToken = token
	cfg.GitHubUsername = ghUser.Login
	cfg.HIRepoOwner = os.Getenv("HI_SIGNAL_REPO_OWNER")
	cfg.HIRepoName = os.Getenv("HI_SIGNAL_REPO_NAME")
	cfg.SupabaseURL = os.Getenv("HI_SUPABASE_URL")
	cfg.SupabaseAnonKey = os.Getenv("HI_SUPABASE_ANON_KEY")

	if cfg.HIRepoOwner == "" {
		cfg.HIRepoOwner = ghUser.Login
	}
	if cfg.HIRepoName == "" {
		cfg.HIRepoName = "hi-signals"
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Show success via TUI
	p := tea.NewProgram(tui.NewAuthModel(deviceCode.UserCode, deviceCode.VerificationURI))
	p.Send(tui.AuthSuccessMsg{Username: ghUser.Login})
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func promptForClientID() (string, error) {
	p := tea.NewProgram(tui.NewClientIDPromptModel())
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	promptModel, ok := model.(tui.ClientIDPromptModel)
	if !ok {
		return "", fmt.Errorf("unexpected error")
	}

	result := promptModel.Result()
	if result.Err != nil {
		return "", result.Err
	}

	clientID := strings.TrimSpace(result.ClientID)
	if clientID == "" {
		return "", fmt.Errorf("client ID cannot be empty")
	}

	return clientID, nil
}

func pollForToken(clientID string, deviceCode deviceCodeResponse) (string, error) {
	interval := deviceCode.Interval
	if interval < 5 {
		interval = 5
	}

	// Start TUI
	p := tea.NewProgram(tui.NewAuthModel(deviceCode.UserCode, deviceCode.VerificationURI))

	// resultCh: poll goroutine writes here exactly once.
	// notifyCh: relay goroutine reads from here to know when to send to TUI.
	// Using two channels avoids the "consumed but not written back" race.
	type pollResult struct {
		token string
		err   error
	}
	resultCh := make(chan pollResult, 1)
	notifyCh := make(chan pollResult, 1)

	// Poll in background — writes to BOTH channels so each reader gets a copy.
	go func() {
		var result pollResult

		for i := 0; i < deviceCode.ExpiresIn/interval; i++ {
			// On subsequent iterations, wait before polling
			if i > 0 {
				time.Sleep(time.Duration(interval) * time.Second)
			}

			tokenReq, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token",
				strings.NewReader(url.Values{
					"client_id":   []string{clientID},
					"device_code": []string{deviceCode.DeviceCode},
					"grant_type":  []string{"urn:ietf:params:oauth:grant-type:device_code"},
				}.Encode()))
			if err != nil {
				result.err = fmt.Errorf("creating token request: %w", err)
				resultCh <- result
				notifyCh <- result
				return
			}
			tokenReq.Header.Set("Accept", "application/json")
			tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := http.DefaultClient.Do(tokenReq)
			if err != nil {
				result.err = fmt.Errorf("polling for token: %w", err)
				resultCh <- result
				notifyCh <- result
				return
			}

			var tokenResp accessTokenResponse
			if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
				resp.Body.Close()
				result.err = fmt.Errorf("decoding token response: %w", err)
				resultCh <- result
				notifyCh <- result
				return
			}
			resp.Body.Close()

			if tokenResp.AccessToken != "" {
				result.token = tokenResp.AccessToken
				resultCh <- result
				notifyCh <- result
				return
			}

			if tokenResp.Error == "authorization_pending" {
				continue
			}
			if tokenResp.Error == "slow_down" {
				interval += 5
				continue
			}
			if tokenResp.Error != "" {
				result.err = fmt.Errorf("auth error: %s", tokenResp.Error)
				resultCh <- result
				notifyCh <- result
				return
			}
		}

		result.err = fmt.Errorf("authentication timed out")
		resultCh <- result
		notifyCh <- result
	}()

	// Relay poll result to the TUI — reads from notifyCh.
	// When the TUI receives the message it quits and p.Run() returns.
	go func() {
		result := <-notifyCh
		if result.token != "" {
			p.Send(tui.AuthSuccessMsg{Username: "fetching..."})
		} else if result.err != nil {
			p.Send(tui.AuthErrMsg{Err: result.err})
		}
	}()

	if _, err := p.Run(); err != nil {
		return "", err
	}

	// Read final result from resultCh — guaranteed to have data
	// because the poll goroutine writes to it before or at the same
	// time as notifyCh, and resultCh has buffer size 1.
	select {
	case result := <-resultCh:
		if result.err != nil {
			return "", result.err
		}
		return result.token, nil
	case <-time.After(1 * time.Second):
		// Safety timeout — user quit TUI before poll finished.
		return "", fmt.Errorf("login cancelled")
	}
}
