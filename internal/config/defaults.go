package config

import "time"

func DefaultProfile() Profile {
	return Profile{
		Name:         "default",
		Interval:     30 * time.Second,
		Duration:     0,
		Recording:    "",
		MovementType: "random",
		Schedules:    nil,
	}
}
