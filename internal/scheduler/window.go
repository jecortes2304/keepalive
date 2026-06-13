package scheduler

import (
	"time"

	"keepalive/internal/config"
)

type Window struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

func ActiveWindow(schedules []config.Schedule, now time.Time) (*Window, bool) {
	weekday := now.Weekday()
	currentTime := now.Format("15:04")

	for _, sched := range schedules {
		if !containsDay(sched.Days, weekday) {
			continue
		}

		startTime, err := time.Parse("15:04", sched.StartTime)
		if err != nil {
			continue
		}

		start := time.Date(now.Year(), now.Month(), now.Day(),
			startTime.Hour(), startTime.Minute(), 0, 0, now.Location())
		end := start.Add(sched.Duration)

		startStr := sched.StartTime
		endStr := end.Format("15:04")

		if currentTime >= startStr && currentTime < endStr {
			return &Window{
				Start:    start,
				End:      end,
				Duration: sched.Duration,
			}, true
		}
	}

	return nil, false
}

func RemainingDuration(w *Window, now time.Time) time.Duration {
	remaining := w.End.Sub(now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func containsDay(days []time.Weekday, day time.Weekday) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}
