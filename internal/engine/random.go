package engine

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/go-vgo/robotgo"
)

type RandomStrategy struct {
	MaxPixels int
}

func (r *RandomStrategy) Execute(ctx context.Context) error {
	maxPx := r.MaxPixels
	if maxPx <= 0 {
		maxPx = 15
	}

	x, y := robotgo.Location()
	dx, dy := JitterPosition(maxPx)

	steps := 3 + rand.IntN(5)
	stepX := dx / steps
	stepY := dy / steps

	for i := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		newX := x + stepX*(i+1)
		newY := y + stepY*(i+1)
		robotgo.Move(newX, newY)

		pause := time.Duration(20+rand.IntN(40)) * time.Millisecond
		time.Sleep(pause)
	}

	returnSteps := 2 + rand.IntN(3)
	finalX, finalY := robotgo.Location()
	retDX := (x - finalX) / returnSteps
	retDY := (y - finalY) / returnSteps

	for i := range returnSteps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		newX := finalX + retDX*(i+1)
		newY := finalY + retDY*(i+1)
		robotgo.Move(newX, newY)

		pause := time.Duration(20+rand.IntN(40)) * time.Millisecond
		time.Sleep(pause)
	}

	robotgo.Move(x, y)
	return nil
}

func (r *RandomStrategy) Name() string {
	return "random"
}
