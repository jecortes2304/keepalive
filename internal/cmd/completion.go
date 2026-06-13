package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate and install shell completions",
	Long: `Generate shell completion scripts and install them.
Without arguments, detects your current shell and installs automatically.

Examples:
  keepalive completion         # auto-detect and install
  keepalive completion zsh     # install zsh completions
  keepalive completion bash    # install bash completions
  keepalive completion fish    # install fish completions`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := detectShell()
		if len(args) > 0 {
			shell = args[0]
		}

		switch shell {
		case "zsh":
			return installZshCompletion()
		case "bash":
			return installBashCompletion()
		case "fish":
			return installFishCompletion()
		default:
			return fmt.Errorf("unsupported shell %q. Use: bash, zsh, or fish", shell)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	switch {
	case contains(shell, "zsh"):
		return "zsh"
	case contains(shell, "bash"):
		return "bash"
	case contains(shell, "fish"):
		return "fish"
	default:
		return "zsh"
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && filepath.Base(s) == sub || matchSubstring(s, sub))
}

func matchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func installZshCompletion() error {
	var dir string
	switch runtime.GOOS {
	case "darwin":
		dir = "/usr/local/share/zsh/site-functions"
		if _, err := os.Stat("/opt/homebrew/share/zsh/site-functions"); err == nil {
			dir = "/opt/homebrew/share/zsh/site-functions"
		}
	default:
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share", "zsh", "site-functions")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating completion dir: %w", err)
	}

	path := filepath.Join(dir, "_keepalive")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating completion file: %w", err)
	}
	defer f.Close()

	if err := rootCmd.GenZshCompletion(f); err != nil {
		return err
	}

	fmt.Printf("Zsh completions installed to: %s\n", path)
	fmt.Println("Restart your shell or run: exec zsh")
	return nil
}

func installBashCompletion() error {
	var dir string
	switch runtime.GOOS {
	case "darwin":
		dir = "/usr/local/etc/bash_completion.d"
		if _, err := os.Stat("/opt/homebrew/etc/bash_completion.d"); err == nil {
			dir = "/opt/homebrew/etc/bash_completion.d"
		}
	default:
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share", "bash-completion", "completions")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating completion dir: %w", err)
	}

	path := filepath.Join(dir, "keepalive")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating completion file: %w", err)
	}
	defer f.Close()

	if err := rootCmd.GenBashCompletion(f); err != nil {
		return err
	}

	fmt.Printf("Bash completions installed to: %s\n", path)
	fmt.Println("Restart your shell or run: source " + path)
	return nil
}

func installFishCompletion() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "fish", "completions")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating completion dir: %w", err)
	}

	path := filepath.Join(dir, "keepalive.fish")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating completion file: %w", err)
	}
	defer f.Close()

	if err := rootCmd.GenFishCompletion(f, true); err != nil {
		return err
	}

	fmt.Printf("Fish completions installed to: %s\n", path)
	return nil
}
