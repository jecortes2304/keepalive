package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"keepalive/internal/daemon"
)

type HomeView struct {
	theme  Theme
	status *daemon.StatusInfo
	width  int
}

func NewHomeView() *HomeView {
	return &HomeView{theme: DefaultTheme}
}

func (v *HomeView) Init() tea.Cmd { return nil }

func (v *HomeView) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
	case StatusUpdateMsg:
		v.status = msg.Info
	}
	return v, nil
}

func (v *HomeView) View() string {
	var b strings.Builder
	box := v.theme.BoxStyle()
	label := v.theme.LabelStyle()
	value := v.theme.ValueStyle()

	b.WriteString(label.Render("Status"))
	b.WriteString("\n\n")

	if v.status != nil && v.status.Running {
		b.WriteString(v.theme.SuccessStyle().Render("● RUNNING"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s  %s\n", label.Render("PID:"), value.Render(fmt.Sprintf("%d", v.status.PID))))
		b.WriteString(fmt.Sprintf("  %s  %s\n", label.Render("Profile:"), value.Render(v.status.Profile)))
		b.WriteString(fmt.Sprintf("  %s  %s\n", label.Render("Strategy:"), value.Render(v.status.Strategy)))
		b.WriteString(fmt.Sprintf("  %s  %s\n", label.Render("Movements:"), value.Render(fmt.Sprintf("%d", v.status.Movements))))
		b.WriteString(fmt.Sprintf("  %s  %s\n", label.Render("Uptime:"), value.Render(time.Since(v.status.StartedAt).Truncate(time.Second).String())))
		if v.status.Duration > 0 {
			b.WriteString(fmt.Sprintf("  %s  %s\n", label.Render("Remaining:"), value.Render(v.status.Remaining.Truncate(time.Second).String())))
		}
	} else {
		b.WriteString(v.theme.HelpStyle().Render("○ NOT RUNNING"))
		b.WriteString("\n\n")
		b.WriteString(v.theme.HelpStyle().Render("  Press 2 to go to Run tab and start a session"))
	}

	return box.Render(b.String())
}

func (v *HomeView) Help() string {
	return "Navigation only — go to Run tab to start/stop"
}

func (v *HomeView) ConsumesInput() bool {
	return false
}

func (v *HomeView) Refresh() tea.Cmd { return nil }

type StatusUpdateMsg struct {
	Info *daemon.StatusInfo
}
