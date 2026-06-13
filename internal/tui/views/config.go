package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keepalive/internal/config"
	"keepalive/internal/recording"
)

type cfgState int

const (
	cfgStateList cfgState = iota
	cfgStateForm
	cfgStateConfirmDelete
)

type ConfigView struct {
	theme      Theme
	state      cfgState
	cfg        *config.AppConfig
	cursor     int
	confirm    ConfirmModel
	formField  int
	nameInput  textinput.Model
	intInput   textinput.Model
	durInput   textinput.Model
	moveIdx    int
	recIdx     int
	recordings []recording.Recording
	editing    bool
	editIdx    int
	err        string
	width      int
}

var movementTypes = []string{"random", "simple", "recorded"}

func NewConfigView(cfg *config.AppConfig) *ConfigView {
	ni := textinput.New()
	ni.Placeholder = "profile name"
	ni.CharLimit = 20

	ii := textinput.New()
	ii.Placeholder = "30s"
	ii.CharLimit = 10

	di := textinput.New()
	di.Placeholder = "0s (indefinite)"
	di.CharLimit = 10

	return &ConfigView{
		theme:     DefaultTheme,
		cfg:       cfg,
		nameInput: ni,
		intInput:  ii,
		durInput:  di,
	}
}

func (v *ConfigView) Init() tea.Cmd { return nil }

func (v *ConfigView) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		return v, nil

	case tea.KeyMsg:
		if v.confirm.Active() {
			v.confirm, _ = v.confirm.Update(msg)
			if v.confirm.Confirmed() {
				v.confirm.State = ConfirmNone
				v.deleteSelected()
			}
			if v.confirm.State == ConfirmNo {
				v.confirm.State = ConfirmNone
				v.state = cfgStateList
			}
			return v, nil
		}
		return v.handleKey(msg)
	}

	return v, nil
}

func (v *ConfigView) handleKey(msg tea.KeyMsg) (ViewModel, tea.Cmd) {
	if v.confirm.Active() {
		return v, nil
	}

	switch v.state {
	case cfgStateList:
		switch msg.String() {
		case "n":
			v.state = cfgStateForm
			v.formField = 0
			v.nameInput.Reset()
			v.intInput.Reset()
			v.durInput.Reset()
			v.moveIdx = 0
			v.recIdx = 0
			v.editing = false
			v.loadRecordings()
			v.nameInput.Focus()
			v.err = ""
			return v, v.nameInput.Cursor.BlinkCmd()
		case "e":
			if len(v.cfg.Profiles) > 0 {
				v.editing = true
				v.editIdx = v.cursor
				p := v.cfg.Profiles[v.cursor]
				v.state = cfgStateForm
				v.formField = 0
				v.nameInput.Reset()
				v.nameInput.SetValue(p.Name)
				v.intInput.Reset()
				v.intInput.SetValue(p.Interval.String())
				v.durInput.Reset()
				if p.Duration > 0 {
					v.durInput.SetValue(p.Duration.String())
				}
				// Set movement type index
				v.moveIdx = 0
				for i, mt := range movementTypes {
					if mt == p.MovementType {
						v.moveIdx = i
						break
					}
				}
				v.loadRecordings()
				// Set recording index
				v.recIdx = 0
				for i, rec := range v.recordings {
					if rec.Name == p.Recording {
						v.recIdx = i
						break
					}
				}
				v.nameInput.Focus()
				v.err = ""
				return v, v.nameInput.Cursor.BlinkCmd()
			}
		case "d":
			if len(v.cfg.Profiles) > 0 {
				name := v.cfg.Profiles[v.cursor].Name
				if name == v.cfg.DefaultProfile {
					v.err = "Cannot delete the default profile"
				} else {
					v.confirm = NewConfirm(fmt.Sprintf("Delete profile %q?", name))
					v.state = cfgStateConfirmDelete
				}
			}
		case "*":
			if len(v.cfg.Profiles) > 0 {
				v.cfg.DefaultProfile = v.cfg.Profiles[v.cursor].Name
				config.Save(v.cfg)
			}
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.cfg.Profiles)-1 {
				v.cursor++
			}
		}

	case cfgStateForm:
		numFields := v.formFieldCount()
		switch msg.String() {
		case "esc":
			v.state = cfgStateList
			return v, nil
		case "tab":
			v.blurAll()
			v.formField = (v.formField + 1) % numFields
			v.focusCurrent()
			return v, nil
		case "shift+tab":
			v.blurAll()
			v.formField = (v.formField + numFields - 1) % numFields
			v.focusCurrent()
			return v, nil
		case "left":
			if v.formField == 3 && v.moveIdx > 0 {
				v.moveIdx--
			}
			if v.formField == 4 && v.recIdx > 0 {
				v.recIdx--
			}
			return v, nil
		case "right":
			if v.formField == 3 && v.moveIdx < len(movementTypes)-1 {
				v.moveIdx++
			}
			if v.formField == 4 && v.recIdx < len(v.recordings)-1 {
				v.recIdx++
			}
			return v, nil
		case "enter":
			return v, v.saveProfile()
		default:
			var cmd tea.Cmd
			switch v.formField {
			case 0:
				v.nameInput, cmd = v.nameInput.Update(msg)
			case 1:
				v.intInput, cmd = v.intInput.Update(msg)
			case 2:
				v.durInput, cmd = v.durInput.Update(msg)
			}
			return v, cmd
		}
	}

	return v, nil
}

func (v *ConfigView) formFieldCount() int {
	if movementTypes[v.moveIdx] == "recorded" {
		return 5
	}
	return 4
}

func (v *ConfigView) loadRecordings() {
	store, err := recording.NewStore(config.DBPath())
	if err != nil {
		return
	}
	defer store.Close()
	v.recordings, _ = store.List()
}

func (v *ConfigView) blurAll() {
	v.nameInput.Blur()
	v.intInput.Blur()
	v.durInput.Blur()
}

func (v *ConfigView) focusCurrent() {
	switch v.formField {
	case 0:
		v.nameInput.Focus()
	case 1:
		v.intInput.Focus()
	case 2:
		v.durInput.Focus()
	}
}

func (v *ConfigView) View() string {
	var b strings.Builder
	box := v.theme.BoxStyle()
	label := v.theme.LabelStyle()

	switch v.state {
	case cfgStateList:
		b.WriteString(label.Render("Profiles"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %-14s %-10s %-10s %-10s\n",
			"NAME", "INTERVAL", "DURATION", "MOVEMENT"))

		for i, p := range v.cfg.Profiles {
			prefix := "  "
			style := v.theme.ItemStyle()
			if i == v.cursor {
				prefix = "▸ "
				style = v.theme.ActiveItemStyle()
			}
			def := ""
			if p.Name == v.cfg.DefaultProfile {
				def = " *"
			}
			dur := "infinite"
			if p.Duration > 0 {
				dur = p.Duration.String()
			}
			line := fmt.Sprintf("%s%-14s %-10s %-10s %-10s%s",
				prefix, p.Name, p.Interval, dur, p.MovementType, def)
			b.WriteString(style.Render(line) + "\n")
		}

		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case cfgStateForm:
		if v.editing {
			b.WriteString(label.Render("Edit Profile"))
		} else {
			b.WriteString(label.Render("New Profile"))
		}
		b.WriteString("\n\n")

		namePrefix := "  "
		if v.formField == 0 {
			namePrefix = "▸ "
		}
		b.WriteString(namePrefix + "Name:     " + v.nameInput.View() + "\n")

		intPrefix := "  "
		if v.formField == 1 {
			intPrefix = "▸ "
		}
		b.WriteString(intPrefix + "Interval: " + v.intInput.View() + "\n")

		durPrefix := "  "
		if v.formField == 2 {
			durPrefix = "▸ "
		}
		b.WriteString(durPrefix + "Duration: " + v.durInput.View() + "\n")

		movePrefix := "  "
		if v.formField == 3 {
			movePrefix = "▸ "
		}
		b.WriteString(fmt.Sprintf("%sMovement: ◀ %s ▶\n", movePrefix, movementTypes[v.moveIdx]))

		if movementTypes[v.moveIdx] == "recorded" {
			recPrefix := "  "
			if v.formField == 4 {
				recPrefix = "▸ "
			}
			recName := "(none)"
			if len(v.recordings) > 0 && v.recIdx < len(v.recordings) {
				recName = v.recordings[v.recIdx].Name
			}
			b.WriteString(fmt.Sprintf("%sRecording: ◀ %s ▶\n", recPrefix, recName))
		}

		b.WriteString("\n")
		if v.editing {
			b.WriteString(v.theme.HelpStyle().Render("  Enter: save | Esc: cancel | Tab: next field | ←→: change"))
		} else {
			b.WriteString(v.theme.HelpStyle().Render("  Enter: create | Esc: cancel | Tab: next field | ←→: change"))
		}
		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case cfgStateConfirmDelete:
		b.WriteString(v.theme.ErrorMsgStyle().Render(v.confirm.View()))
	}

	return box.Render(b.String())
}

func (v *ConfigView) Help() string {
	switch v.state {
	case cfgStateForm:
		if v.editing {
			return "Tab: next field | ←→: change type | Enter: save | Esc: cancel"
		}
		return "Tab: next field | ←→: change type | Enter: create | Esc: cancel"
	default:
		return "n: new profile | e: edit | d: delete | *: set default | ↑↓/jk: navigate"
	}
}

func (v *ConfigView) ConsumesInput() bool {
	return v.state == cfgStateForm || v.confirm.Active()
}

func (v *ConfigView) Refresh() tea.Cmd { return nil }

func (v *ConfigView) saveProfile() tea.Cmd {
	name := strings.TrimSpace(v.nameInput.Value())
	if name == "" {
		v.err = "Name cannot be empty"
		return nil
	}

	intStr := strings.TrimSpace(v.intInput.Value())
	if intStr == "" {
		intStr = "30s"
	}
	interval, err := time.ParseDuration(intStr)
	if err != nil {
		v.err = "Invalid interval format"
		return nil
	}

	durStr := strings.TrimSpace(v.durInput.Value())
	var duration time.Duration
	if durStr != "" && durStr != "0s" {
		d, err := time.ParseDuration(durStr)
		if err != nil {
			v.err = "Invalid duration format"
			return nil
		}
		duration = d
	}

	recName := ""
	if movementTypes[v.moveIdx] == "recorded" && len(v.recordings) > 0 && v.recIdx < len(v.recordings) {
		recName = v.recordings[v.recIdx].Name
	}

	if v.editing {
		// Check name conflict if name changed
		oldName := v.cfg.Profiles[v.editIdx].Name
		if name != oldName {
			for i, p := range v.cfg.Profiles {
				if i != v.editIdx && p.Name == name {
					v.err = fmt.Sprintf("Profile %q already exists", name)
					return nil
				}
			}
			// Update default profile reference if needed
			if v.cfg.DefaultProfile == oldName {
				v.cfg.DefaultProfile = name
			}
		}

		v.cfg.Profiles[v.editIdx].Name = name
		v.cfg.Profiles[v.editIdx].Interval = interval
		v.cfg.Profiles[v.editIdx].Duration = duration
		v.cfg.Profiles[v.editIdx].MovementType = movementTypes[v.moveIdx]
		v.cfg.Profiles[v.editIdx].Recording = recName
	} else {
		profile := config.Profile{
			Name:         name,
			Interval:     interval,
			Duration:     duration,
			MovementType: movementTypes[v.moveIdx],
			Recording:    recName,
		}

		if err := v.cfg.AddProfile(profile); err != nil {
			v.err = err.Error()
			return nil
		}
	}

	if err := config.Save(v.cfg); err != nil {
		v.err = err.Error()
		return nil
	}

	v.state = cfgStateList
	v.editing = false
	v.err = ""
	return nil
}

func (v *ConfigView) deleteSelected() {
	if v.cursor >= len(v.cfg.Profiles) {
		return
	}
	name := v.cfg.Profiles[v.cursor].Name
	v.cfg.DeleteProfile(name)
	config.Save(v.cfg)
	if v.cursor >= len(v.cfg.Profiles) && v.cursor > 0 {
		v.cursor--
	}
	v.state = cfgStateList
}
