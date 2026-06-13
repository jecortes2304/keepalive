package recording

import (
	"context"
	"time"

	"github.com/go-vgo/robotgo"
)

type Recorder struct {
	sampleInterval time.Duration
	points         []MovementPoint
	startTime      time.Time
}

func NewRecorder(sampleInterval time.Duration) *Recorder {
	if sampleInterval <= 0 {
		sampleInterval = 50 * time.Millisecond
	}
	return &Recorder{
		sampleInterval: sampleInterval,
	}
}

func (r *Recorder) Start(ctx context.Context) {
	r.startTime = time.Now()
	r.points = nil

	var lastX, lastY int
	seq := 0
	lastTime := r.startTime

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.sampleInterval):
		}

		x, y := robotgo.Location()
		now := time.Now()

		if x != lastX || y != lastY || seq == 0 {
			r.points = append(r.points, MovementPoint{
				Seq:     seq,
				X:       x,
				Y:       y,
				DelayMs: now.Sub(lastTime).Milliseconds(),
			})
			seq++
			lastX, lastY = x, y
			lastTime = now
		}
	}
}

func (r *Recorder) Result(name string) *Recording {
	var totalMs int64
	for _, p := range r.points {
		totalMs += p.DelayMs
	}
	return &Recording{
		Name:       name,
		CreatedAt:  r.startTime,
		DurationMs: totalMs,
		Points:     r.points,
	}
}

func (r *Recorder) PointCount() int {
	return len(r.points)
}
