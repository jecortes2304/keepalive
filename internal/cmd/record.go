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
	"keepalive/internal/recording"
)

var (
	recordList   bool
	recordDelete string
	recordForce  bool
	recordRename string
	recordTo     string
)

var recordCmd = &cobra.Command{
	Use:   "record [name]",
	Short: "Record mouse movements for realistic playback",
	Long:  "Start recording your mouse movements. Press Ctrl+C to stop and save.",
	RunE:  runRecord,
}

func init() {
	recordCmd.Flags().BoolVar(&recordList, "list", false, "list all saved recordings")
	recordCmd.Flags().StringVar(&recordDelete, "delete", "", "delete a recording by name")
	recordCmd.Flags().BoolVar(&recordForce, "force", false, "force deletion even if recording is used by a profile")
	recordCmd.Flags().StringVar(&recordRename, "rename", "", "rename a recording (use with --to)")
	recordCmd.Flags().StringVar(&recordTo, "to", "", "new name for the recording (use with --rename)")
	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
	store, err := recording.NewStore(config.DBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	if recordList {
		recordings, err := store.List()
		if err != nil {
			return err
		}
		if len(recordings) == 0 {
			fmt.Println("No recordings saved.")
			return nil
		}
		fmt.Printf("%-20s %-12s %-20s\n", "NAME", "DURATION", "CREATED")
		for _, r := range recordings {
			fmt.Printf("%-20s %-12s %-20s\n",
				r.Name,
				recording.FormatDuration(r.DurationMs),
				r.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		return nil
	}

	if recordDelete != "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		for i := range cfg.Profiles {
			if cfg.Profiles[i].Recording == recordDelete {
				if !recordForce {
					return fmt.Errorf("recording %q is used by profile %q. Remove or change the profile first (or use --force)", recordDelete, cfg.Profiles[i].Name)
				}
				cfg.Profiles[i].Recording = ""
				cfg.Profiles[i].MovementType = "random"
			}
		}

		if recordForce {
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
		}

		if err := store.Delete(recordDelete); err != nil {
			return err
		}
		fmt.Printf("Deleted recording %q\n", recordDelete)
		return nil
	}

	if recordRename != "" {
		if recordTo == "" {
			return fmt.Errorf("--rename requires --to flag with the new name")
		}
		if err := store.Rename(recordRename, recordTo); err != nil {
			return err
		}

		// Update any profiles referencing the old recording name
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		updated := false
		for i := range cfg.Profiles {
			if cfg.Profiles[i].Recording == recordRename {
				cfg.Profiles[i].Recording = recordTo
				updated = true
			}
		}
		if updated {
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
		}

		fmt.Printf("Renamed recording %q to %q\n", recordRename, recordTo)
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("please provide a name for the recording, or use --list / --delete / --rename")
	}

	name := args[0]
	if store.Exists(name) {
		return fmt.Errorf("recording %q already exists. Delete it first or choose another name", name)
	}

	fmt.Printf("Recording mouse movements as %q. Press Ctrl+C to stop...\n", name)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	rec := recording.NewRecorder(50 * time.Millisecond)

	go func() {
		<-sigCh
		cancel()
	}()

	rec.Start(ctx)

	result := rec.Result(name)
	if len(result.Points) == 0 {
		fmt.Println("No movements captured.")
		return nil
	}

	if err := store.Save(result); err != nil {
		return fmt.Errorf("saving recording: %w", err)
	}

	fmt.Printf("Saved recording %q (%d points, %s)\n",
		name, len(result.Points), recording.FormatDuration(result.DurationMs))
	return nil
}
