package recording

import "time"

type Recording struct {
	ID         int64
	Name       string
	CreatedAt  time.Time
	DurationMs int64
	Points     []MovementPoint
}

type MovementPoint struct {
	Seq     int
	X       int
	Y       int
	DelayMs int64
}
