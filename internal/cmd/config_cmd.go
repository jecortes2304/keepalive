package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration profiles",
}

var configCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigCreate,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE:  runConfigList,
}

var configShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigShow,
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigDelete,
}

var configEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit an existing profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigEdit,
}

var configDefaultCmd = &cobra.Command{
	Use:   "set-default [name]",
	Short: "Set the default profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigSetDefault,
}

var (
	configInterval     string
	configDuration     string
	configMovementType string
	configRecording    string

	configEditInterval     string
	configEditDuration     string
	configEditMovementType string
	configEditRecording    string
)

func init() {
	configCreateCmd.Flags().StringVar(&configInterval, "interval", "30s", "interval between movements")
	configCreateCmd.Flags().StringVar(&configDuration, "duration", "0s", "run duration (0 = indefinite)")
	configCreateCmd.Flags().StringVar(&configMovementType, "movement", "random", "movement type (simple, random, recorded)")
	configCreateCmd.Flags().StringVar(&configRecording, "recording", "", "recording name to use")

	configEditCmd.Flags().StringVar(&configEditInterval, "interval", "", "interval between movements")
	configEditCmd.Flags().StringVar(&configEditDuration, "duration", "", "run duration (0 = indefinite)")
	configEditCmd.Flags().StringVar(&configEditMovementType, "movement", "", "movement type (simple, random, recorded)")
	configEditCmd.Flags().StringVar(&configEditRecording, "recording", "", "recording name to use")

	configCmd.AddCommand(configCreateCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configDeleteCmd)
	configCmd.AddCommand(configDefaultCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigCreate(_ *cobra.Command, args []string) error {
	name := args[0]

	interval, err := time.ParseDuration(configInterval)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	duration, err := time.ParseDuration(configDuration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	profile := config.Profile{
		Name:         name,
		Interval:     interval,
		Duration:     duration,
		MovementType: configMovementType,
		Recording:    configRecording,
	}

	if err := cfg.AddProfile(profile); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Created profile %q\n", name)
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	profile, err := cfg.GetProfile(name)
	if err != nil {
		return err
	}

	if cmd.Flags().Changed("interval") {
		interval, err := time.ParseDuration(configEditInterval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		profile.Interval = interval
	}

	if cmd.Flags().Changed("duration") {
		duration, err := time.ParseDuration(configEditDuration)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		profile.Duration = duration
	}

	if cmd.Flags().Changed("movement") {
		profile.MovementType = configEditMovementType
	}

	if cmd.Flags().Changed("recording") {
		profile.Recording = configEditRecording
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Updated profile %q\n", name)
	return nil
}

func runConfigList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("%-15s %-10s %-10s %-10s %s\n", "NAME", "INTERVAL", "DURATION", "MOVEMENT", "DEFAULT")
	for _, p := range cfg.Profiles {
		def := ""
		if p.Name == cfg.DefaultProfile {
			def = "*"
		}
		dur := "infinite"
		if p.Duration > 0 {
			dur = p.Duration.String()
		}
		fmt.Printf("%-15s %-10s %-10s %-10s %s\n",
			p.Name, p.Interval, dur, p.MovementType, def)
	}
	return nil
}

func runConfigShow(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	profile, err := cfg.GetProfile(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Name:          %s\n", profile.Name)
	fmt.Printf("Interval:      %s\n", profile.Interval)
	if profile.Duration > 0 {
		fmt.Printf("Duration:      %s\n", profile.Duration)
	} else {
		fmt.Printf("Duration:      indefinite\n")
	}
	fmt.Printf("Movement Type: %s\n", profile.MovementType)
	if profile.Recording != "" {
		fmt.Printf("Recording:     %s\n", profile.Recording)
	}
	if len(profile.Schedules) > 0 {
		fmt.Printf("Schedules:\n")
		for _, s := range profile.Schedules {
			days := ""
			for i, d := range s.Days {
				if i > 0 {
					days += ","
				}
				days += d.String()[:3]
			}
			fmt.Printf("  - %s at %s for %s\n", days, s.StartTime, s.Duration)
		}
	}
	return nil
}

func runConfigDelete(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := cfg.DeleteProfile(args[0]); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Deleted profile %q\n", args[0])
	return nil
}

func runConfigSetDefault(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := cfg.SetDefault(args[0]); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Default profile set to %q\n", args[0])
	return nil
}
