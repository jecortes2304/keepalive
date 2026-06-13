package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
	"keepalive/internal/scheduler"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Run the scheduler daemon (used internally)",
	Hidden: true,
	RunE:   runDaemonCmd,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}

func runDaemonCmd(*cobra.Command, []string) error {
	pidPath := config.PIDPath()

	if pid, running := daemon.IsRunning(pidPath); running {
		return fmt.Errorf("daemon already running (PID %d)", pid)
	}

	if err := daemon.WritePID(pidPath); err != nil {
		return err
	}
	defer daemon.RemovePID(pidPath)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	sched := scheduler.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("Daemon shutting down...")
		cancel()
	}()

	return sched.Start(ctx)
}
