// Package tui is the interactive wizard. Picking a layout lands on a review
// screen where every setting already holds a sensible default, so applying
// takes one key and editors are opened only for what needs changing.
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/monconf"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

// ── state machine ────────────────────────────────────────────────────────────

type tuiState int

// The wizard is two screens deep: pick a layout, then land on the review hub
// where every setting already has a sensible default. Editor states are only
// ever reached from the hub and always return to it, so there is no long
// chain of steps to walk back through.
const (
	tuiDetecting tuiState = iota
	tuiLayoutSelect
	tuiReview
	tuiModeSelect
	tuiTransformSelect
	tuiVRRSelect
	tuiDirectionSelect
	tuiOrderEdit
	tuiConfigView
	tuiProfileName
	tuiApplying
	tuiReloading
	tuiDone
	tuiErr
)

// editorStates are the states opened from the review hub; escape returns to it.
func isEditorState(s tuiState) bool {
	switch s {
	case tuiModeSelect, tuiTransformSelect, tuiVRRSelect, tuiDirectionSelect,
		tuiOrderEdit, tuiConfigView, tuiProfileName:
		return true
	}
	return false
}

// ── messages ─────────────────────────────────────────────────────────────────

type monDetectedMsg struct {
	monitors []hypr.Monitor
	target   monconf.Target
	err      error
}

type configWrittenMsg struct {
	result monconf.Result
	err    error
}

type reloadDoneMsg struct{ err error }

type profileSavedMsg struct{ err error }

// ── list item types ───────────────────────────────────────────────────────────

type layoutListItem struct{ opt layout.Option }

func (i layoutListItem) Title() string       { return i.opt.Name }
func (i layoutListItem) Description() string { return "" }
func (i layoutListItem) FilterValue() string { return i.opt.Name }

type modeListItem struct {
	mode layout.Mode
	// shortcut labels a strategy entry (Preferred / Highest resolution /
	// Highest refresh) that resolves to a concrete mode; empty for plain modes.
	shortcut string
	current  bool
}

func (i modeListItem) Title() string {
	if i.shortcut != "" {
		return i.shortcut + "  " + ui.Dimmed.Render(layout.FormatMode(i.mode))
	}
	s := layout.FormatMode(i.mode)
	if i.current {
		return s + "  " + ui.Dimmed.Render("current")
	}
	return s
}
func (i modeListItem) Description() string { return "" }
func (i modeListItem) FilterValue() string { return i.shortcut + layout.FormatMode(i.mode) }

type dirListItem struct {
	name string
	dir  layout.Direction
}

func (i dirListItem) Title() string       { return i.name }
func (i dirListItem) Description() string { return "" }
func (i dirListItem) FilterValue() string { return i.name }

// settingListItem is a generic labelled integer choice used by the transform
// and VRR selection screens.
type settingListItem struct {
	label   string
	value   int
	current bool
}

func (i settingListItem) Title() string {
	if i.current {
		return i.label + "  " + ui.Dimmed.Render("current")
	}
	return i.label
}
func (i settingListItem) Description() string { return "" }
func (i settingListItem) FilterValue() string { return i.label }

// ── key map ───────────────────────────────────────────────────────────────────

type tuiKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	MoveUp   key.Binding
	MoveDown key.Binding
	Apply    key.Binding
	Back     key.Binding
	Quit     key.Binding
}

var tuiKeys = tuiKeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	MoveUp:   key.NewBinding(key.WithKeys("shift+up", "K"), key.WithHelp("shift+↑", "move up")),
	MoveDown: key.NewBinding(key.WithKeys("shift+down", "J"), key.WithHelp("shift+↓", "move down")),
	Apply:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "apply")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// ── model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	state  tuiState
	width  int
	height int

	// pipeline data
	// target is the file the layout will be written to; the config view and
	// apply both use its format.
	target        monconf.Target
	monitors      []hypr.Monitor
	activeIndexes []int
	direction     layout.Direction
	activeConfigs []layout.MonitorConfig
	configLines   []string
	applyResult   monconf.Result
	mirrored      bool

	// review hub state
	reviewCursor int
	// editIdx is the activeConfigs entry an editor screen is acting on: the
	// monitor whose mode or rotation is being picked, or the row being moved
	// on the reorder screen.
	editIdx int
	// orderBackup restores the pre-edit order when reordering is cancelled.
	orderBackup []layout.MonitorConfig

	// components
	spinner   spinner.Model
	list      list.Model
	textInput textinput.Model
	viewport  viewport.Model

	// feedback
	err         error
	statusLines []string
	inputErr    string
}

func newTUIModel() tuiModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.Accent

	ti := textinput.New()
	ti.Prompt = ui.PromptGlyph.Render("❯ ")
	ti.CharLimit = 64
	ti.PlaceholderStyle = ui.Dimmed

	return tuiModel{
		state:     tuiDetecting,
		spinner:   s,
		textInput: ti,
	}
}

// ── init ──────────────────────────────────────────────────────────────────────

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			monitors, err := hypr.Detect()
			if err != nil {
				return monDetectedMsg{err: err}
			}
			target, err := monconf.DetectTarget()
			return monDetectedMsg{monitors: monitors, target: target, err: err}
		},
	)
}
