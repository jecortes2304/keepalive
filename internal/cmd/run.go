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
	"keepalive/internal/engine"
	"keepalive/internal/recording"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start keeping the machine active",
	RunE:  runRun,
}

var (
	runDuration  string
	runInterval  string
	runProfile   string
	runRecording string
	runDaemon    bool
)

func init() {
	runCmd.Flags().StringVarP(&runDuration, "duration", "d", "", "how long to run (e.g. 10m, 1h)")
	runCmd.Flags().StringVarP(&runInterval, "interval", "i", "", "interval between movements (e.g. 30s)")
	runCmd.Flags().StringVarP(&runProfile, "config", "c", "", "profile name to use")
	runCmd.Flags().StringVarP(&runRecording, "recording", "r", "", "recording name to play back")
	runCmd.Flags().BoolVar(&runDaemon, "daemon", false, "run in background as daemon")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	pidPath := config.PIDPath()
	socketPath := config.SocketPath()

	if pid, running := daemon.IsRunning(pidPath); running {
		return fmt.Errorf("keepalive is already running (PID %d). Use 'keepalive stop' first", pid)
	}

	if runDaemon {
		binary, _ := os.Executable()
		daemonArgs := []string{"run"}
		if runDuration != "" {
			daemonArgs = append(daemonArgs, "--duration", runDuration)
		}
		if runInterval != "" {
			daemonArgs = append(daemonArgs, "--interval", runInterval)
		}
		if runProfile != "" {
			daemonArgs = append(daemonArgs, "--config", runProfile)
		}
		if runRecording != "" {
			daemonArgs = append(daemonArgs, "--recording", runRecording)
		}

		pid, err := daemon.StartDetached(binary, daemonArgs)
		if err != nil {
			return err
		}
		fmt.Printf("Started keepalive in background (PID %d)\n", pid)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	profileName := runProfile
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return err
	}

	var opts []engine.Option

	interval := profile.Interval
	if runInterval != "" {
		d, err := time.ParseDuration(runInterval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		interval = d
	}
	opts = append(opts, engine.WithInterval(interval))

	var duration time.Duration
	if runDuration != "" {
		d, err := time.ParseDuration(runDuration)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		duration = d
	} else {
		duration = profile.Duration
	}
	if duration > 0 {
		opts = append(opts, engine.WithDuration(duration))
	}

	recName := runRecording
	if recName == "" {
		recName = profile.Recording
	}
	if recName != "" {
		store, err := recording.NewStore(config.DBPath())
		if err != nil {
			return err
		}
		defer store.Close()

		rec, err := store.Get(recName)
		if err != nil {
			return err
		}
		player := recording.NewPlayer(rec, true)
		opts = append(opts, engine.WithStrategy(player))
	} else {
		switch profile.MovementType {
		case "simple":
			opts = append(opts, engine.WithStrategy(&engine.SimpleStrategy{}))
		default:
			opts = append(opts, engine.WithStrategy(&engine.RandomStrategy{MaxPixels: 15}))
		}
	}

	opts = append(opts, engine.WithOnTick(func(info engine.TickInfo) {
		if info.Remaining >= 0 {
			fmt.Printf("[%s] Movement #%d at (%d,%d) | remaining: %s\n",
				info.Timestamp.Format("15:04:05"), info.Movements,
				info.Position.X, info.Position.Y,
				info.Remaining.Truncate(time.Second))
		} else {
			fmt.Printf("[%s] Movement #%d at (%d,%d) | running: %s\n",
				info.Timestamp.Format("15:04:05"), info.Movements,
				info.Position.X, info.Position.Y,
				info.TotalTime.Truncate(time.Second))
		}
	}))

	eng := engine.New(opts...)

	if err := daemon.WritePID(pidPath); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer daemon.RemovePID(pidPath)

	ipcServer := daemon.NewServer(socketPath)
	if err := ipcServer.Start(); err != nil {
		return fmt.Errorf("starting IPC server: %w", err)
	}
	defer ipcServer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			info := eng.Info()
			ipcServer.UpdateStatus(daemon.StatusInfo{
				PID:       os.Getpid(),
				StartedAt: time.Now().Add(-info.TotalTime),
				Strategy:  info.Strategy,
				Movements: info.Movements,
				Duration:  duration,
				Remaining: info.Remaining,
				Profile:   profileName,
				Running:   eng.IsRunning(),
			})
			time.Sleep(time.Second)
		}
	}()

	go func() {
		<-sigCh
		fmt.Println("\nStopping...")
		cancel()
	}()

	if duration > 0 {
		fmt.Printf("Keepalive started (profile: %s, duration: %s, interval: %s)\n",
			profileName, duration, interval)
	} else {
		fmt.Printf("Keepalive started (profile: %s, interval: %s). Press Ctrl+C to stop.\n",
			profileName, interval)
	}

	return eng.Start(ctx)
}
