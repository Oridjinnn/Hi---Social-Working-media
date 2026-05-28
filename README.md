# 🏠 HI — Social Working Media

frien> **Terminal-native collaboration for developers. Signals, not noise.**

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.20-00ADD8?logo=go)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)](CONTRIBUTING.md)

HI is a **terminal-first collaboration tool** built for modern software teams and knowledge workers. It replaces noisy Slack/Discord chatter with **structured intent signals** — short, actionable requests that connect you with the right people at the right time. All from your terminal.

- **Browse signals** — See who needs help, who's hiring, who's collaborating
- **Chat P2P** — Lightweight messaging backed by GitHub Issue comments
- **Collaborate live** — GroupHouse workspaces with file sharing, agent protocol, and shared context
- **Stay ahead** — Market Intel surfaces trends, opportunities, and tech signals
- **Never lose context** — Local Rewind remembers everything you've seen

---

## ✨ Features

| Feature | Description |
|---|---|
| **Signal Feed** | Browse structured collaboration intent signals (looking for contributors, hiring, showcasing, etc.) |
| **P2P Chat** | Direct messaging via GitHub Issue comments with 15s polling for near-realtime conversation |
| **GroupHouse** | Collaborative workspace with multi-agent protocol — file reads/writes, command execution, broadcasts, DM |
| **Market Intel** | Trend analysis powered by GitHub Search + HN Algolia API with sparkline charts and HTML export |
| **Trust Ranking** | Signal quality scoring based on freshness, profile completeness, and response rate |
| **Local Rewind** | Review history of visited signals, chat sessions, and GroupHouse events — press `h` from any tab |
| **Realtime Sync** | Supabase-backed event bus with exponential backoff reconnection, state machine, and connection status indicator |
| **Auth & Security** | GitHub OAuth with secure credential storage, startup validation, and sanitized error handling |
| **Ghost Signals** | Browse public signals from anonymous/guest users alongside authenticated content |
| **Notifications** | Desktop notifications for new signals, messages, and workspace events |

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.20+** (for building from source)
- **Git**
- **A GitHub account** (for OAuth and the signal repository)

### One-liner Install

```bash
curl -sfL https://raw.githubusercontent.com/Oridjinnn/Hi---Social-Working-media/main/hi/install.sh | sh
```

### Manual Build

```bash
git clone https://github.com/Oridjinnn/Hi---Social-Working-media.git
cd Hi---Social-Working-media/hi
go build -o hi .
./hi auth
```

The `auth` command launches the GitHub OAuth flow and configures HI for backend access.

> **First time?** Run `./hi auth` — the TUI will walk you through setup and credential configuration.

---

## 🎮 Usage

### CLI Commands

| Command | Description |
|---|---|
| `hi auth` | GitHub OAuth authentication flow |
| `hi search [query]` | Search signals by keyword |
| `hi connect <signal-id>` | Connect with a signal author |
| `hi profile` | View and edit your profile |
| `hi digest` | Generate a digest of recent activity |
| `hi signal` | Post a new collaboration signal |
| `hi grouphouse` | Manage GroupHouse workspaces |

### TUI Key Bindings

| Key | Action |
|---|---|
| `Tab` / `1` `2` `3` | Switch between Home / Feed / GroupHouse tabs |
| `↑` / `↓` | Navigate signal list |
| `→` / `Enter` | Open signal detail view |
| `c` | Start a chat with signal author (from detail view) |
| `m` | Open Market Intel panel |
| `h` | Open Rewind / history for current tab |
| `e` | Export market chart as HTML (from Market view) |
| `?` | Show command palette / help (planned) |
| `q` / `Esc` | Quit / go back |

### Environment Variables

| Variable | Description | Required |
|---|---|---|
| `HI_GITHUB_CLIENT_ID` | GitHub OAuth App client ID | Yes (or build-time ldflags) |
| `HI_SUPABASE_URL` | Supabase project URL | Yes (or build-time ldflags) |
| `HI_SUPABASE_ANON_KEY` | Supabase anonymous key | Yes (or build-time ldflags) |
| `HI_SIGNAL_REPO_OWNER` | GitHub owner for the signal repository | Yes (or build-time ldflags) |
| `HI_SIGNAL_REPO_NAME` | GitHub repository name for signals | Yes (or build-time ldflags) |
| `HI_CACHE_TTL` | Cache TTL for API responses (default: 5m) | No |

### Configuration

Configuration is stored at `~/.config/hi/config.json`. Connection limits and chat configurations reside in `~/.config/hi/connections.json`.

---

## 🏗 Architecture

```
hi/
├── main.go               # Entry point, build-time variable injection
├── cmd/                   # CLI commands (cobra-based)
│   ├── root.go           # TUI launcher + auth gate
│   ├── auth.go           # GitHub OAuth flow
│   ├── connect.go        # Signal connection flow
│   ├── search.go         # Signal search
│   ├── profile.go        # User profile management
│   ├── signal.go         # Signal posting
│   ├── grouphouse.go     # Workspace management
│   ├── digest.go         # Activity digest
│   └── secret.go         # Unlisted
├── tui/                   # Terminal UI (Bubble Tea + Lip Gloss)
│   ├── app.go            # Root model with tab navigation
│   ├── feed.go           # Signal feed list
│   ├── detail.go         # Signal detail view
│   ├── chat.go           # P2P chat view
│   ├── grouphouse.go     # Workspace UI
│   ├── market.go         # Market Intel with sparkline charts
│   ├── trend.go          # Trend panel (render above feed)
│   ├── auth.go           # Auth gate wizard
│   ├── profile.go        # Profile editing UI
│   ├── wizard.go         # First-run onboarding flow
│   ├── realtime_bridge.go# Supabase realtime connection state machine
│   ├── sync.go           # State sync across tabs
│   ├── trust.go          # Trust/ranking indicators
│   ├── toast.go          # Toast notification system
│   └── styles.go         # Shared styling constants
├── github/                # GitHub API integration
│   ├── client.go         # HTTP client + auth
│   ├── issues.go         # Issue CRUD (signal backing store)
│   ├── chat.go           # P2P chat via issue comments
│   ├── search.go         # Search API + ghost signal fetching
│   ├── user.go           # User profile API
│   ├── repo.go           # Repository metadata
│   └── signals.go        # Signal-specific API orchestration
├── grouphouse/            # Collaborative workspace protocol
│   ├── protocol.go       # Message types, envelope, serialization
│   ├── server.go         # Workspace server (WebSocket-based)
│   ├── client.go         # Workspace client
│   ├── workspace.go      # Workspace state management
│   └── cmd.go            # Workspace command execution
├── models/                # Domain types
│   ├── signal.go         # SignalType, CommitmentLevel, DifficultyLevel
│   ├── user.go           # User profile model
│   ├── event.go          # Domain events
│   ├── repo.go           # Repository model
│   ├── rank.go           # Trust/quality scoring
│   └── rank_test.go      # Rank tests
├── market/                # Market analytics
│   └── intel.go          # Trend data fetching (GitHub Search + HN Algolia)
├── history/               # Local persistence
│   ├── history.go        # Rewind/history store
│   └── history_test.go   # History tests
├── config/                # Configuration management
│   ├── config.go         # Config loading/storage
│   └── security.go       # Auth validation, secure storage, sanitization
├── supabase/              # Supabase backend client
│   ├── client.go         # REST client
│   ├── events.go         # Event publishing
│   └── realtime.go       # WebSocket realtime subscription
├── notify/                # Desktop notifications
│   └── notify.go         # Cross-platform notification dispatch
├── utils/                 # Utility modules
│   ├── browser.go        # Browser launcher
│   ├── cache.go          # TTL-based response cache
│   ├── clipboard.go      # Clipboard access
│   └── format.go         # Text formatting helpers
├── scripts/               # Build scripts
│   ├── build.sh          # Linux/macOS cross-compile
│   └── build.ps1         # Windows build
└── bin/                   # Compiled binaries (after build)
```

### Data Flow

```
GitHub Issues (signal store)
      ↑↓
  github/ client ──► tui/ feed/detail
      │
      ├──► supabase/ events (realtime broadcast)
      │         │
      │         └──► tui/ realtime_bridge ──► tui/ chat / toast
      │
      └──► grouphouse/ workspace protocol (WebSocket P2P)
                │
                └──► tui/ grouphouse (collaborative session UI)
```

---

## 🔧 Building

### Linux / macOS

```bash
# Build all targets (linux, darwin, windows) for amd64
./scripts/build.sh all amd64

# Build only for macOS (darwin) arm64
./scripts/build.sh darwin arm64

# Build only for current host
./scripts/build.sh
```

### Windows (PowerShell)

```powershell
.\scripts\build.ps1 -Target windows -Arch amd64
.\scripts\build.ps1 -Target all -Arch amd64
```

### Build-time Variables

You can embed credentials at build time using `-ldflags`:

```bash
go build -ldflags="\
  -X main.GitHubClientID=your_client_id \
  -X main.SupabaseURL=your_supabase_url \
  -X main.SupabaseAnonKey=your_anon_key \
  -X main.SignalRepoOwner=your_org \
  -X main.SignalRepoName=signals \
  -o hi .
```

These are automatically wired into environment variables (`HI_GITHUB_CLIENT_ID`, etc.) at runtime unless already set.

---

## 🧪 Development

```bash
# Run all tests
go test ./...

# Format code
gofmt -w .

# Lint
golangci-lint run
```

**CI Pipeline:** `gofmt` → `golangci-lint` → `go build` → `go test` runs on every push and PR to `main`.

---

## 📸 Screenshots

> *Coming soon — TUI screenshots and asciicast demos.*
> 
> Run `./hi` and press `Tab` to explore the Feed, GroupHouse, and Market tabs.

---

## 🤝 Contributing

1. **Fork** the repository
2. **Create a branch:** `git checkout -b feature/your-feature`
3. **Commit** with clear, descriptive messages
4. **Open a pull request**

See [TODO.md](TODO.md) for the current development roadmap and priorities.

### Code Style

- Format with `gofmt` before committing
- Run `golangci-lint` and address warnings
- All new code should include tests

---

## 🗺 Roadmap

| Priority | Area | Status |
|---|---|---|
| P0 | Realtime reliability, state consistency, auth hygiene, trust ranking | ✅ Complete |
| P1 | Onboarding wizard, command discoverability, match quality | 🔜 In progress |
| P2 | Funnel analytics, market intel depth, ecosystem extensibility | 📋 Planned |

See [TODO.md](TODO.md) for the full implementation backlog and acceptance criteria.

---

## 📄 License

This project is open source under the MIT License. See [LICENSE](LICENSE) for details.

---

<p align="center">
  Made for developers who build in the terminal.<br>
  <strong>HI</strong> — Signals, not noise.
</p>