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
	"keepalive/internal/logging"
	"keepalive/internal/recording"
)

type Scheduler struct {
	cron           *cron.Cron
	cfg            *config.AppConfig
	mu             sync.Mutex
	active         bool
	cancel         context.CancelFunc
	OnStatusChange func(profile string, strategy string, running bool, movements int64, duration, remaining time.Duration)
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

			logging.Info("registering cron job: expr=%q profile=%s duration=%s", cronExpr, p.Name, duration)
			_, err := s.cron.AddFunc(cronExpr, func() {
				logging.Info("cron triggered: profile=%s duration=%s", p.Name, duration)
				s.runForDuration(ctx, p, duration)
			})
			if err != nil {
				logging.Error("failed to register cron job: expr=%q error=%v", cronExpr, err)
			}
		}
	}

	s.cron.Start()
	logging.Info("cron scheduler running, %d entries registered", len(s.cron.Entries()))

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
	logging.Info("checking catch-up: current time=%s weekday=%s", now.Format("15:04:05"), now.Weekday())

	for _, profile := range s.cfg.Profiles {
		w, active := ActiveWindow(profile.Schedules, now)
		if active {
			remaining := RemainingDuration(w, now)
			if remaining > 0 {
				logging.Info("catch-up: active window found for profile=%s remaining=%s", profile.Name, remaining)
				p := profile
				go s.runForDuration(ctx, p, remaining)
				return nil
			}
		}
	}
	logging.Info("catch-up: no active windows found")
	return nil
}

func (s *Scheduler) runForDuration(ctx context.Context, profile config.Profile, duration time.Duration) {
	s.mu.Lock()
	if s.active {
		logging.Warn("runForDuration skipped: already active")
		s.mu.Unlock()
		return
	}
	s.active = true
	runCtx, cancel := context.WithTimeout(ctx, duration)
	s.cancel = cancel
	s.mu.Unlock()

	logging.Info("engine starting: profile=%s movement=%s duration=%s", profile.Name, profile.MovementType, duration)

	defer func() {
		cancel()
		s.mu.Lock()
		s.active = false
		s.cancel = nil
		s.mu.Unlock()
		if s.OnStatusChange != nil {
			s.OnStatusChange(profile.Name, "", false, 0, 0, 0)
		}
	}()

	var eng *engine.Engine
	if s.OnStatusChange != nil {
		eng = s.buildEngineWithCallback(profile, duration)
	} else {
		eng = s.buildEngine(profile)
	}

	eng.Start(runCtx)
}

func (s *Scheduler) buildStrategy(profile config.Profile) engine.MovementStrategy {
	switch profile.MovementType {
	case "recorded":
		if profile.Recording == "" {
			logging.Warn("profile %s has movement_type=recorded but no recording set, falling back to random", profile.Name)
			return &engine.RandomStrategy{MaxPixels: 15}
		}
		store, err := recording.NewStore(config.DBPath())
		if err != nil {
			logging.Error("opening recording store: %v, falling back to random", err)
			return &engine.RandomStrategy{MaxPixels: 15}
		}
		rec, err := store.Get(profile.Recording)
		store.Close()
		if err != nil {
			logging.Error("loading recording %q: %v, falling back to random", profile.Recording, err)
			return &engine.RandomStrategy{MaxPixels: 15}
		}
		logging.Info("loaded recording %q (%d points)", profile.Recording, len(rec.Points))
		return recording.NewPlayer(rec, true)
	case "simple":
		return &engine.SimpleStrategy{}
	default:
		return &engine.RandomStrategy{MaxPixels: 15}
	}
}

func (s *Scheduler) buildEngineWithCallback(profile config.Profile, duration time.Duration) *engine.Engine {
	strategy := s.buildStrategy(profile)

	opts := []engine.Option{
		engine.WithStrategy(strategy),
		engine.WithOnTick(func(info engine.TickInfo) {
			if s.OnStatusChange != nil {
				s.OnStatusChange(profile.Name, info.Strategy, true, info.Movements, duration, info.Remaining)
			}
		}),
	}
	if profile.Interval > 0 {
		opts = append(opts, engine.WithInterval(profile.Interval))
	}

	return engine.New(opts...)
}

func (s *Scheduler) buildEngine(profile config.Profile) *engine.Engine {
	strategy := s.buildStrategy(profile)

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
