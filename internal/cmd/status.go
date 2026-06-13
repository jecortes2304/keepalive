package cmd

import (
	"encoding/json"
	"fmt"
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

		pid, running := daemon.IsRunning(pidPath)
		if !running {
			fmt.Println("No keepalive instance is running.")
			return nil
		}

		client := daemon.NewClient(socketPath)
		info, err := client.GetStatus()
		if err != nil {
			fmt.Printf("Instance running (PID %d) but unable to query status: %v\n", pid, err)
			return nil
		}

		if statusJSON {
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Keepalive is running\n")
		fmt.Printf("  PID:        %d\n", info.PID)
		fmt.Printf("  Profile:    %s\n", info.Profile)
		fmt.Printf("  Strategy:   %s\n", info.Strategy)
		fmt.Printf("  Movements:  %d\n", info.Movements)
		fmt.Printf("  Uptime:     %s\n", time.Since(info.StartedAt).Truncate(time.Second))
		if info.Duration > 0 {
			fmt.Printf("  Duration:   %s\n", info.Duration)
			fmt.Printf("  Remaining:  %s\n", info.Remaining.Truncate(time.Second))
		} else {
			fmt.Printf("  Duration:   indefinite\n")
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output status as JSON")
	rootCmd.AddCommand(statusCmd)
}
