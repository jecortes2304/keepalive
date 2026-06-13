package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"keepalive/internal/config"
	"keepalive/internal/engine"
	"keepalive/internal/recording"
)

type Scheduler struct {
	cron   *cron.Cron
	cfg    *config.AppConfig
	mu     sync.Mutex
	active bool
	cancel context.CancelFunc
}

func New(cfg *config.AppConfig) *Scheduler {
	return &Scheduler{
		cron: cron.New(),
		cfg:  cfg,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.catchUp(ctx); err != nil {
		return err
	}

	for _, profile := range s.cfg.Profiles {
		p := profile
		for _, sched := range p.Schedules {
			cronExpr := buildCronExpr(sched)
			duration := sched.Duration

			s.cron.AddFunc(cronExpr, func() {
				s.runForDuration(ctx, p, duration)
			})
		}
	}

	s.cron.Start()

	<-ctx.Done()
	s.cron.Stop()
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.cron.Stop()
}

func (s *Scheduler) catchUp(ctx context.Context) error {
	now := time.Now()

	for _, profile := range s.cfg.Profiles {
		w, active := ActiveWindow(profile.Schedules, now)
		if active {
			remaining := RemainingDuration(w, now)
			if remaining > 0 {
				p := profile
				go s.runForDuration(ctx, p, remaining)
				return nil
			}
		}
	}
	return nil
}

func (s *Scheduler) runForDuration(ctx context.Context, profile config.Profile, duration time.Duration) {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	runCtx, cancel := context.WithTimeout(ctx, duration)
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		cancel()
		s.mu.Lock()
		s.active = false
		s.cancel = nil
		s.mu.Unlock()
	}()

	eng := s.buildEngine(profile)
	eng.Start(runCtx)
}

func (s *Scheduler) buildEngine(profile config.Profile) *engine.Engine {
	var strategy engine.MovementStrategy

	switch profile.MovementType {
	case "recorded":
		store, err := recording.NewStore(config.DBPath())
		if err == nil {
			rec, err := store.Get(profile.Recording)
			if err == nil {
				strategy = recording.NewPlayer(rec, true)
			}
			store.Close()
		}
	case "simple":
		strategy = &engine.SimpleStrategy{}
	default:
		strategy = &engine.RandomStrategy{MaxPixels: 15}
	}

	opts := []engine.Option{
		engine.WithStrategy(strategy),
	}
	if profile.Interval > 0 {
		opts = append(opts, engine.WithInterval(profile.Interval))
	}

	return engine.New(opts...)
}

func buildCronExpr(sched config.Schedule) string {
	days := ""
	for i, d := range sched.Days {
		if i > 0 {
			days += ","
		}
		days += fmt.Sprintf("%d", d)
	}

	hour, minute := parseTime(sched.StartTime)
	return fmt.Sprintf("%s %s * * %s", minute, hour, days)
}

func parseTime(t string) (string, string) {
	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return "0", "0"
	}
	hour := strings.TrimLeft(parts[0], "0")
	minute := strings.TrimLeft(parts[1], "0")
	if hour == "" {
		hour = "0"
	}
	if minute == "" {
		minute = "0"
	}
	return hour, minute
}
