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
- Git
- GitHub account

### Build

```bash
cd /home/habel/Documents/HI/hi
go build -o hi .
```

### Run

```bash
./hi auth
```

This will start the authentication flow and connect the application to the signal backend.

### Rewind and history

- Press `h` in the feed tab to review recently visited signals and chat sessions.
- Press `h` in the grouphouse tab to see recent workspace events.

### Development

- Run tests: `go test ./...`
- Format: `gofmt -w .`
- CI: GitHub Actions runs `gofmt`, `golangci-lint`, `go build`, and `go test` on push/PR to `main`.

### Release

Tag a release (e.g. `v0.1.0`) and CI will build platform binaries and publish a GitHub Release with artifacts.

## Recommended Repositories

- Primary repo: `https://github.com/Oridjinnn/Hi---Social-Working-media`
- Backend signals repo: `https://github.com/Oridjinnn/hi-signals`

## Notes

- The repository is configured to use `github.com/Oridjinnn/hi` as its Go module path.
- Keep build artifacts out of version control; only source files should be committed.

## Contributing

Pull requests are welcome. For the cleanest collaboration path:

1. Fork the repository
2. Create a feature branch
3. Submit a PR with a clear description of your changes

## License

This project is available under the terms of your chosen open source license.
