package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
)

var (
	logsTail  int
	logsClear bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show daemon logs",
	Long:  "Display the daemon log file contents. Use --tail N to show last N lines.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := config.LogPath()

		if logsClear {
			if err := os.Truncate(logPath, 0); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No log file found.")
					return nil
				}
				return err
			}
			fmt.Println("Logs cleared.")
			return nil
		}

		data, err := os.ReadFile(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("No log file found at: %s\n", logPath)
				fmt.Println("The daemon has not written any logs yet.")
				return nil
			}
			return err
		}

		if len(data) == 0 {
			fmt.Println("Log file is empty.")
			return nil
		}

		if logsTail > 0 {
			lines := splitLines(data)
			start := len(lines) - logsTail
			if start < 0 {
				start = 0
			}
			for _, line := range lines[start:] {
				fmt.Println(line)
			}
			return nil
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 0, "show last N lines")
	logsCmd.Flags().BoolVar(&logsClear, "clear", false, "clear the log file")
	rootCmd.AddCommand(logsCmd)
}

func splitLines(data []byte) []string {
	var lines []string
	var current []byte
	for _, b := range data {
		if b == '\n' {
			lines = append(lines, string(current))
			current = current[:0]
		} else {
			current = append(current, b)
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}
