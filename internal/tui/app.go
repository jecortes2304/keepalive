package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"keepalive/internal/config"
	"keepalive/internal/daemon"
	"keepalive/internal/tui/views"
)

type View int

const (
	ViewHome View = iota
	ViewRun
	ViewRecordings
	ViewSchedule
	ViewConfig
)

var viewNames = []string{"Home", "Run", "Recordings", "Schedule", "Config"}

type AppModel struct {
	currentView View
	width       int
	height      int
	theme       views.Theme
	cfg         *config.AppConfig
	quitting    bool

	homeView       views.ViewModel
	runView        views.ViewModel
	recordingsView views.ViewModel
	scheduleView   views.ViewModel
	configView     views.ViewModel
}

type statusTickMsg time.Time

func NewAppModel() AppModel {
	cfg, _ := config.Load()
	theme := views.DefaultTheme

	return AppModel{
		currentView:    ViewHome,
		theme:          theme,
		cfg:            cfg,
		homeView:       views.NewHomeView(),
		runView:        views.NewRunView(cfg),
		recordingsView: views.NewRecordingsView(),
		scheduleView:   views.NewScheduleView(cfg),
		configView:     views.NewConfigView(cfg),
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.recordingsView.Init(),
		pollStatus(),
		tickStatus(),
	)
}

func (m AppModel) activeView() views.ViewModel {
	switch m.currentView {
	case ViewHome:
		return m.homeView
	case ViewRun:
		return m.runView
	case ViewRecordings:
		return m.recordingsView
	case ViewSchedule:
		return m.scheduleView
	case ViewConfig:
		return m.configView
	}
	return m.homeView
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.activeView().ConsumesInput() && m.isGlobalKey(msg) {
			return m.handleGlobal(msg)
		}
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case statusTickMsg:
		return m, tea.Batch(pollStatus(), tickStatus())

	case views.StatusUpdateMsg:
		m.homeView.Update(msg)
		m.runView.Update(msg)
	}

	var cmd tea.Cmd
	switch m.currentView {
	case ViewHome:
		m.homeView, cmd = m.homeView.Update(msg)
	case ViewRun:
		m.runView, cmd = m.runView.Update(msg)
	case ViewRecordings:
		m.recordingsView, cmd = m.recordingsView.Update(msg)
	case ViewSchedule:
		m.scheduleView, cmd = m.scheduleView.Update(msg)
	case ViewConfig:
		m.configView, cmd = m.configView.Update(msg)
	}

	return m, cmd
}

func (m AppModel) isGlobalKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "ctrl+c":
		return true
	case "q":
		return true
	case "1", "2", "3", "4", "5":
		return true
	}
	return false
}

func (m AppModel) handleGlobal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("1"))):
		m.currentView = ViewHome
	case key.Matches(msg, key.NewBinding(key.WithKeys("2"))):
		m.currentView = ViewRun
	case key.Matches(msg, key.NewBinding(key.WithKeys("3"))):
		m.currentView = ViewRecordings
	case key.Matches(msg, key.NewBinding(key.WithKeys("4"))):
		m.currentView = ViewSchedule
	case key.Matches(msg, key.NewBinding(key.WithKeys("5"))):
		m.currentView = ViewConfig
	}
	return m, m.activeView().Refresh()
}

func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	switch m.currentView {
	case ViewHome:
		b.WriteString(m.homeView.View())
	case ViewRun:
		b.WriteString(m.runView.View())
	case ViewRecordings:
		b.WriteString(m.recordingsView.View())
	case ViewSchedule:
		b.WriteString(m.scheduleView.View())
	case ViewConfig:
		b.WriteString(m.configView.View())
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m AppModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(string(m.theme.Accent))).
		PaddingLeft(1).
		Render("keepalive")
	return title
}

func (m AppModel) renderTabs() string {
	var tabs []string
	for i, name := range viewNames {
		var s lipgloss.Style
		if View(i) == m.currentView {
			s = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(string(m.theme.Text))).
				Background(lipgloss.Color(string(m.theme.Primary))).
				Padding(0, 2)
		} else {
			s = lipgloss.NewStyle().
				Foreground(lipgloss.Color(string(m.theme.TextMuted))).
				Padding(0, 2)
		}
		tabs = append(tabs, s.Render(fmt.Sprintf("%d %s", i+1, name)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m AppModel) renderFooter() string {
	var help string
	switch m.currentView {
	case ViewHome:
		help = m.homeView.Help()
	case ViewRun:
		help = m.runView.Help()
	case ViewRecordings:
		help = m.recordingsView.Help()
	case ViewSchedule:
		help = m.scheduleView.Help()
	case ViewConfig:
		help = m.configView.Help()
	}

	global := "q: quit | 1-5: switch tab"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(string(m.theme.TextMuted))).
		Render(global + " | " + help)
}

func pollStatus() tea.Cmd {
	return func() tea.Msg {
		socketPath := config.SocketPath()
		client := daemon.NewClient(socketPath)
		info, err := client.GetStatus()
		if err != nil {
			return views.StatusUpdateMsg{Info: nil}
		}
		return views.StatusUpdateMsg{Info: info}
	}
}

func tickStatus() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
	})
}

func Run() error {
	p := tea.NewProgram(NewAppModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
