package engine

import (
	"context"
	"image"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
)

type TickInfo struct {
	Timestamp time.Time
	Position  image.Point
	Strategy  string
	Movements int64
	Remaining time.Duration
	TotalTime time.Duration
}

type Option func(*Engine)

type Engine struct {
	strategy  MovementStrategy
	interval  time.Duration
	duration  time.Duration
	onTick    func(TickInfo)
	mu        sync.Mutex
	running   bool
	movements int64
	startedAt time.Time
	cancel    context.CancelFunc
}

func New(opts ...Option) *Engine {
	e := &Engine{
		strategy: &RandomStrategy{MaxPixels: 15},
		interval: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func WithStrategy(s MovementStrategy) Option {
	return func(e *Engine) { e.strategy = s }
}

func WithInterval(d time.Duration) Option {
	return func(e *Engine) { e.interval = d }
}

func WithDuration(d time.Duration) Option {
	return func(e *Engine) { e.duration = d }
}

func WithOnTick(fn func(TickInfo)) Option {
	return func(e *Engine) { e.onTick = fn }
}

func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return ErrAlreadyRunning
	}
	e.running = true
	e.movements = 0
	e.startedAt = time.Now()

	if e.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.duration)
		defer cancel()
	}

	ctx, e.cancel = context.WithCancel(ctx)
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.cancel = nil
		e.mu.Unlock()
	}()

	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := e.strategy.Execute(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		e.mu.Lock()
		e.movements++
		movements := e.movements
		e.mu.Unlock()

		if e.onTick != nil {
			x, y := robotgo.Location()
			var remaining time.Duration
			if e.duration > 0 {
				elapsed := time.Since(e.startedAt)
				remaining = e.duration - elapsed
				if remaining < 0 {
					remaining = 0
				}
			} else {
				remaining = -1
			}
			e.onTick(TickInfo{
				Timestamp: time.Now(),
				Position:  image.Point{X: x, Y: y},
				Strategy:  e.strategy.Name(),
				Movements: movements,
				Remaining: remaining,
				TotalTime: time.Since(e.startedAt),
			})
		}

		interval := JitterDuration(e.interval, 0.2)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *Engine) Info() TickInfo {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return TickInfo{}
	}
	var remaining time.Duration
	if e.duration > 0 {
		remaining = e.duration - time.Since(e.startedAt)
		if remaining < 0 {
			remaining = 0
		}
	} else {
		remaining = -1
	}
	return TickInfo{
		Timestamp: time.Now(),
		Strategy:  e.strategy.Name(),
		Movements: e.movements,
		Remaining: remaining,
		TotalTime: time.Since(e.startedAt),
	}
}
