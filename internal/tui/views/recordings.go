package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keepalive/internal/config"
	"keepalive/internal/recording"
)

type recState int

const (
	recStateList recState = iota
	recStateNameInput
	recStateRecording
	recStatePlaying
	recStateConfirmDelete
	recStateRename
)

type RecordingsView struct {
	theme       Theme
	state       recState
	recordings  []recording.Recording
	cursor      int
	nameInput   textinput.Model
	renameInput textinput.Model
	confirm     ConfirmModel
	recCancel   context.CancelFunc
	recorder    *recording.Recorder
	startTime   time.Time
	pointCount  int
	playCancel  context.CancelFunc
	err         string
	width       int
}

func NewRecordingsView() *RecordingsView {
	ni := textinput.New()
	ni.Placeholder = "recording name"
	ni.CharLimit = 30

	ri := textinput.New()
	ri.Placeholder = "new name"
	ri.CharLimit = 30

	return &RecordingsView{
		theme:       DefaultTheme,
		nameInput:   ni,
		renameInput: ri,
	}
}

func (v *RecordingsView) Init() tea.Cmd {
	return v.loadRecordings()
}

func (v *RecordingsView) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		return v, nil

	case RecordingListMsg:
		v.recordings = msg.Recordings
		if v.state == recStateRename {
			v.state = recStateList
			v.err = ""
		}
		return v, nil

	case recTickMsg:
		v.pointCount = msg.Points
		return v, nil

	case recDoneMsg:
		v.state = recStateList
		return v, v.loadRecordings()

	case playTickMsg:
		if v.state == recStatePlaying {
			if v.playCancel == nil {
				v.state = recStateList
				return v, nil
			}
			return v, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return playTickMsg{}
			})
		}
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
				v.state = recStateList
			}
			return v, nil
		}
		return v.handleKey(msg)
	}

	return v, nil
}

func (v *RecordingsView) handleKey(msg tea.KeyMsg) (ViewModel, tea.Cmd) {
	if v.confirm.Active() {
		return v, nil
	}

	switch v.state {
	case recStateList:
		switch msg.String() {
		case "n":
			v.state = recStateNameInput
			v.nameInput.Reset()
			v.nameInput.Focus()
			v.err = ""
			return v, v.nameInput.Cursor.BlinkCmd()
		case "r":
			if len(v.recordings) > 0 {
				v.state = recStateRename
				v.renameInput.Reset()
				v.renameInput.SetValue(v.recordings[v.cursor].Name)
				v.renameInput.Focus()
				v.err = ""
				return v, v.renameInput.Cursor.BlinkCmd()
			}
		case "p":
			if len(v.recordings) > 0 {
				return v, v.playSelected()
			}
		case "d":
			if len(v.recordings) > 0 {
				name := v.recordings[v.cursor].Name
				v.confirm = NewConfirm(fmt.Sprintf("Delete recording %q?", name))
				v.state = recStateConfirmDelete
			}
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.recordings)-1 {
				v.cursor++
			}
		}

	case recStateNameInput:
		switch msg.String() {
		case "esc":
			v.state = recStateList
			return v, nil
		case "enter":
			return v, v.startRecording()
		default:
			var cmd tea.Cmd
			v.nameInput, cmd = v.nameInput.Update(msg)
			return v, cmd
		}

	case recStateRecording:
		if msg.String() == "s" || msg.String() == "esc" || msg.String() == "enter" {
			return v, v.stopRecording()
		}

	case recStatePlaying:
		if msg.String() == "s" || msg.String() == "esc" || msg.String() == "enter" {
			if v.playCancel != nil {
				v.playCancel()
				v.playCancel = nil
			}
			v.state = recStateList
			return v, nil
		}

	case recStateRename:
		switch msg.String() {
		case "esc":
			v.state = recStateList
			return v, nil
		case "enter":
			return v, v.renameSelected()
		default:
			var cmd tea.Cmd
			v.renameInput, cmd = v.renameInput.Update(msg)
			return v, cmd
		}
	}

	return v, nil
}

func (v *RecordingsView) View() string {
	var b strings.Builder
	box := v.theme.BoxStyle()
	label := v.theme.LabelStyle()

	switch v.state {
	case recStateList:
		b.WriteString(label.Render("Recordings"))
		b.WriteString("\n\n")

		if len(v.recordings) == 0 {
			b.WriteString(v.theme.HelpStyle().Render("  No recordings yet. Press n to create one."))
		} else {
			for i, rec := range v.recordings {
				prefix := "  "
				style := v.theme.ItemStyle()
				if i == v.cursor {
					prefix = "▸ "
					style = v.theme.ActiveItemStyle()
				}
				line := fmt.Sprintf("%s%-18s %8s   %s",
					prefix, rec.Name,
					recording.FormatDuration(rec.DurationMs),
					rec.CreatedAt.Format("2006-01-02 15:04"))
				b.WriteString(style.Render(line) + "\n")
			}
		}
		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case recStateNameInput:
		b.WriteString(label.Render("New Recording"))
		b.WriteString("\n\n")
		b.WriteString("  Name: " + v.nameInput.View())
		b.WriteString("\n\n")
		b.WriteString(v.theme.HelpStyle().Render("  Enter: start recording | Esc: cancel"))
		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case recStateRecording:
		b.WriteString(v.theme.SuccessStyle().Render("● Recording..."))
		b.WriteString("\n\n")
		elapsed := time.Since(v.startTime).Truncate(time.Second)
		b.WriteString(fmt.Sprintf("  Elapsed:  %s\n", elapsed))
		b.WriteString(fmt.Sprintf("  Points:   %d\n", v.pointCount))
		b.WriteString("\n")
		b.WriteString(v.theme.HelpStyle().Render("  Move your mouse around. Press s/Enter/Esc to stop and save."))

	case recStatePlaying:
		name := ""
		if v.cursor < len(v.recordings) {
			name = v.recordings[v.cursor].Name
		}
		b.WriteString(v.theme.SuccessStyle().Render("▶ Playing: " + name))
		b.WriteString("\n\n")
		elapsed := time.Since(v.startTime).Truncate(time.Second)
		b.WriteString(fmt.Sprintf("  Elapsed:  %s\n", elapsed))
		b.WriteString("\n")
		b.WriteString(v.theme.HelpStyle().Render("  Press s/Enter/Esc to stop playback."))

	case recStateRename:
		b.WriteString(label.Render("Rename Recording"))
		b.WriteString("\n\n")
		b.WriteString("  New name: " + v.renameInput.View())
		b.WriteString("\n\n")
		b.WriteString(v.theme.HelpStyle().Render("  Enter: rename | Esc: cancel"))
		if v.err != "" {
			b.WriteString("\n" + v.theme.ErrorMsgStyle().Render("  "+v.err))
		}

	case recStateConfirmDelete:
		b.WriteString(v.theme.ErrorMsgStyle().Render(v.confirm.View()))
	}

	return box.Render(b.String())
}

func (v *RecordingsView) Help() string {
	switch v.state {
	case recStateNameInput:
		return "Enter: start | Esc: back"
	case recStateRecording:
		return "s/Enter/Esc: stop and save"
	case recStatePlaying:
		return "s/Enter/Esc: stop playback"
	case recStateRename:
		return "Enter: rename | Esc: cancel"
	default:
		return "n: new recording | r: rename | p: play | d: delete | ↑↓/jk: navigate"
	}
}

func (v *RecordingsView) ConsumesInput() bool {
	return v.state == recStateNameInput || v.state == recStateRecording || v.state == recStatePlaying || v.state == recStateRename || v.confirm.Active()
}

func (v *RecordingsView) Refresh() tea.Cmd {
	return v.loadRecordings()
}

func (v *RecordingsView) loadRecordings() tea.Cmd {
	return func() tea.Msg {
		store, err := recording.NewStore(config.DBPath())
		if err != nil {
			return RecordingListMsg{}
		}
		defer store.Close()

		recs, _ := store.List()
		return RecordingListMsg{Recordings: recs}
	}
}

func (v *RecordingsView) startRecording() tea.Cmd {
	name := strings.TrimSpace(v.nameInput.Value())
	if name == "" {
		v.err = "Name cannot be empty"
		return nil
	}

	store, err := recording.NewStore(config.DBPath())
	if err != nil {
		v.err = err.Error()
		return nil
	}
	if store.Exists(name) {
		store.Close()
		v.err = fmt.Sprintf("Recording %q already exists", name)
		return nil
	}
	store.Close()

	v.state = recStateRecording
	v.startTime = time.Now()
	v.pointCount = 0
	v.err = ""

	ctx, cancel := context.WithCancel(context.Background())
	v.recCancel = cancel
	v.recorder = recording.NewRecorder(50 * time.Millisecond)

	go v.recorder.Start(ctx)

	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return recTickMsg{Points: v.recorder.PointCount()}
	})
}

func (v *RecordingsView) stopRecording() tea.Cmd {
	if v.recCancel != nil {
		v.recCancel()
		v.recCancel = nil
	}

	name := strings.TrimSpace(v.nameInput.Value())
	result := v.recorder.Result(name)

	return func() tea.Msg {
		store, err := recording.NewStore(config.DBPath())
		if err != nil {
			return recDoneMsg{}
		}
		defer store.Close()

		if len(result.Points) > 0 {
			store.Save(result)
		}
		return recDoneMsg{}
	}
}

func (v *RecordingsView) playSelected() tea.Cmd {
	if v.cursor >= len(v.recordings) {
		return nil
	}
	name := v.recordings[v.cursor].Name

	store, err := recording.NewStore(config.DBPath())
	if err != nil {
		v.err = err.Error()
		return nil
	}
	defer store.Close()

	rec, err := store.Get(name)
	if err != nil {
		v.err = err.Error()
		return nil
	}

	v.state = recStatePlaying
	v.startTime = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	v.playCancel = cancel

	player := recording.NewPlayer(rec, false)
	go func() {
		player.Execute(ctx)
		cancel()
	}()

	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return playTickMsg{}
	})
}

func (v *RecordingsView) renameSelected() tea.Cmd {
	if v.cursor >= len(v.recordings) {
		return nil
	}
	oldName := v.recordings[v.cursor].Name
	newName := strings.TrimSpace(v.renameInput.Value())

	if newName == "" {
		v.err = "Name cannot be empty"
		return nil
	}
	if newName == oldName {
		v.state = recStateList
		return nil
	}

	return func() tea.Msg {
		store, err := recording.NewStore(config.DBPath())
		if err != nil {
			return RecordingListMsg{}
		}
		defer store.Close()

		if err := store.Rename(oldName, newName); err != nil {
			return RecordingListMsg{Recordings: nil}
		}

		// Update any profiles referencing the old name
		cfg, err := config.Load()
		if err == nil {
			updated := false
			for i := range cfg.Profiles {
				if cfg.Profiles[i].Recording == oldName {
					cfg.Profiles[i].Recording = newName
					updated = true
				}
			}
			if updated {
				config.Save(cfg)
			}
		}

		recs, _ := store.List()
		return RecordingListMsg{Recordings: recs}
	}
}

func (v *RecordingsView) deleteSelected() tea.Cmd {
	if v.cursor >= len(v.recordings) {
		return nil
	}
	name := v.recordings[v.cursor].Name

	cfg, err := config.Load()
	if err == nil {
		for _, p := range cfg.Profiles {
			if p.Recording == name {
				v.err = fmt.Sprintf("Recording %q is used by profile %q. Remove or change the profile first.", name, p.Name)
				v.state = recStateList
				return nil
			}
		}
	}

	return func() tea.Msg {
		store, err := recording.NewStore(config.DBPath())
		if err != nil {
			return RecordingListMsg{}
		}
		defer store.Close()

		store.Delete(name)
		recs, _ := store.List()
		return RecordingListMsg{Recordings: recs}
	}
}

type recTickMsg struct {
	Points int
}

type recDoneMsg struct{}

type playTickMsg struct{}
