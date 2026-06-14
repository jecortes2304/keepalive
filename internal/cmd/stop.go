package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
)

var stopDaemon bool

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the active session or the daemon entirely",
	Long: `Stop the currently running keepalive session.
By default, only stops the active mouse movement session.
The daemon continues running and will trigger future schedules.

Use --daemon to kill the daemon process entirely.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pidPath := config.PIDPath()
		socketPath := config.SocketPath()

		pid, running := daemon.IsRunning(pidPath)
		if !running {
			return fmt.Errorf("no keepalive instance is running")
		}

		if stopDaemon {
			if err := daemon.StopRunning(pidPath, socketPath); err != nil {
				return err
			}
			fmt.Printf("Daemon stopped (PID %d)\n", pid)
			return nil
		}

		client := daemon.NewClient(socketPath)
		if err := client.SendStopSession(); err != nil {
			return fmt.Errorf("stopping session: %w", err)
		}
		fmt.Println("Active session stopped. Daemon continues running for future schedules.")
		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVar(&stopDaemon, "daemon", false, "stop the daemon process entirely (not just the active session)")
	rootCmd.AddCommand(stopCmd)
}
