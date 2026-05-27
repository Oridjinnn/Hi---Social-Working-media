package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Oridjinnn/hi/config"
	"github.com/Oridjinnn/hi/grouphouse"
)

var (
	houseName string
	housePort int
	agentName string
	agentID   string
	asAgent   bool
	houseHost string
)

var grouphouseCmd = &cobra.Command{
	Use:   "grouphouse",
	Short: "Manage group houses — shared collaborative workspaces",
	Long: `Group houses are shared workspaces where multiple users and AI agents
can collaborate in real-time. The housemaster runs a server on their machine,
and agents/guests connect via WebSocket to write code, run commands, and chat.`,
}

var grouphouseStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a group house server (become the housemaster)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if houseName == "" {
			houseName = cfg.GitHubUsername + "-house"
			if houseName == "-house" {
				houseName = "my-house"
			}
		}

		configDir := config.ConfigDir()
		ws, err := grouphouse.NewWorkspace(configDir, houseName)
		if err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		fmt.Printf("🏠 Starting Group House '%s'...\n", houseName)
		fmt.Printf("   Workspace: %s\n", ws.Path)
		fmt.Printf("   Port:      %d\n", housePort)
		fmt.Println()
		fmt.Println("   Agents can connect with:")
		fmt.Printf("   hi grouphouse join --host ws://localhost:%d --name <agent-name>\n", housePort)
		fmt.Println()
		fmt.Println("   Press Ctrl+C to stop the house.")

		server := grouphouse.NewServer(houseName, housePort, ws)

		// Handle graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println("\n🛑 Shutting down group house...")
			server.Stop()
			os.Exit(0)
		}()

		return server.Start()
	},
}

var grouphouseJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join a group house as an agent or guest",
	RunE: func(cmd *cobra.Command, args []string) error {
		if agentName == "" {
			return fmt.Errorf("--name is required to join a group house")
		}
		if houseHost == "" {
			return fmt.Errorf("--host is required (e.g., ws://localhost:9753)")
		}

		kind := grouphouse.KindGuest
		if asAgent {
			kind = grouphouse.KindAgent
			if agentID == "" {
				agentID = agentName + "-" + fmt.Sprintf("%d", os.Getpid())
			}
		}

		client := grouphouse.NewClient(houseHost, agentName, kind, agentID)

		// Set up callbacks
		client.OnConnected = func() {
			fmt.Printf("✅ Connected to group house at %s as '%s' (%s)\n", houseHost, agentName, kind)
			fmt.Println("   Type messages to broadcast. Type /help for commands.")
			fmt.Println()
		}

		client.OnAgentJoined = func(info grouphouse.AgentInfo) {
			fmt.Printf("👋 %s (%s) joined the house\n", info.Name, info.Kind)
		}

		client.OnAgentLeft = func(name string) {
			fmt.Printf("👋 %s left the house\n", name)
		}

		client.OnMessage = func(msg grouphouse.Message) {
			switch msg.Type {
			case grouphouse.MsgBroadcast:
				var payload grouphouse.BroadcastPayload
				if data, _ := json.Marshal(msg.Payload); len(data) > 0 {
					json.Unmarshal(data, &payload)
				}
				fmt.Printf("[%s] %s\n", msg.Sender, payload.Text)
			case grouphouse.MsgRunResult:
				var payload grouphouse.RunResultPayload
				if data, _ := json.Marshal(msg.Payload); len(data) > 0 {
					json.Unmarshal(data, &payload)
				}
				fmt.Printf("⚡ %s ran: %s\n", msg.Sender, payload.Command)
				if payload.Stdout != "" {
					fmt.Println(payload.Stdout)
				}
				if payload.Stderr != "" {
					fmt.Printf("⚠️  %s\n", payload.Stderr)
				}
				fmt.Printf("   → exit code %d (%s)\n", payload.ExitCode, payload.Duration)
			case grouphouse.MsgFileChanged:
				var payload grouphouse.FileChangedPayload
				if data, _ := json.Marshal(msg.Payload); len(data) > 0 {
					json.Unmarshal(data, &payload)
				}
				fmt.Printf("📝 %s %s: %s\n", msg.Sender, payload.Action, payload.Path)
			case grouphouse.MsgAgentList:
				var payload grouphouse.AgentListPayload
				if data, _ := json.Marshal(msg.Payload); len(data) > 0 {
					json.Unmarshal(data, &payload)
				}
				fmt.Printf("🏠 House: %s\n", payload.HouseName)
				fmt.Printf("📁 Workspace: %s\n", payload.WorkspacePath)
				fmt.Printf("👥 Agents (%d):\n", len(payload.Agents))
				for _, a := range payload.Agents {
					fmt.Printf("   • %s (%s)\n", a.Name, a.Kind)
				}
			}
		}

		client.OnRunResult = func(result grouphouse.RunResultPayload) {
			fmt.Printf("⚡ Result: %s\n", result.Command)
			if result.Stdout != "" {
				fmt.Println(result.Stdout)
			}
			if result.Stderr != "" {
				fmt.Printf("⚠️  %s\n", result.Stderr)
			}
			fmt.Printf("   → exit %d (%s)\n", result.ExitCode, result.Duration)
		}

		client.OnError = func(err grouphouse.ErrorPayload) {
			fmt.Printf("❌ Error: %s\n", err.Message)
		}

		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		// Interactive loop
		scanner := make(chan string, 1)
		go func() {
			var buf [1024]byte
			for {
				n, _ := os.Stdin.Read(buf[:])
				if n <= 0 {
					break
				}
				scanner <- string(buf[:n-1]) // remove newline
			}
		}()

		for {
			select {
			case line := <-scanner:
				if line == "" {
					continue
				}
				if err := handleClientCommand(client, line); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}
		}
	},
}

func handleClientCommand(client *grouphouse.Client, line string) error {
	switch {
	case line == "/quit" || line == "/exit":
		client.Close()
		os.Exit(0)
		return nil

	case line == "/help":
		fmt.Println("Commands:")
		fmt.Println("  /help             Show this help")
		fmt.Println("  /quit             Disconnect")
		fmt.Println("  /agents           List connected agents")
		fmt.Println("  /tree             List workspace files")
		fmt.Println("  /run <cmd>        Run a command in the workspace")
		fmt.Println("  /write <path>     Write file content (then paste text, Ctrl+D to end)")
		fmt.Println("  /dm <name> <msg>  Send a private message")
		fmt.Println("  anything else     Broadcast to all participants")
		return nil

	case line == "/agents":
		return client.ListAgents()

	case line == "/tree":
		return client.RequestWorkspaceTree()

	case line == "/dm":
		fmt.Println("Usage: /dm <agent-name> <message>")
		return nil

	default:
		return client.Broadcast(line)
	}
}

var grouphouseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active group houses on this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDir()
		housesDir := filepath.Join(configDir, "houses")

		entries, err := os.ReadDir(housesDir)
		if err != nil {
			fmt.Println("No group houses found.")
			return nil
		}

		fmt.Printf("Group Houses (%d):\n", len(entries))
		for _, e := range entries {
			if e.IsDir() {
				fmt.Printf("  🏠 %s\n", e.Name())
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(grouphouseCmd)
	grouphouseCmd.AddCommand(grouphouseStartCmd)
	grouphouseCmd.AddCommand(grouphouseJoinCmd)
	grouphouseCmd.AddCommand(grouphouseListCmd)

	grouphouseStartCmd.Flags().StringVar(&houseName, "name", "", "House name (default: <username>-house)")
	grouphouseStartCmd.Flags().IntVar(&housePort, "port", 9753, "WebSocket server port")

	grouphouseJoinCmd.Flags().StringVar(&agentName, "name", "", "Your name in the house")
	grouphouseJoinCmd.Flags().StringVar(&houseHost, "host", "", "Server URL (e.g., ws://localhost:9753)")
	grouphouseJoinCmd.Flags().BoolVar(&asAgent, "agent", false, "Connect as an AI agent")
	grouphouseJoinCmd.Flags().StringVar(&agentID, "agent-id", "", "Unique agent identifier (auto-generated if empty)")
}