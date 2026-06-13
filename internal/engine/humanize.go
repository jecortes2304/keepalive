package engine

import (
	"math/rand/v2"
	"time"
)

func JitterDuration(base time.Duration, maxPct float64) time.Duration {
	if maxPct <= 0 {
		return base
	}
	jitter := float64(base) * maxPct * (rand.Float64()*2 - 1)
	result := time.Duration(float64(base) + jitter)
	if result < time.Second {
		result = time.Second
	}
	return result
}

func JitterPosition(maxPx int) (int, int) {
	if maxPx <= 0 {
		return 0, 0
	}
	dx := rand.IntN(maxPx*2+1) - maxPx
	dy := rand.IntN(maxPx*2+1) - maxPx
	return dx, dy
}
