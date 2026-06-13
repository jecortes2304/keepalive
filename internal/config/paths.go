package config

import (
	"os"
	"path/filepath"
	"runtime"
)

func Dir() string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "keepalive")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, "keepalive")
	default:
		cfg := os.Getenv("XDG_CONFIG_HOME")
		if cfg == "" {
			cfg = filepath.Join(home, ".config")
		}
		return filepath.Join(cfg, "keepalive")
	}
}

func DataDir() string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "keepalive")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, "keepalive")
	default:
		data := os.Getenv("XDG_DATA_HOME")
		if data == "" {
			data = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(data, "keepalive")
	}
}

func RuntimeDir() string {
	if runtime.GOOS == "linux" {
		if runDir := os.Getenv("XDG_RUNTIME_DIR"); runDir != "" {
			return filepath.Join(runDir, "keepalive")
		}
	}
	return filepath.Join(os.TempDir(), "keepalive")
}

func DBPath() string {
	return filepath.Join(DataDir(), "keepalive.db")
}

func PIDPath() string {
	return filepath.Join(RuntimeDir(), "keepalive.pid")
}

func SocketPath() string {
	return filepath.Join(RuntimeDir(), "keepalive.sock")
}

func LogDir() string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "keepalive")
	case "windows":
		return filepath.Join(DataDir(), "logs")
	default:
		state := os.Getenv("XDG_STATE_HOME")
		if state == "" {
			state = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(state, "keepalive")
	}
}

func LogPath() string {
	return filepath.Join(LogDir(), "keepalive.log")
}
