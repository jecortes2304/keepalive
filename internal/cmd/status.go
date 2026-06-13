package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the running keepalive instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidPath := config.PIDPath()
		socketPath := config.SocketPath()

		installed := isDaemonInstalled()

		pid, running := daemon.IsRunning(pidPath)

		if statusJSON {
			return printJSONStatus(installed, running, pid, socketPath)
		}

		fmt.Println("Keepalive Status")
		fmt.Println("────────────────────────────────")

		if installed {
			fmt.Println("  Daemon installed:  yes")
		} else {
			fmt.Println("  Daemon installed:  no")
		}

		if !running {
			fmt.Println("  Daemon running:    no")
			fmt.Println()
			if !installed {
				fmt.Println("  Run 'keepalive install' to set up auto-start,")
				fmt.Println("  or 'keepalive daemon' to run manually.")
			} else {
				fmt.Println("  Daemon is installed but not running.")
				fmt.Println("  Try 'keepalive uninstall && keepalive install' to restart.")
			}
			return nil
		}

		fmt.Printf("  Daemon running:    yes (PID %d)\n", pid)

		client := daemon.NewClient(socketPath)
		info, err := client.GetStatus()
		if err != nil {
			fmt.Printf("  IPC status:        unavailable (%v)\n", err)
			fmt.Println()
			fmt.Printf("  Log file: %s\n", config.LogPath())
			return nil
		}

		fmt.Printf("  Profile:           %s\n", info.Profile)
		fmt.Printf("  Strategy:          %s\n", info.Strategy)
		fmt.Printf("  Movements:         %d\n", info.Movements)
		fmt.Printf("  Uptime:            %s\n", time.Since(info.StartedAt).Truncate(time.Second))
		if info.Duration > 0 {
			fmt.Printf("  Session duration:  %s\n", info.Duration)
			fmt.Printf("  Remaining:         %s\n", info.Remaining.Truncate(time.Second))
		}
		fmt.Println()
		fmt.Printf("  Log file: %s\n", config.LogPath())
		fmt.Printf("  Config:   %s\n", config.Dir()+"/config.yaml")

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output status as JSON")
	rootCmd.AddCommand(statusCmd)
}

func isDaemonInstalled() bool {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		plistPath := home + "/Library/LaunchAgents/com.keepalive.daemon.plist"
		_, err := os.Stat(plistPath)
		return err == nil
	case "linux":
		home, _ := os.UserHomeDir()
		servicePath := home + "/.config/systemd/user/keepalive.service"
		_, err := os.Stat(servicePath)
		return err == nil
	default:
		return false
	}
}

func printJSONStatus(installed, running bool, pid int, socketPath string) error {
	result := map[string]any{
		"installed": installed,
		"running":   running,
		"pid":       pid,
		"log_file":  config.LogPath(),
	}

	if running {
		client := daemon.NewClient(socketPath)
		info, err := client.GetStatus()
		if err == nil {
			result["profile"] = info.Profile
			result["strategy"] = info.Strategy
			result["movements"] = info.Movements
			result["uptime"] = time.Since(info.StartedAt).Truncate(time.Second).String()
			if info.Duration > 0 {
				result["duration"] = info.Duration.String()
				result["remaining"] = info.Remaining.Truncate(time.Second).String()
			}
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return nil
}
