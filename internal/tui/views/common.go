package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Surface    lipgloss.Color
	Text       lipgloss.Color
	TextMuted  lipgloss.Color
	Success    lipgloss.Color
	Error      lipgloss.Color
}

var DefaultTheme = Theme{
	Primary:    lipgloss.Color("#1a5fb4"),
	Secondary:  lipgloss.Color("#62a0ea"),
	Accent:     lipgloss.Color("#3584e4"),
	Background: lipgloss.Color("#1c1c2e"),
	Surface:    lipgloss.Color("#2a2a40"),
	Text:       lipgloss.Color("#ffffff"),
	TextMuted:  lipgloss.Color("#9090b0"),
	Success:    lipgloss.Color("#57e389"),
	Error:      lipgloss.Color("#ed333b"),
}

func (t Theme) BoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2)
}

func (t Theme) ActiveItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Primary).
		Padding(0, 1)
}

func (t Theme) ItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.TextMuted).
		Padding(0, 1)
}

func (t Theme) LabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true)
}

func (t Theme) ValueStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Text)
}

func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Success).Bold(true)
}

func (t Theme) ErrorMsgStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Error)
}

func (t Theme) HelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.TextMuted)
}

type ViewModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
	View() string
	Help() string
	ConsumesInput() bool
	Refresh() tea.Cmd
}

type ConfirmState int

const (
	ConfirmNone ConfirmState = iota
	ConfirmActive
	ConfirmYes
	ConfirmNo
)

type ConfirmModel struct {
	Message string
	State   ConfirmState
	focused bool
}

func NewConfirm(message string) ConfirmModel {
	return ConfirmModel{Message: message, State: ConfirmActive, focused: true}
}

func (c ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	if !c.focused {
		return c, nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
			c.State = ConfirmYes
			c.focused = false
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N", "esc"))):
			c.State = ConfirmNo
			c.focused = false
		}
	}
	return c, nil
}

func (c ConfirmModel) View() string {
	if c.State == ConfirmActive {
		return c.Message + " (y/n)"
	}
	return ""
}

func (c ConfirmModel) Active() bool {
	return c.State == ConfirmActive
}

func (c ConfirmModel) Confirmed() bool {
	return c.State == ConfirmYes
}
