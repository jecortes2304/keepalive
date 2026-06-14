package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
	"keepalive/internal/logging"
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
	if err := logging.Init(config.LogPath()); err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}
	defer logging.Close()

	logging.Info("daemon starting (PID %d)", os.Getpid())

	pidPath := config.PIDPath()
	socketPath := config.SocketPath()

	if pid, running := daemon.IsRunning(pidPath); running {
		logging.Error("daemon already running (PID %d)", pid)
		return fmt.Errorf("daemon already running (PID %d)", pid)
	}

	if err := daemon.WritePID(pidPath); err != nil {
		logging.Error("writing PID file: %v", err)
		return err
	}
	defer daemon.RemovePID(pidPath)

	cfg, err := config.Load()
	if err != nil {
		logging.Error("loading config: %v", err)
		return err
	}

	schedCount := 0
	for _, p := range cfg.Profiles {
		schedCount += len(p.Schedules)
	}
	logging.Info("loaded config: %d profiles, %d schedules", len(cfg.Profiles), schedCount)

	ipcServer := daemon.NewServer(socketPath)
	if err := ipcServer.Start(); err != nil {
		logging.Error("starting IPC server: %v", err)
		return fmt.Errorf("starting IPC server: %w", err)
	}
	defer ipcServer.Stop()
	logging.Info("IPC server listening on %s", socketPath)

	startedAt := time.Now()
	ipcServer.UpdateStatus(daemon.StatusInfo{
		PID:       os.Getpid(),
		StartedAt: startedAt,
		Strategy:  "scheduler",
		Profile:   "daemon",
		Running:   true,
	})

	sched := scheduler.New(cfg)
	sched.OnStatusChange = func(profile string, strategy string, running bool, movements int64, duration, remaining time.Duration) {
		if running {
			logging.Info("schedule active: profile=%s strategy=%s movements=%d", profile, strategy, movements)
		} else {
			logging.Info("schedule completed: profile=%s movements=%d", profile, movements)
		}
		info := daemon.StatusInfo{
			PID:       os.Getpid(),
			StartedAt: startedAt,
			Strategy:  strategy,
			Movements: movements,
			Duration:  duration,
			Remaining: remaining,
			Profile:   profile,
			Running:   true,
		}
		if !running {
			info.Strategy = "scheduler"
			info.Profile = "daemon"
		}
		ipcServer.UpdateStatus(info)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ipcServer.OnStopSession = func() {
		logging.Info("stop-session command received via IPC")
		sched.StopSession()
	}

	ipcServer.OnStop = func() {
		logging.Info("stop-daemon command received via IPC")
		cancel()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logging.Info("received signal: %s, shutting down", sig)
		cancel()
	}()

	logging.Info("scheduler started, waiting for triggers")
	err = sched.Start(ctx)
	logging.Info("daemon exiting")
	return err
}
