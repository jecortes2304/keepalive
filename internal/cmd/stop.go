package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running keepalive instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidPath := config.PIDPath()
		socketPath := config.SocketPath()

		pid, running := daemon.IsRunning(pidPath)
		if !running {
			return fmt.Errorf("no keepalive instance is running")
		}

		if err := daemon.StopRunning(pidPath, socketPath); err != nil {
			return err
		}

		fmt.Printf("Stopped keepalive (PID %d)\n", pid)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
