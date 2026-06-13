package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"keepalive/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "keepalive",
	Short: "Keep your machine active with human-like mouse movements",
	Long:  "A CLI/TUI tool that prevents your machine from sleeping by simulating realistic mouse movements.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := tui.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
