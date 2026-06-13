package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"keepalive/internal/scheduler"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install keepalive as a system daemon",
	Long:  "Register keepalive with the OS daemon manager (launchd/systemd/Task Scheduler) to auto-start on boot.",
	RunE: func(cmd *cobra.Command, args []string) error {
		binary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving binary path: %w", err)
		}

		if err := scheduler.Install(binary); err != nil {
			return fmt.Errorf("installing daemon: %w", err)
		}

		fmt.Println("Keepalive daemon installed and started.")
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the keepalive system daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := scheduler.Uninstall(); err != nil {
			return fmt.Errorf("uninstalling daemon: %w", err)
		}

		fmt.Println("Keepalive daemon removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
