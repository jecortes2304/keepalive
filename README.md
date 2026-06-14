# keepalive

A CLI/TUI tool that keeps your machine active with human-like mouse movements.

## Features

- **Multiple movement strategies**: simple (1px back and forth), random (natural-looking movements), or recorded (replay your own patterns)
- **Recording mode**: Capture real mouse movements and play them back for maximum realism
- **Duration control**: Run for a specific time (`--duration 10m`) or indefinitely
- **Scheduler**: Program recurring sessions (e.g., every Thursday and Friday at 10:00 for 30 minutes)
- **Daemon mode**: Run in background with `--daemon`, query status, and stop remotely
- **Configuration profiles**: Save and switch between different configurations
- **TUI interface**: Interactive terminal UI with tab navigation (powered by Charm)
- **Cross-platform**: macOS, Linux, and Windows

## Installation

### From source

```bash
go install keepalive/cmd/keepalive@latest
```

### From releases

Download the latest binary from the [Releases](https://github.com/jecortes/keepalive/releases) page.

## Usage

### TUI Mode

```bash
keepalive
```

Launches the interactive terminal UI. Navigate with number keys (1-5) or Tab.

### CLI Commands

```bash
# Start with default profile
keepalive run

# Run for 10 minutes
keepalive run --duration 10m

# Run in background
keepalive run --daemon --duration 1h

# Check status
keepalive status

# Stop running instance
keepalive stop

# Record mouse movements
keepalive record mypattern

# List recordings
keepalive record --list

# Run with a specific recording
keepalive run --recording mypattern

# Manage schedules
keepalive schedule add --days mon,thu,fri --start 10:00 --duration 30m
keepalive schedule list
keepalive schedule remove 0

# Manage profiles
keepalive config create work --interval 45s --duration 2h
keepalive config list
keepalive config set-default work

# Install as system daemon (auto-start on boot)
keepalive install

# Remove system daemon
keepalive uninstall

# Version
keepalive version
```

## Configuration

Configuration is stored at:
- **macOS**: `~/Library/Application Support/keepalive/config.yaml`
- **Linux**: `~/.config/keepalive/config.yaml`
- **Windows**: `%APPDATA%/keepalive/config.yaml`

### Example config

```yaml
default_profile: default
profiles:
  - name: default
    interval: 30s
    duration: 0s
    movement_type: random
  - name: work
    interval: 45s
    duration: 2h
    movement_type: recorded
    recording: my-pattern
    schedules:
      - days: [4, 5]
        start_time: "10:00"
        duration: 30m
```

## Building

```bash
# Build
make build

# Run directly
make run

# Run tests
make test
```

## Releasing

Releases are fully automated via GitHub Actions. When you push a version tag, the CI pipeline builds native binaries for all platforms, generates installers, and publishes everything to a GitHub Release.

### Release cycle

```bash
# 1. Make your changes and commit
git add .
git commit -m "add new feature X"

# 2. Tag a version (follows semver)
git tag v1.0.0

# 3. Push the tag — this triggers the release workflow
git push origin v1.0.0
```

That's it. GitHub Actions will:
1. Build native binaries (macOS amd64/arm64, Linux amd64/arm64, Windows amd64)
2. Generate `.deb` and `.rpm` packages for Linux
3. Build an NSIS installer (`.exe` setup wizard) for Windows
4. Update the Homebrew tap formula
5. Create checksums and publish the GitHub Release

### Local snapshot (dry run)

To test the release process locally without publishing:

```bash
make release-local
```

This uses GoReleaser to build archives for your current platform only.

### Versioning

This project follows [Semantic Versioning](https://semver.org/):
- `MAJOR`: Breaking changes to CLI interface
- `MINOR`: New features (new commands, options)
- `PATCH`: Bug fixes

### Release artifacts

| Platform | Artifacts |
|----------|-----------|
| macOS | `.tar.gz` binary, Homebrew formula |
| Linux | `.tar.gz` binary, `.deb`, `.rpm` |
| Windows | `.zip` binary, `.exe` installer (NSIS) |

### Homebrew installation (macOS)

Once the tap repo is set up:

```bash
brew tap jecortes2304/tap
brew install keepalive
```

### Setup (one-time)

To enable the full release pipeline you need two GitHub secrets in your repo settings:

1. **`GITHUB_TOKEN`** — Provided automatically by GitHub Actions
2. **`HOMEBREW_TAP_TOKEN`** — A GitHub PAT with `repo` scope for `jecortes2304/homebrew-tap`

You also need to create the repository `jecortes2304/homebrew-tap` (can be empty with just a README).

## Architecture

```
cmd/keepalive/main.go       Entry point
internal/
  cmd/                      Cobra CLI commands
  config/                   Configuration (Viper + YAML)
  engine/                   Movement engine (strategy pattern)
  recording/                Mouse recording/playback + SQLite
  daemon/                   PID file, IPC socket, process management
  scheduler/                Cron scheduling + OS daemon integration
  tui/                      Terminal UI (Bubbletea + Lipgloss)
```

## License

MIT
