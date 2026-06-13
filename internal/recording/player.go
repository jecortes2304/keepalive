package recording

import (
	"context"
	"time"

	"github.com/go-vgo/robotgo"
)

type Player struct {
	recording *Recording
	loop      bool
}

func NewPlayer(rec *Recording, loop bool) *Player {
	return &Player{recording: rec, loop: loop}
}

func (p *Player) Execute(ctx context.Context) error {
	for {
		for _, point := range p.recording.Points {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if point.DelayMs > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(point.DelayMs) * time.Millisecond):
				}
			}

			robotgo.Move(point.X, point.Y)
		}

		if !p.loop {
			return nil
		}
	}
}

func (p *Player) Name() string {
	return "recorded:" + p.recording.Name
}
