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

This project uses [GoReleaser](https://goreleaser.com/) for releases.

### Creating a release

```bash
# Tag a version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Build release locally (dry run)
goreleaser release --snapshot --clean

# Actual release (requires GITHUB_TOKEN)
goreleaser release --clean
```

### Versioning

This project follows [Semantic Versioning](https://semver.org/):
- `MAJOR`: Breaking changes to CLI interface
- `MINOR`: New features (new commands, options)
- `PATCH`: Bug fixes

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
