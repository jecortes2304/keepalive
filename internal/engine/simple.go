package engine

import (
	"context"
	"time"

	"github.com/go-vgo/robotgo"
)

type SimpleStrategy struct{}

func (s *SimpleStrategy) Execute(ctx context.Context) error {
	x, y := robotgo.Location()
	robotgo.Move(x+1, y)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(200 * time.Millisecond):
	}
	robotgo.Move(x, y)
	return nil
}

func (s *SimpleStrategy) Name() string {
	return "simple"
}
