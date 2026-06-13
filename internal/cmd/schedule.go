package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"keepalive/internal/config"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage scheduled keepalive sessions",
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new schedule",
	Long:  "Add a recurring schedule. Example: keepalive schedule add --days mon,thu,fri --start 10:00 --duration 30m",
	RunE:  runScheduleAdd,
}

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all schedules",
	RunE:  runScheduleList,
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove [index]",
	Short: "Remove a schedule by index",
	Args:  cobra.ExactArgs(1),
	RunE:  runScheduleRemove,
}

var scheduleEditCmd = &cobra.Command{
	Use:   "edit [index]",
	Short: "Edit a schedule by index",
	Args:  cobra.ExactArgs(1),
	RunE:  runScheduleEdit,
}

var (
	scheduleDays     string
	scheduleStart    string
	scheduleDuration string
	scheduleProfile  string

	scheduleEditDays     string
	scheduleEditStart    string
	scheduleEditDuration string
	scheduleEditProfile  string
)

func init() {
	scheduleAddCmd.Flags().StringVar(&scheduleDays, "days", "", "days of the week (e.g. mon,wed,fri)")
	scheduleAddCmd.Flags().StringVar(&scheduleStart, "start", "", "start time (e.g. 10:00)")
	scheduleAddCmd.Flags().StringVar(&scheduleDuration, "duration", "", "duration (e.g. 30m, 1h)")
	scheduleAddCmd.Flags().StringVar(&scheduleProfile, "profile", "", "profile to use (default: default)")
	scheduleAddCmd.MarkFlagRequired("days")
	scheduleAddCmd.MarkFlagRequired("start")
	scheduleAddCmd.MarkFlagRequired("duration")

	scheduleEditCmd.Flags().StringVar(&scheduleEditDays, "days", "", "days of the week (e.g. mon,wed,fri)")
	scheduleEditCmd.Flags().StringVar(&scheduleEditStart, "start", "", "start time (e.g. 10:00)")
	scheduleEditCmd.Flags().StringVar(&scheduleEditDuration, "duration", "", "duration (e.g. 30m, 1h)")
	scheduleEditCmd.Flags().StringVar(&scheduleEditProfile, "profile", "", "profile to use")

	scheduleCmd.AddCommand(scheduleAddCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleRemoveCmd)
	scheduleCmd.AddCommand(scheduleEditCmd)
	rootCmd.AddCommand(scheduleCmd)
}

func runScheduleAdd(cmd *cobra.Command, args []string) error {
	days, err := parseDays(scheduleDays)
	if err != nil {
		return err
	}

	if _, err := time.Parse("15:04", scheduleStart); err != nil {
		return fmt.Errorf("invalid start time %q (use HH:MM format)", scheduleStart)
	}

	duration, err := time.ParseDuration(scheduleDuration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	profileName := scheduleProfile
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return err
	}

	sched := config.Schedule{
		Days:      days,
		StartTime: scheduleStart,
		Duration:  duration,
	}
	profile.Schedules = append(profile.Schedules, sched)

	if err := config.Save(cfg); err != nil {
		return err
	}

	dayNames := make([]string, len(days))
	for i, d := range days {
		dayNames[i] = d.String()[:3]
	}
	fmt.Printf("Added schedule: %s at %s for %s (profile: %s)\n",
		strings.Join(dayNames, ","), scheduleStart, duration, profileName)
	return nil
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	idx := 0
	for _, profile := range cfg.Profiles {
		for _, sched := range profile.Schedules {
			dayNames := make([]string, len(sched.Days))
			for i, d := range sched.Days {
				dayNames[i] = d.String()[:3]
			}
			fmt.Printf("[%d] %s at %s for %s (profile: %s)\n",
				idx, strings.Join(dayNames, ","), sched.StartTime, sched.Duration, profile.Name)
			idx++
		}
	}

	if idx == 0 {
		fmt.Println("No schedules configured.")
	}
	return nil
}

func runScheduleRemove(cmd *cobra.Command, args []string) error {
	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	current := 0
	for pi := range cfg.Profiles {
		for si := range cfg.Profiles[pi].Schedules {
			if current == idx {
				cfg.Profiles[pi].Schedules = append(
					cfg.Profiles[pi].Schedules[:si],
					cfg.Profiles[pi].Schedules[si+1:]...,
				)
				if err := config.Save(cfg); err != nil {
					return err
				}
				fmt.Printf("Removed schedule [%d]\n", idx)
				return nil
			}
			current++
		}
	}

	return fmt.Errorf("schedule index %d not found", idx)
}

func runScheduleEdit(cmd *cobra.Command, args []string) error {
	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid index: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Find the schedule at the given global index
	current := 0
	for pi := range cfg.Profiles {
		for si := range cfg.Profiles[pi].Schedules {
			if current == idx {
				sched := &cfg.Profiles[pi].Schedules[si]

				if cmd.Flags().Changed("days") {
					days, err := parseDays(scheduleEditDays)
					if err != nil {
						return err
					}
					sched.Days = days
				}

				if cmd.Flags().Changed("start") {
					if _, err := time.Parse("15:04", scheduleEditStart); err != nil {
						return fmt.Errorf("invalid start time %q (use HH:MM format)", scheduleEditStart)
					}
					sched.StartTime = scheduleEditStart
				}

				if cmd.Flags().Changed("duration") {
					duration, err := time.ParseDuration(scheduleEditDuration)
					if err != nil {
						return fmt.Errorf("invalid duration: %w", err)
					}
					sched.Duration = duration
				}

				if cmd.Flags().Changed("profile") {
					// Move schedule to a different profile
					if _, err := cfg.GetProfile(scheduleEditProfile); err != nil {
						return err
					}
					if scheduleEditProfile != cfg.Profiles[pi].Name {
						// Remove from current profile, add to target
						schedCopy := *sched
						cfg.Profiles[pi].Schedules = append(
							cfg.Profiles[pi].Schedules[:si],
							cfg.Profiles[pi].Schedules[si+1:]...,
						)
						targetProfile, _ := cfg.GetProfile(scheduleEditProfile)
						targetProfile.Schedules = append(targetProfile.Schedules, schedCopy)
					}
				}

				if err := config.Save(cfg); err != nil {
					return err
				}

				fmt.Printf("Updated schedule [%d]\n", idx)
				return nil
			}
			current++
		}
	}

	return fmt.Errorf("schedule index %d not found", idx)
}

func parseDays(s string) ([]time.Weekday, error) {
	dayMap := map[string]time.Weekday{
		"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday,
		"wed": time.Wednesday, "thu": time.Thursday, "fri": time.Friday,
		"sat": time.Saturday,
	}

	parts := strings.Split(strings.ToLower(s), ",")
	var days []time.Weekday
	for _, p := range parts {
		p = strings.TrimSpace(p)
		d, ok := dayMap[p]
		if !ok {
			return nil, fmt.Errorf("invalid day %q (use: mon,tue,wed,thu,fri,sat,sun)", p)
		}
		days = append(days, d)
	}
	return days, nil
}
