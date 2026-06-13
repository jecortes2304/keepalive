package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keepalive/internal/config"
)

type schedState int

const (
	schedStateList schedState = iota
	schedStateForm
	schedStateConfirmDelete
)

type scheduleEntry struct {
	ProfileName string
	Schedule    config.Schedule
	ProfileIdx  int
	SchedIdx    int
}

type ScheduleView struct {
	theme     Theme
	state     schedState
	cfg       *config.AppConfig
	entries   []scheduleEntry
	cursor    int
	confirm   ConfirmModel
	formField int
	days      [7]bool
	timeInput textinput.Model
	durInput  textinput.Model
	profIdx   int
	editing   bool
	editIdx   int
	err       string
	width     int
}

var dayLabels = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func NewScheduleView(cfg *config.AppConfig) *ScheduleView {
	ti := textinput.New()
	ti.Placeholder = "10:00"
	ti.CharLimit = 5

	di := textinput.New()
	di.Placeholder = "30m"
	di.CharLimit = 10

	v := &ScheduleView{
		theme:     DefaultTheme,
		cfg:       cfg,
		timeInput: ti,
		durInput:  di,
	}
	v.buildEntries()
	return v
}

func (v *ScheduleView) buildEntries() {
	v.entries = nil
	for pi, profile := range v.cfg.Profiles {
		for si, sched := range profile.Schedules {
			v.entries = append(v.entries, scheduleEntry{
				ProfileName: profile.Name,
				Schedule:    sched,
				ProfileIdx:  pi,
				SchedIdx:    si,
			})
		}
	}
}

func (v *ScheduleView) Init() tea.Cmd { return nil }

func (v *ScheduleView) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		return v, nil

	case configReloadMsg:
		v.cfg = msg.Cfg
		v.buildEntries()
		return v, nil

	case tea.KeyMsg:
		if v.confirm.Active() {
			v.confirm, _ = v.confirm.Update(msg)
			if v.confirm.Confirmed() {
				v.confirm.State = ConfirmNone
				return v, v.deleteSelected()
			}
			if v.confirm.State == ConfirmNo {
				v.confirm.State = ConfirmNone
				v.state = schedStateList
			}
			return v, nil
		}
		return v.handleKey(msg)
	}

	return v, nil
}

func (v *ScheduleView) handleKey(msg tea.KeyMsg) (ViewModel, tea.Cmd) {
	if v.confirm.Active() {
		return v, nil
	}

	switch v.state {
	case schedStateList:
		switch msg.String() {
		case "a":
			v.state = schedStateForm
			v.formField = 0
			v.days = [7]bool{}
			v.timeInput.Reset()
			v.durInput.Reset()
			v.profIdx = 0
			v.editing = false
			v.err = ""
		case "e":
			if len(v.entries) > 0 {
				v.editing = true
				v.editIdx = v.cursor
				e := v.entries[v.cursor]
				v.state = schedStateForm
				v.formField = 0
				// Pre-fill days
				v.days = [7]bool{}
				for _, d := range e.Schedule.Days {
					v.days[int(d)] = true
				}
				v.timeInput.Reset()
				v.timeInput.SetValue(e.Schedule.StartTime)
				v.durInput.Reset()
				v.durInput.SetValue(e.Schedule.Duration.String())
				// Set profile index
				v.profIdx = e.ProfileIdx
				v.err = ""
			}
		case "d":
			if len(v.entries) > 0 {
				e := v.entries[v.cursor]
				v.confirm = NewConfirm(fmt.Sprintf("Remove schedule at %s?", e.Schedule.StartTime))
				v.state = schedStateConfirmDelete
			}
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.entries)-1 {
				v.cursor++
			}
		}

	case schedStateForm:
		switch msg.String() {
		case "esc":
			v.state = schedStateList
			return v, nil
		case "tab":
			v.blurAll()
			v.formField = (v.formField + 1) % 4
			v.focusCurrent()
			return v, nil
		case "shift+tab":
			v.blurAll()
			v.formField = (v.formField + 3) % 4
			v.focusCurrent()
			return v, nil
		case "enter":
			return v, v.saveSchedule()
		case "left":
			if v.formField == 3 && v.profIdx > 0 {
				v.profIdx--
			}
			return v, nil
		case "right":
			if v.formField == 3 && v.profIdx < len(v.cfg.Profiles)-1 {
				v.profIdx++
			}
			return v, nil
		default:
			if v.formField == 0 {
				// Days field: toggle with 0-6
				if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '6' {
					idx := int(msg.String()[0] - '0')
					v.days[idx] = !v.days[idx]
				}
				return v, nil
			}
			if v.formField == 1 {
				var cmd tea.Cmd
				v.timeInput, cmd = v.timeInput.Update(msg)
				return v, cmd
			}
			if v.formField == 2 {
				var cmd tea.Cmd
				v.durInput, cmd = v.durInput.Update(msg)
				return v, cmd
			}
		}
	}

	return v, nil
}

func (v *ScheduleView) blurAll() {
	v.timeInput.Blur()
	v.durInput.Blur()
}

func (v *ScheduleView) focusCurrent() {
	switch v.formField {
	case 1:
		v.timeInput.Focus()
	case 2:
		v.durInput.Focus()
	}
}

func (v *ScheduleView) View() string {
	var b strings.Builder
	box := v.theme.BoxStyle()
	label := v.theme.LabelStyle()

	switch v.state {
	case schedStateList:
		b.WriteString(label.Render("Schedules"))
		b.WriteString("\n\n")

		if len(v.entries) == 0 {
			b.WriteString(v.theme.HelpStyle().Render("  No schedules configured. Press a to add one."))
		} else {
			for i, e := range v.entries {
				prefix := "  "
				style := v.theme.ItemStyle()
				if i == v.cursor {
					prefix = "▸ "
					style = v.theme.ActiveItemStyle()
				}
				days := formatDays(e.Schedule.Days)
				line := fmt.Sprintf("%s%s at %s for %s (%s)",
					prefix, days, e.Schedule.StartTime, e.Schedule.Duration, e.ProfileName)
				b.WriteString(style.Render(line) + "\n")
			}
		}

	case schedStateForm:
		if v.editing {
			b.WriteString(label.Render("Edit Schedule"))
		} else {
			b.WriteString(label.Render("Add Schedule"))
		}
		b.WriteString("\n\n")

		daysPrefix := "  "
		if v.formField == 0 {
			daysPrefix = "▸ "
		}
		b.WriteString(daysPrefix + "Days: ")
		for i, d := range dayLabels {
			if v.days[i] {
				b.WriteString(v.theme.SuccessStyle().Render("["+d+"]") + " ")
			} else {
				b.WriteString(v.theme.HelpStyle().Render(" "+d+" ") + " ")
			}
		}
		b.WriteString("\n")
		b.WriteString(v.theme.HelpStyle().Render("         Press 0-6 to toggle (0=Sun, 1=Mon, ... 6=Sat)"))
		b.WriteString("\n\n")

		timePrefix := "  "
		if v.formField == 1 {
			timePrefix = "▸ "
		}
		b.WriteString(timePrefix + "Start: " + v.timeInput.View() + "\n")

		durPrefix := "  "
		if v.formField == 2 {
			durPrefix = "▸ "
		}
		b.WriteString(durPrefix + "Duration: " + v.durInput.View() + "\n")

		profPrefix := "  "
		if v.formField == 3 {
			profPrefix = "▸ "
		}
		profileName := v.cfg.Profiles[v.profIdx].Name
		b.WriteString(fmt.Sprintf("%sProfile: ◀ %s ▶\n", profPrefix, profileName))

		b.WriteString("\n")
		b.WriteString(v.theme.HelpStyle().Render("  Enter: save | Esc: cancel | Tab: next field"))
		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case schedStateConfirmDelete:
		b.WriteString(v.theme.ErrorMsgStyle().Render(v.confirm.View()))
	}

	return box.Render(b.String())
}

func (v *ScheduleView) Help() string {
	switch v.state {
	case schedStateForm:
		return "0-6: toggle day | Tab: next field | Enter: save | Esc: cancel"
	default:
		return "a: add schedule | e: edit | d: delete | ↑↓/jk: navigate"
	}
}

func (v *ScheduleView) ConsumesInput() bool {
	return v.state == schedStateForm || v.confirm.Active()
}

func (v *ScheduleView) Refresh() tea.Cmd { return nil }

func (v *ScheduleView) saveSchedule() tea.Cmd {
	var days []time.Weekday
	for i, selected := range v.days {
		if selected {
			days = append(days, time.Weekday(i))
		}
	}
	if len(days) == 0 {
		v.err = "Select at least one day"
		return nil
	}

	startTime := strings.TrimSpace(v.timeInput.Value())
	if _, err := time.Parse("15:04", startTime); err != nil {
		v.err = "Invalid time format (use HH:MM)"
		return nil
	}

	durStr := strings.TrimSpace(v.durInput.Value())
	duration, err := time.ParseDuration(durStr)
	if err != nil {
		v.err = "Invalid duration (e.g. 30m, 1h)"
		return nil
	}

	sched := config.Schedule{
		Days:      days,
		StartTime: startTime,
		Duration:  duration,
	}

	if v.editing {
		e := v.entries[v.editIdx]
		if v.profIdx == e.ProfileIdx {
			// Same profile, just update in place
			v.cfg.Profiles[e.ProfileIdx].Schedules[e.SchedIdx] = sched
		} else {
			// Moving to different profile: remove from old, add to new
			profile := &v.cfg.Profiles[e.ProfileIdx]
			profile.Schedules = append(profile.Schedules[:e.SchedIdx], profile.Schedules[e.SchedIdx+1:]...)
			v.cfg.Profiles[v.profIdx].Schedules = append(v.cfg.Profiles[v.profIdx].Schedules, sched)
		}
	} else {
		v.cfg.Profiles[v.profIdx].Schedules = append(v.cfg.Profiles[v.profIdx].Schedules, sched)
	}

	if err := config.Save(v.cfg); err != nil {
		v.err = err.Error()
		return nil
	}

	v.buildEntries()
	v.state = schedStateList
	v.editing = false
	v.err = ""
	return nil
}

func (v *ScheduleView) deleteSelected() tea.Cmd {
	if v.cursor >= len(v.entries) {
		return nil
	}
	e := v.entries[v.cursor]
	profile := &v.cfg.Profiles[e.ProfileIdx]
	profile.Schedules = append(profile.Schedules[:e.SchedIdx], profile.Schedules[e.SchedIdx+1:]...)
	config.Save(v.cfg)
	v.buildEntries()
	if v.cursor >= len(v.entries) && v.cursor > 0 {
		v.cursor--
	}
	v.state = schedStateList
	return nil
}

func formatDays(days []time.Weekday) string {
	names := make([]string, len(days))
	for i, d := range days {
		names[i] = d.String()[:3]
	}
	return strings.Join(names, ",")
}

type configReloadMsg struct {
	Cfg *config.AppConfig
}
