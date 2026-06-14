package views

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
	"keepalive/internal/engine"
	"keepalive/internal/recording"
)

type runState int

const (
	runStateIdle runState = iota
	runStateForm
	runStateRunning
	runStateConfirmStop
)

type RunView struct {
	theme         Theme
	state         runState
	status        *daemon.StatusInfo
	cfg           *config.AppConfig
	width         int
	profileIdx    int
	durationInput textinput.Model
	formField     int
	daemonMode    bool
	confirm       ConfirmModel
	err           string
	tickInfo      engine.TickInfo
	engineCancel  context.CancelFunc
}

func NewRunView(cfg *config.AppConfig) *RunView {
	di := textinput.New()
	di.Placeholder = "10m, 1h, or empty for indefinite"
	di.CharLimit = 20

	return &RunView{
		theme:         DefaultTheme,
		cfg:           cfg,
		durationInput: di,
	}
}

func (v *RunView) Init() tea.Cmd { return nil }

func (v *RunView) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		return v, nil

	case StatusUpdateMsg:
		v.status = msg.Info
		return v, nil

	case engineTickMsg:
		v.tickInfo = msg.Info
		return v, nil

	case engineDoneMsg:
		v.state = runStateIdle
		v.engineCancel = nil
		return v, nil

	case tea.KeyMsg:
		if v.confirm.Active() {
			v.confirm, _ = v.confirm.Update(msg)
			if v.confirm.Confirmed() {
				v.confirm.State = ConfirmNone
				return v, v.stopSession()
			}
			if v.confirm.State == ConfirmNo {
				v.confirm.State = ConfirmNone
				v.state = runStateIdle
			}
			return v, nil
		}
		return v.handleKey(msg)
	}

	return v, nil
}

func (v *RunView) handleKey(msg tea.KeyMsg) (ViewModel, tea.Cmd) {
	if v.confirm.Active() {
		return v, nil
	}

	switch v.state {
	case runStateIdle:
		switch msg.String() {
		case "s", "enter":
			v.state = runStateForm
			v.formField = 0
			v.profileIdx = 0
			v.daemonMode = false
			v.durationInput.Reset()
			v.durationInput.Focus()
			v.err = ""
			return v, v.durationInput.Cursor.BlinkCmd()
		case "x":
			if v.status != nil && v.status.Running {
				v.confirm = NewConfirm("Stop the running session?")
				v.state = runStateConfirmStop
			}
		}

	case runStateForm:
		switch msg.String() {
		case "esc":
			v.state = runStateIdle
			return v, nil
		case "tab", "down":
			v.formField = (v.formField + 1) % 3
			if v.formField == 0 {
				v.durationInput.Focus()
			} else {
				v.durationInput.Blur()
			}
			return v, nil
		case "shift+tab", "up":
			v.formField = (v.formField + 2) % 3
			if v.formField == 0 {
				v.durationInput.Focus()
			} else {
				v.durationInput.Blur()
			}
			return v, nil
		case "left":
			if v.formField == 1 && v.profileIdx > 0 {
				v.profileIdx--
			}
			return v, nil
		case "right":
			if v.formField == 1 && v.profileIdx < len(v.cfg.Profiles)-1 {
				v.profileIdx++
			}
			return v, nil
		case " ":
			if v.formField == 2 {
				v.daemonMode = !v.daemonMode
			}
			return v, nil
		case "enter":
			return v, v.startSession()
		default:
			if v.formField == 0 {
				var cmd tea.Cmd
				v.durationInput, cmd = v.durationInput.Update(msg)
				return v, cmd
			}
		}

	case runStateRunning:
		if msg.String() == "x" || msg.String() == "esc" {
			v.confirm = NewConfirm("Stop the running session?")
			v.state = runStateConfirmStop
		}
	}

	return v, nil
}

func (v *RunView) View() string {
	var b strings.Builder
	box := v.theme.BoxStyle()
	label := v.theme.LabelStyle()

	switch v.state {
	case runStateIdle:
		if v.status != nil && v.status.Running {
			b.WriteString(v.theme.SuccessStyle().Render("● Session Active"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("  Profile:    %s\n", v.status.Profile))
			b.WriteString(fmt.Sprintf("  Strategy:   %s\n", v.status.Strategy))
			b.WriteString(fmt.Sprintf("  Movements:  %d\n", v.status.Movements))
			b.WriteString(fmt.Sprintf("  Uptime:     %s\n", time.Since(v.status.StartedAt).Truncate(time.Second)))
			if v.status.Duration > 0 {
				b.WriteString(fmt.Sprintf("  Remaining:  %s\n", v.status.Remaining.Truncate(time.Second)))
			}
			b.WriteString("\n")
			b.WriteString(v.theme.HelpStyle().Render("  Press x to stop"))
		} else {
			b.WriteString(v.theme.HelpStyle().Render("○ No session running"))
			b.WriteString("\n\n")
			b.WriteString(v.theme.HelpStyle().Render("  Press s or Enter to start a new session"))
		}

	case runStateForm:
		b.WriteString(label.Render("Start Session"))
		b.WriteString("\n\n")

		durLabel := "  Duration: "
		if v.formField == 0 {
			durLabel = "▸ Duration: "
		}
		b.WriteString(durLabel + v.durationInput.View() + "\n")

		profLabel := "  Profile:  "
		if v.formField == 1 {
			profLabel = "▸ Profile:  "
		}
		profileName := v.cfg.Profiles[v.profileIdx].Name
		b.WriteString(fmt.Sprintf("%s◀ %s ▶\n", profLabel, profileName))

		daemonLabel := "  Daemon:   "
		if v.formField == 2 {
			daemonLabel = "▸ Daemon:   "
		}
		check := "[ ]"
		if v.daemonMode {
			check = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s%s run in background\n", daemonLabel, check))

		b.WriteString("\n")
		b.WriteString(v.theme.HelpStyle().Render("  Enter: start | Esc: cancel | Tab: next field | Space: toggle"))

		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case runStateRunning:
		b.WriteString(v.theme.SuccessStyle().Render("● Running (in-process)"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Strategy:   %s\n", v.tickInfo.Strategy))
		b.WriteString(fmt.Sprintf("  Movements:  %d\n", v.tickInfo.Movements))
		b.WriteString(fmt.Sprintf("  Uptime:     %s\n", v.tickInfo.TotalTime.Truncate(time.Second)))
		if v.tickInfo.Remaining >= 0 {
			b.WriteString(fmt.Sprintf("  Remaining:  %s\n", v.tickInfo.Remaining.Truncate(time.Second)))
		}
		b.WriteString("\n")
		b.WriteString(v.theme.HelpStyle().Render("  Press x or Esc to stop"))

	case runStateConfirmStop:
		b.WriteString(v.theme.ErrorMsgStyle().Render(v.confirm.View()))
	}

	return box.Render(b.String())
}

func (v *RunView) Help() string {
	switch v.state {
	case runStateForm:
		return "Tab/↑↓: navigate | ←→: change profile | Space: toggle | Enter: start | Esc: back"
	case runStateRunning:
		return "x: stop session"
	default:
		if v.status != nil && v.status.Running {
			return "x: stop | s: restart with different config"
		}
		return "s/Enter: start session"
	}
}

func (v *RunView) ConsumesInput() bool {
	return v.state == runStateForm || v.state == runStateRunning || v.confirm.Active()
}

func (v *RunView) Refresh() tea.Cmd { return nil }

func (v *RunView) startSession() tea.Cmd {
	profile := v.cfg.Profiles[v.profileIdx]

	var duration time.Duration
	if v.durationInput.Value() != "" {
		d, err := time.ParseDuration(v.durationInput.Value())
		if err != nil {
			v.err = fmt.Sprintf("Invalid duration: %v", err)
			return nil
		}
		duration = d
	}

	if v.daemonMode {
		binary, _ := os.Executable()
		args := []string{"run"}
		if duration > 0 {
			args = append(args, "--duration", duration.String())
		}
		args = append(args, "--config", profile.Name)

		pid, err := daemon.StartDetached(binary, args)
		if err != nil {
			v.err = fmt.Sprintf("Failed to start daemon: %v", err)
			return nil
		}
		v.state = runStateIdle
		v.err = ""
		_ = pid
		return nil
	}

	var opts []engine.Option
	opts = append(opts, engine.WithInterval(profile.Interval))
	if duration > 0 {
		opts = append(opts, engine.WithDuration(duration))
	}

	switch profile.MovementType {
	case "simple":
		opts = append(opts, engine.WithStrategy(&engine.SimpleStrategy{}))
	case "recorded":
		if profile.Recording != "" {
			store, err := recording.NewStore(config.DBPath())
			if err != nil {
				v.err = fmt.Sprintf("Cannot open recordings: %v", err)
				return nil
			}
			rec, err := store.Get(profile.Recording)
			store.Close()
			if err != nil {
				v.err = fmt.Sprintf("Recording %q not found", profile.Recording)
				return nil
			}
			opts = append(opts, engine.WithStrategy(recording.NewPlayer(rec, true)))
		} else {
			opts = append(opts, engine.WithStrategy(&engine.RandomStrategy{MaxPixels: 15}))
		}
	default:
		opts = append(opts, engine.WithStrategy(&engine.RandomStrategy{MaxPixels: 15}))
	}

	eng := engine.New(opts...)
	ctx, cancel := context.WithCancel(context.Background())
	v.engineCancel = cancel
	v.state = runStateRunning
	v.err = ""

	return func() tea.Msg {
		tickCh := make(chan engine.TickInfo, 1)
		opts = append(opts, engine.WithOnTick(func(info engine.TickInfo) {
			select {
			case tickCh <- info:
			default:
			}
		}))
		eng = engine.New(opts...)

		go func() {
			eng.Start(ctx)
		}()

		for {
			select {
			case <-ctx.Done():
				return engineDoneMsg{}
			case info := <-tickCh:
				return engineTickMsg{Info: info}
			}
		}
	}
}

func (v *RunView) stopSession() tea.Cmd {
	if v.engineCancel != nil {
		v.engineCancel()
		v.engineCancel = nil
		v.state = runStateIdle
		return nil
	}

	pidPath := config.PIDPath()
	socketPath := config.SocketPath()
	err := daemon.StopRunning(pidPath, socketPath)
	if err != nil {
		return func() tea.Msg {
			v.err = fmt.Sprintf("Failed to stop session: %v", err)
			return nil
		}
	}
	v.state = runStateIdle
	return nil
}

type engineTickMsg struct {
	Info engine.TickInfo
}

type engineDoneMsg struct{}

type RecordingListMsg struct {
	Recordings []recording.Recording
}
