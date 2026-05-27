# Hi — Social Working Media

Hi is a lightweight social collaboration CLI application designed for modern knowledge workers. It combines chat, signals, and workspace tracking into a terminal-first experience focused on fast communication and contextual awareness.

## Key Features

- Terminal-native interface with fast access to conversations and activity signals
- Local rewind/history view for visited signals, chat sessions, and grouphouse events
- Integration with a backend signal service for issues, comments, and notifications
- Support for authentication and session-based workflows
- Modular codebase with clear package boundaries for CLI, backend, and UI layers

## Project Structure

- `cmd/` — command definitions and CLI entrypoints
- `github/` — GitHub integration and API client logic
- `grouphouse/` — collaborative workspace and protocol implementations
- `tui/` — terminal UI components and application flow
- `models/` — core domain models for events, signals, and users
- `config/` — application configuration and security handling
- `supabase/` — realtime and event integrations
- `utils/` — helper utilities for browser, clipboard, format, and cache

## Getting Started

### Prerequisites

- Go 1.22+
README

Hi — Social Working Media
===================================

Hi is a terminal-first collaboration tool for software teams and knowledge workers. It surfaces signals (short, focused requests), lightweight chat, and shared workspace sessions, with a local Rewind feature that helps you rediscover recent activity.

Why Hi
------
- Fast, keyboard-driven TUI built with Bubble Tea and Lip Gloss
- Local Rewind/history for quick context (signals visited, chat sessions, grouphouse events)
- Extensible architecture: clear separation between CLI, TUI, backend integrations, and models

Quick Start
-----------
Prerequisites

- Go 1.20+ (recommended)
- Git

Build and run locally

```bash
cd /home/habel/Documents/HI/hi
go build -o hi .
./hi auth
```

The `auth` command begins the GitHub authentication flow and configures the client for interacting with the signal backend.

Using Rewind
------------
- Press `h` in the Feed tab to view recently opened signals and chat sessions.
- Press `h` in the GroupHouse tab to review recent workspace events.

Development
-----------
- Run the test suite: `go test ./...`
- Format code: `gofmt -w .`
- CI: GitHub Actions runs `gofmt`, `golangci-lint`, `go build`, and `go test` on push/PR to `main`.

Repository Layout
-----------------
- `cmd/` — CLI commands and entrypoints
- `tui/` — terminal UI components and application state
- `github/` — GitHub API client and integrations
- `grouphouse/` — collaborative workspace server/client code
- `models/` — domain models (signals, users, events)
- `history/` — local Rewind/history persistence
- `config/`, `supabase/`, `utils/`, `notify/` — ancillary support packages

Releases
--------
Tag a release (for example `v0.1.0`) and the CI release workflow will build binaries and publish a GitHub Release with artifacts.

Contributing
------------
We welcome contributions. Recommended flow:

1. Fork the repository
2. Create a branch `feature/your-feature`
3. Open a Pull Request with a clear description and tests where applicable

License
-------
This project is released under an open-source license. Add your chosen license file to the repository.

Contact
-------
For questions or to report issues, open an Issue in the repository or reach out via the project maintainer's GitHub account.
