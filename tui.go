package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	case tuiModeSelect, tuiTransformSelect, tuiDirectionSelect,
		tuiOrderEdit, tuiConfigView, tuiProfileName:
		return true
	}
	return false
}

// ── messages ─────────────────────────────────────────────────────────────────

type monDetectedMsg struct {
	monitors []monitor
	err      error
}

type configWrittenMsg struct {
	result applyResult
	err    error
}

type reloadDoneMsg struct{ err error }

type profileSavedMsg struct{ err error }

// ── list item types ───────────────────────────────────────────────────────────

type layoutListItem struct{ opt layoutOption }

func (i layoutListItem) Title() string       { return i.opt.Name }
func (i layoutListItem) Description() string { return "" }
func (i layoutListItem) FilterValue() string { return i.opt.Name }

type modeListItem struct {
	mode monitorMode
	// shortcut labels a strategy entry (Preferred / Highest resolution /
	// Highest refresh) that resolves to a concrete mode; empty for plain modes.
	shortcut string
	current  bool
}

func (i modeListItem) Title() string {
	if i.shortcut != "" {
		return i.shortcut + "  " + styleDimmed.Render(formatMonitorMode(i.mode))
	}
	s := formatMonitorMode(i.mode)
	if i.current {
		return s + "  " + styleDimmed.Render("current")
	}
	return s
}
func (i modeListItem) Description() string { return "" }
func (i modeListItem) FilterValue() string { return i.shortcut + formatMonitorMode(i.mode) }

type dirListItem struct {
	name string
	dir  layoutDirection
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
		return i.label + "  " + styleDimmed.Render("current")
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
	Rotate   key.Binding
	VRR      key.Binding
	Apply    key.Binding
	Write    key.Binding
	Save     key.Binding
	Config   key.Binding
	Back     key.Binding
	Quit     key.Binding
}

var tuiKeys = tuiKeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	MoveUp:   key.NewBinding(key.WithKeys("shift+up", "K"), key.WithHelp("shift+↑", "move up")),
	MoveDown: key.NewBinding(key.WithKeys("shift+down", "J"), key.WithHelp("shift+↓", "move down")),
	Rotate:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rotate")),
	VRR:      key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "vrr")),
	Apply:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "apply")),
	Write:    key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "write only")),
	Save:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save profile")),
	Config:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "view config")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// ── model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	state  tuiState
	width  int
	height int

	// pipeline data
	monitors      []monitor
	activeIndexes []int
	direction     layoutDirection
	activeConfigs []activeMonitorConfig
	configLines   []string
	applyResult   applyResult
	mirrored      bool

	// review hub state
	reviewCursor int
	// editIdx is the activeConfigs entry an editor screen is acting on: the
	// monitor whose mode or rotation is being picked, or the row being moved
	// on the reorder screen.
	editIdx int
	// orderBackup restores the pre-edit order when reordering is cancelled.
	orderBackup []activeMonitorConfig
	// reloadAfterWrite distinguishes "apply" from "write only".
	reloadAfterWrite bool

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
	s.Style = styleAccent

	ti := textinput.New()
	ti.Prompt = stylePromptGlyph.Render("❯ ")
	ti.CharLimit = 64
	ti.PlaceholderStyle = styleDimmed

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
			monitors, err := detectMonitors()
			return monDetectedMsg{monitors: monitors, err: err}
		},
	)
}

// ── update ────────────────────────────────────────────────────────────────────

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.resizeComponents()
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, tuiKeys.Quit) && m.state != tuiApplying && m.state != tuiReloading {
			return m, tea.Quit
		}
		if key.Matches(msg, tuiKeys.Back) && backAllowed(m.state) {
			return m.goBack()
		}

	case spinner.TickMsg:
		if m.state == tuiDetecting || m.state == tuiApplying || m.state == tuiReloading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case monDetectedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = tuiErr
			return m, nil
		}
		if len(msg.monitors) == 0 {
			m.err = fmt.Errorf("no monitors detected")
			m.state = tuiErr
			return m, nil
		}
		m.monitors = msg.monitors
		m.state = tuiLayoutSelect
		m.list = m.makeLayoutList()
		return m, nil

	case configWrittenMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = tuiErr
			return m, nil
		}
		m.applyResult = msg.result
		m.statusLines = append(m.statusLines,
			styleSuccess.Render("✓")+"  Config written to "+styleMuted.Render(msg.result.ConfigPath),
		)
		if msg.result.BackupPath != "" {
			m.statusLines = append(m.statusLines,
				styleDimmed.Render("  Backup: "+msg.result.BackupPath),
			)
		}
		if !m.reloadAfterWrite {
			m.statusLines = append(m.statusLines,
				styleDimmed.Render("  Run ")+styleAccent.Render("hyprctl reload")+styleDimmed.Render(" to apply."),
			)
			m.state = tuiDone
			return m, nil
		}
		return m.startReload()

	case reloadDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = tuiErr
			return m, nil
		}
		m.statusLines = append(m.statusLines,
			styleSuccess.Render("✓")+"  Hyprland reloaded.",
		)
		m.state = tuiDone
		return m, nil

	case profileSavedMsg:
		// A failed save keeps the name screen open so the name can be fixed;
		// a successful one drops straight back to the hub.
		if msg.err != nil {
			m.inputErr = "Failed to save profile: " + msg.err.Error()
			return m, nil
		}
		m.textInput.Blur()
		m.statusLines = append(m.statusLines,
			styleSuccess.Render("✓")+"  Profile saved.",
		)
		m.state = tuiReview
		return m, nil
	}

	switch m.state {
	case tuiLayoutSelect:
		return m.updateLayoutSelect(msg)
	case tuiReview:
		return m.updateReview(msg)
	case tuiModeSelect:
		return m.updateModeSelect(msg)
	case tuiTransformSelect:
		return m.updateTransformSelect(msg)
	case tuiDirectionSelect:
		return m.updateDirectionSelect(msg)
	case tuiOrderEdit:
		return m.updateOrderEdit(msg)
	case tuiConfigView:
		return m.updateConfigView(msg)
	case tuiProfileName:
		return m.updateProfileName(msg)
	case tuiDone, tuiErr:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	return m, nil
}

// ── state update helpers ──────────────────────────────────────────────────────

// defaultModeSelections picks each active monitor's starting mode: the highest
// refresh rate available, matching the `quick` command's default. Every
// setting on the review hub starts from a default like this, so nothing has to
// be chosen before the layout can be applied.
func defaultModeSelections(monitors []monitor, activeIndexes []int) map[int]monitorMode {
	modes := make(map[int]monitorMode, len(activeIndexes))
	for _, idx := range activeIndexes {
		modes[idx] = resolveModeStrategy(monitors[idx], modeStrategyHighrr)
	}
	return modes
}

func (m tuiModel) updateLayoutSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(layoutListItem)
		if !ok {
			return m, nil
		}
		if item.opt.ID == layoutQuit {
			return m, tea.Quit
		}
		indexes, err := activeIndexesForLayout(m.monitors, item.opt.ID)
		if err != nil {
			m.err = err
			m.state = tuiErr
			return m, nil
		}
		m.activeIndexes = indexes
		m.mirrored = item.opt.ID == layoutMirror
		m.direction = leftToRight
		m.activeConfigs = buildActiveMonitorConfigs(m.monitors, indexes, defaultModeSelections(m.monitors, indexes))
		m.reviewCursor = 0
		m.state = tuiReview
		m = m.resizeComponents()
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// ── review hub ────────────────────────────────────────────────────────────────

type reviewRowKind int

const (
	reviewRowMonitor reviewRowKind = iota
	reviewRowDirection
	reviewRowOrder
)

type reviewRow struct {
	kind reviewRowKind
	// idx is the activeConfigs entry a monitor row refers to.
	idx int
}

// reviewRows lists the hub's editable rows. Mirrored layouts stack every
// monitor on the same source, so they have no direction or order to set, and
// a single monitor has nothing to reorder.
func (m tuiModel) reviewRows() []reviewRow {
	rows := make([]reviewRow, 0, len(m.activeConfigs)+2)
	for i := range m.activeConfigs {
		rows = append(rows, reviewRow{kind: reviewRowMonitor, idx: i})
	}
	if !m.mirrored {
		rows = append(rows, reviewRow{kind: reviewRowDirection})
		if len(m.activeConfigs) > 1 {
			rows = append(rows, reviewRow{kind: reviewRowOrder})
		}
	}
	return rows
}

func (m tuiModel) selectedReviewRow() (reviewRow, bool) {
	rows := m.reviewRows()
	if m.reviewCursor < 0 || m.reviewCursor >= len(rows) {
		return reviewRow{}, false
	}
	return rows[m.reviewCursor], true
}

func (m tuiModel) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.Up):
		if m.reviewCursor > 0 {
			m.reviewCursor--
		}
	case key.Matches(km, tuiKeys.Down):
		if m.reviewCursor < len(m.reviewRows())-1 {
			m.reviewCursor++
		}
	case key.Matches(km, tuiKeys.Enter):
		return m.openReviewEditor()
	case key.Matches(km, tuiKeys.Rotate):
		return m.openTransformEditor()
	case key.Matches(km, tuiKeys.VRR):
		return m.cycleVRR()
	case key.Matches(km, tuiKeys.Config):
		return m.openConfigView()
	case key.Matches(km, tuiKeys.Save):
		return m.openProfileName()
	case key.Matches(km, tuiKeys.Apply):
		return m.startApply(true)
	case key.Matches(km, tuiKeys.Write):
		return m.startApply(false)
	}
	return m, nil
}

// openReviewEditor opens the editor for the highlighted row: mode for a
// monitor, direction or reorder for the layout rows.
func (m tuiModel) openReviewEditor() (tea.Model, tea.Cmd) {
	row, ok := m.selectedReviewRow()
	if !ok {
		return m, nil
	}
	switch row.kind {
	case reviewRowMonitor:
		mon := m.monitors[m.activeConfigs[row.idx].Index]
		modes := availableMonitorModes(mon)
		if len(modes) == 0 {
			return m, nil
		}
		m.editIdx = row.idx
		m.state = tuiModeSelect
		m.list = m.makeModeList(mon, modes)
		m.list.Select(modeListIdx(m.activeConfigs[row.idx].Mode, modes))
		return m, nil

	case reviewRowDirection:
		m.state = tuiDirectionSelect
		m.list = m.makeDirectionList()
		m.list.Select(directionListIdx(m.direction))
		return m, nil

	case reviewRowOrder:
		m.orderBackup = append([]activeMonitorConfig(nil), m.activeConfigs...)
		m.editIdx = 0
		m.state = tuiOrderEdit
		return m, nil
	}
	return m, nil
}

// openTransformEditor opens the rotation list for the highlighted monitor.
func (m tuiModel) openTransformEditor() (tea.Model, tea.Cmd) {
	row, ok := m.selectedReviewRow()
	if !ok || row.kind != reviewRowMonitor {
		return m, nil
	}
	m.editIdx = row.idx
	m.state = tuiTransformSelect
	m.list = m.makeTransformList(m.activeConfigs[row.idx].Transform)
	return m, nil
}

// cycleVRR steps the highlighted monitor through off → on → fullscreen. VRR
// has only three values and no spatial effect, so it is cycled in place rather
// than costing a screen of its own.
func (m tuiModel) cycleVRR() (tea.Model, tea.Cmd) {
	row, ok := m.selectedReviewRow()
	if !ok || row.kind != reviewRowMonitor {
		return m, nil
	}
	m.activeConfigs[row.idx].VRR = (m.activeConfigs[row.idx].VRR + 1) % 3
	return m, nil
}

func (m tuiModel) openConfigView() (tea.Model, tea.Cmd) {
	m.configLines = m.buildConfigLines()
	m.state = tuiConfigView
	m = m.resizeComponents()
	m.viewport.SetContent(strings.Join(m.configLines, "\n"))
	return m, nil
}

func (m tuiModel) openProfileName() (tea.Model, tea.Cmd) {
	m.state = tuiProfileName
	m.textInput.SetValue("")
	m.textInput.Placeholder = "my-profile"
	m.inputErr = ""
	m.textInput.Focus()
	return m, textinput.Blink
}

// backToReview returns from an editor screen to the hub.
func (m tuiModel) backToReview() tuiModel {
	m.state = tuiReview
	m.inputErr = ""
	m.textInput.Blur()
	return m
}

// buildConfigLines renders the config for the current review state.
func (m tuiModel) buildConfigLines() []string {
	if m.mirrored {
		return renderMirroredConfigLines(m.monitors, m.activeConfigs, mirrorSourceIndex(m.monitors, m.activeIndexes))
	}
	return renderPositionedConfigLines(m.monitors, m.activeConfigs, m.direction)
}

// ── editor screens ────────────────────────────────────────────────────────────

func (m tuiModel) updateModeSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(modeListItem)
		if !ok {
			return m, nil
		}
		m.activeConfigs[m.editIdx].Mode = item.mode
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) updateTransformSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(settingListItem)
		if !ok {
			return m, nil
		}
		m.activeConfigs[m.editIdx].Transform = item.value
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) updateDirectionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(dirListItem)
		if !ok {
			return m, nil
		}
		m.direction = item.dir
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateOrderEdit moves the highlighted monitor through the order with
// shift+↑/↓, redrawing the live preview on every move.
func (m tuiModel) updateOrderEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.MoveUp):
		if m.editIdx > 0 {
			m.activeConfigs[m.editIdx-1], m.activeConfigs[m.editIdx] = m.activeConfigs[m.editIdx], m.activeConfigs[m.editIdx-1]
			m.editIdx--
		}
	case key.Matches(km, tuiKeys.MoveDown):
		if m.editIdx < len(m.activeConfigs)-1 {
			m.activeConfigs[m.editIdx+1], m.activeConfigs[m.editIdx] = m.activeConfigs[m.editIdx], m.activeConfigs[m.editIdx+1]
			m.editIdx++
		}
	case key.Matches(km, tuiKeys.Up):
		if m.editIdx > 0 {
			m.editIdx--
		}
	case key.Matches(km, tuiKeys.Down):
		if m.editIdx < len(m.activeConfigs)-1 {
			m.editIdx++
		}
	case key.Matches(km, tuiKeys.Enter):
		m.orderBackup = nil
		return m.backToReview(), nil
	}
	return m, nil
}

func (m tuiModel) updateConfigView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok &&
		(key.Matches(km, tuiKeys.Enter) || key.Matches(km, tuiKeys.Config)) {
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m tuiModel) updateProfileName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		name := strings.TrimSpace(m.textInput.Value())
		if err := validateProfileName(name); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		lines := m.buildConfigLines()
		meta := m.profileMetadata()
		return m, func() tea.Msg {
			dir, err := defaultProfilesDir()
			if err != nil {
				return profileSavedMsg{err: err}
			}
			return profileSavedMsg{err: saveProfile(dir, name, lines, meta)}
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// ── apply ─────────────────────────────────────────────────────────────────────

// startApply writes the config, then reloads Hyprland unless the user chose to
// write only. There is no confirmation step: the previous file is backed up
// first and a failed reload rolls back automatically.
func (m tuiModel) startApply(reload bool) (tea.Model, tea.Cmd) {
	m.reloadAfterWrite = reload
	m.configLines = m.buildConfigLines()
	m.state = tuiApplying
	lines := m.configLines
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			path, err := defaultMonitorConfigPath()
			if err != nil {
				return configWrittenMsg{err: err}
			}
			result, err := applyMonitorConfig(path, lines, time.Now())
			return configWrittenMsg{result: result, err: err}
		},
	)
}

func (m tuiModel) startReload() (tea.Model, tea.Cmd) {
	m.state = tuiReloading
	result := m.applyResult
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := reloadHyprland(ctx, systemCommandRunner); err != nil {
				if rbErr := rollbackMonitorConfig(result); rbErr != nil {
					return reloadDoneMsg{err: fmt.Errorf("reload failed: %v; rollback failed: %w", err, rbErr)}
				}
				if result.BackupPath != "" {
					return reloadDoneMsg{err: fmt.Errorf("reload failed: %v; restored backup from %s", err, result.BackupPath)}
				}
				return reloadDoneMsg{err: fmt.Errorf("reload failed: %v; removed newly created config", err)}
			}
			return reloadDoneMsg{}
		},
	)
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if m.width == 0 {
		return ""
	}
	if m.width < 40 || m.height < 10 {
		return "  Terminal too small — resize to continue.\n"
	}
	// MaxHeight clips any component quirk (e.g. the list paginator drawing one
	// line beyond its set height) so the footer is never pushed off-screen.
	contentH := m.height - 2
	body := lipgloss.NewStyle().Height(contentH).MaxHeight(contentH).Render(m.bodyView())
	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		body,
		m.footerView(),
	)
}

func (m tuiModel) headerView() string {
	appName := styleAccent.Copy().Bold(true).Render("hypr-layout")
	step := styleMuted.Render(m.stepLabel())
	inner := "  " + appName + "  " + styleDimmed.Render("·") + "  " + step
	return m.barLine(inner)
}

func (m tuiModel) footerView() string {
	return m.barLine("  " + m.footerHints())
}

// barLine pads a header/footer line to the full width and truncates it to a
// single line so narrow terminals never wrap it onto a second row.
func (m tuiModel) barLine(inner string) string {
	pad := m.width - lipgloss.Width(inner)
	if pad > 0 {
		inner += strings.Repeat(" ", pad)
	}
	return styleBarBg.Copy().MaxWidth(m.width).MaxHeight(1).Render(inner)
}

func (m tuiModel) stepLabel() string {
	switch m.state {
	case tuiDetecting:
		return "Detecting monitors"
	case tuiLayoutSelect:
		return "Select layout"
	case tuiReview:
		return "Review layout"
	case tuiModeSelect:
		if mon, ok := m.editedMonitor(); ok {
			return "Mode for " + mon.Name
		}
		return "Select mode"
	case tuiTransformSelect:
		if mon, ok := m.editedMonitor(); ok {
			return "Rotation for " + mon.Name
		}
		return "Rotation"
	case tuiDirectionSelect:
		return "Layout direction"
	case tuiOrderEdit:
		return "Monitor order"
	case tuiConfigView:
		return "Generated config"
	case tuiProfileName:
		return "Profile name"
	case tuiApplying:
		return "Writing config…"
	case tuiReloading:
		return "Reloading…"
	case tuiDone:
		return "Done"
	case tuiErr:
		return "Error"
	default:
		return ""
	}
}

func (m tuiModel) footerHints() string {
	dim := func(s string) string { return styleDimmed.Render(s) }
	acc := func(s string) string { return styleAccent.Render(s) }
	sep := styleDimmed.Render("  ·  ")
	back := acc("esc") + " back"
	switch m.state {
	case tuiLayoutSelect:
		return dim("↑/↓") + " navigate" + sep + acc("enter") + " select" + sep + acc("q") + " quit"
	case tuiReview:
		return m.reviewHints(sep)
	case tuiModeSelect, tuiTransformSelect, tuiDirectionSelect:
		return dim("↑/↓") + " navigate" + sep + acc("enter") + " select" + sep + back + sep + acc("q") + " quit"
	case tuiOrderEdit:
		return dim("↑/↓") + " select" + sep + acc("shift+↑/↓") + " move" + sep +
			acc("enter") + " done" + sep + acc("esc") + " cancel"
	case tuiConfigView:
		return dim("↑/↓") + " scroll" + sep + back + sep + acc("q") + " quit"
	case tuiProfileName:
		return acc("enter") + " save" + sep + back + sep + acc("q") + " quit"
	case tuiDone, tuiErr:
		return dim("any key") + " exit"
	default:
		return ""
	}
}

// reviewHints tailors the hub footer to the highlighted row, so the rotation
// and VRR shortcuts only advertise themselves on the rows they act on. The hub
// has more shortcuts than a narrow terminal can show, so the less essential
// ones drop off rather than being truncated mid-word.
func (m tuiModel) reviewHints(sep string) string {
	acc := func(s string) string { return styleAccent.Render(s) }

	always := []string{acc("enter") + " edit", acc("a") + " apply", acc("q") + " quit"}
	optional := []string{acc("s") + " save", acc("c") + " config", acc("w") + " write"}
	if row, ok := m.selectedReviewRow(); ok && row.kind == reviewRowMonitor {
		optional = append([]string{acc("r") + " rotate", acc("v") + " vrr"}, optional...)
	}
	optional = append(optional, styleDimmed.Render("↑/↓")+" move")

	// Keep "quit" last while dropping optional hints from the least useful end.
	for n := len(optional); n >= 0; n-- {
		parts := append(append([]string{}, always[:len(always)-1]...), optional[:n]...)
		parts = append(parts, always[len(always)-1])
		hints := strings.Join(parts, sep)
		if n == 0 || lipgloss.Width(hints)+2 <= m.width {
			return hints
		}
	}
	return ""
}

func (m tuiModel) bodyView() string {
	contentH := m.height - 2 // one line each for header and footer
	switch m.state {
	case tuiDetecting:
		return m.spinnerView(contentH, "Detecting monitors…")
	case tuiLayoutSelect:
		return m.layoutSelectView(contentH)
	case tuiReview:
		return m.reviewView(contentH)
	case tuiModeSelect:
		return m.modeSelectView(contentH)
	case tuiTransformSelect:
		return m.transformSelectView(contentH)
	case tuiDirectionSelect:
		return m.directionSelectView(contentH)
	case tuiOrderEdit:
		return m.orderEditView(contentH)
	case tuiConfigView:
		return m.configView(contentH)
	case tuiProfileName:
		return m.profileNameView(contentH)
	case tuiApplying:
		return m.spinnerView(contentH, "Writing config…")
	case tuiReloading:
		return m.spinnerView(contentH, "Reloading Hyprland…")
	case tuiDone:
		return m.doneView(contentH)
	case tuiErr:
		return m.errView(contentH)
	default:
		return ""
	}
}

// profileMetadata captures how the current layout was produced for the
// profile header: direction (or "mirror"), and the ordered monitors with
// their chosen modes.
func (m tuiModel) profileMetadata() profileMetadata {
	meta := profileMetadata{
		SavedAt:   time.Now(),
		Direction: string(m.direction),
	}
	if m.mirrored {
		meta.Direction = "mirror"
	}
	for _, cfg := range m.activeConfigs {
		meta.Monitors = append(meta.Monitors, m.monitors[cfg.Index].Name+" "+formatMonitorMode(cfg.Mode))
	}
	return meta
}

// editedMonitor returns the monitor an editor screen is acting on.
func (m tuiModel) editedMonitor() (monitor, bool) {
	if m.editIdx < 0 || m.editIdx >= len(m.activeConfigs) {
		return monitor{}, false
	}
	return m.monitors[m.activeConfigs[m.editIdx].Index], true
}

// ── body sub-views ────────────────────────────────────────────────────────────

func (m tuiModel) spinnerView(h int, label string) string {
	// Small logo + spinner, shown during detect/apply/reload
	var b strings.Builder
	b.WriteString("\n\n")
	if m.state == tuiDetecting {
		b.WriteString("  " + styleDimmed.Copy().Bold(true).Render("◆ hypr-layout") + "\n")
		b.WriteString("  " + styleDimmed.Render("monitor layout manager") + "\n")
		b.WriteString("\n")
	}
	b.WriteString("  " + m.spinner.View() + "  " + styleBase.Render(label) + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

// appendPreviewSection appends a live arrangement preview beneath body when
// enough vertical room remains for it; otherwise body is returned unchanged.
// The remaining room is measured from the rendered body so every screen fits
// the terminal without per-screen line accounting. Mirror layouts render a
// source box with a caption instead of positioned boxes.
func (m tuiModel) appendPreviewSection(body string, configs []activeMonitorConfig, direction layoutDirection, mirrored bool, sourceIdx int) string {
	body = strings.TrimRight(body, "\n")
	if len(configs) == 0 {
		return body
	}
	contentH := m.height - 2
	// 3 lines of chrome around the diagram: blank, "Preview" title, blank.
	availH := contentH - lipgloss.Height(body) - 3
	previewW := m.width - 6
	if previewW < 24 {
		previewW = 24
	}
	var preview string
	if mirrored {
		preview = renderMirrorPreview(m.monitors, configs, sourceIdx, previewW, availH)
	} else {
		preview = renderDirectionPreview(m.monitors, configs, direction, previewW, availH)
	}
	if preview == "" {
		return body
	}
	return body + "\n\n  " + styleTitle.Render("Preview") + "\n\n" + indentBlock(preview, 2)
}

// mirrorSource returns the source monitor index for the given active indexes
// when the mirror layout is in play.
func (m tuiModel) mirrorSource(activeIndexes []int) int {
	if len(activeIndexes) == 0 {
		return 0
	}
	return mirrorSourceIndex(m.monitors, activeIndexes)
}

// transformLabels name the eight Hyprland transform values: 0-3 rotate
// counter-clockwise in 90° steps, 4-7 are the same rotations of a flipped
// (mirrored) image.
var transformLabels = []string{
	"Normal",
	"90°",
	"180°",
	"270°",
	"Flipped",
	"Flipped + 90°",
	"Flipped + 180°",
	"Flipped + 270°",
}

func transformLabel(value int) string {
	if value < 0 || value >= len(transformLabels) {
		return "Normal"
	}
	return transformLabels[value]
}

func vrrLabel(value int) string {
	switch value {
	case 1:
		return "vrr on"
	case 2:
		return "vrr fullscreen"
	default:
		return "vrr off"
	}
}

// reviewView renders the hub: every monitor with its settings, then the
// layout-wide direction and order, over a live preview of the arrangement.
// Each row is already filled in with a default, so applying takes one key.
func (m tuiModel) reviewView(h int) string {
	var b strings.Builder
	b.WriteString("\n")

	rows := m.reviewRows()
	nameW := m.reviewNameWidth()
	for i, row := range rows {
		selected := i == m.reviewCursor
		b.WriteString(m.reviewRowLine(row, selected, nameW))
		// Separate the monitor rows from the layout-wide rows below them.
		if row.kind == reviewRowMonitor && i+1 < len(rows) && rows[i+1].kind != reviewRowMonitor {
			b.WriteString("\n")
		}
	}

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	return lipgloss.NewStyle().Height(h).Render(body)
}

// reviewNameWidth sizes the label column so monitor names and the Direction /
// Order labels line up in one column.
func (m tuiModel) reviewNameWidth() int {
	w := len("Direction")
	for _, cfg := range m.activeConfigs {
		if n := len(m.monitors[cfg.Index].Name); n > w {
			w = n
		}
	}
	return w
}

func (m tuiModel) reviewRowLine(row reviewRow, selected bool, nameW int) string {
	var label, value string
	switch row.kind {
	case reviewRowMonitor:
		cfg := m.activeConfigs[row.idx]
		label = m.monitors[cfg.Index].Name
		value = fmt.Sprintf("%-16s  %-14s  %s",
			formatMonitorMode(cfg.Mode), transformLabel(cfg.Transform), vrrLabel(cfg.VRR))
	case reviewRowDirection:
		label = "Direction"
		value = directionLabel(m.direction)
	case reviewRowOrder:
		label = "Order"
		value = m.orderSummary()
	}

	marker := "  "
	labelStyle := styleBase
	if selected {
		marker = styleAccent.Render("▸ ")
		labelStyle = styleSelected
	}
	return "  " + marker + labelStyle.Render(fmt.Sprintf("%-*s", nameW, label)) +
		"  " + styleDimmed.Render(value) + "\n"
}

// orderSummary describes the current monitor order as a single line.
func (m tuiModel) orderSummary() string {
	names := make([]string, 0, len(m.activeConfigs))
	for _, cfg := range m.activeConfigs {
		names = append(names, m.monitors[cfg.Index].Name)
	}
	return strings.Join(names, " → ")
}

func directionLabel(d layoutDirection) string {
	switch d {
	case rightToLeft:
		return "Right → left"
	case topToBottom:
		return "Top → bottom"
	case bottomToTop:
		return "Bottom → top"
	}
	return "Left → right"
}

func (m tuiModel) directionSelectView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Select layout direction") + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	// Live preview: recalculate and redraw for whichever item is highlighted.
	if item, ok := m.list.SelectedItem().(dirListItem); ok {
		body = m.appendPreviewSection(body, m.activeConfigs, item.dir, false, 0)
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

func (m tuiModel) layoutSelectView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Detected monitors") + "\n\n")
	for _, mon := range m.monitors {
		focused := "  " + styleDimmed.Render("○")
		if mon.Focused {
			focused = "  " + styleSuccess.Render("●")
		}
		b.WriteString(fmt.Sprintf("%s  %-14s %s\n",
			focused,
			styleBase.Render(mon.Name),
			styleDimmed.Render(fmt.Sprintf("%dx%d @ %sHz", mon.Width, mon.Height, formatFloat(mon.RefreshRate))),
		))
	}
	b.WriteString("\n  " + styleTitle.Render("Select a layout") + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	// Live preview of which monitors the highlighted layout would activate.
	if item, ok := m.list.SelectedItem().(layoutListItem); ok && item.opt.ID != layoutQuit {
		indexes, err := activeIndexesForLayout(m.monitors, item.opt.ID)
		if err != nil {
			body = strings.TrimRight(body, "\n") + "\n\n  " + styleDimmed.Render("Not available: "+err.Error())
		} else {
			configs := buildActiveMonitorConfigs(m.monitors, indexes, nil)
			mirrored := item.opt.ID == layoutMirror
			body = m.appendPreviewSection(body, configs, leftToRight, mirrored, m.mirrorSource(indexes))
		}
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

// modeSelectView shows the mode list with a live preview of the arrangement
// using the highlighted mode for the monitor being configured.
func (m tuiModel) modeSelectView(h int) string {
	title := "Select mode"
	if mon, ok := m.editedMonitor(); ok {
		title = "Select mode for " + mon.Name
	}

	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render(title) + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	if item, ok := m.list.SelectedItem().(modeListItem); ok && m.editIdx < len(m.activeConfigs) {
		configs := make([]activeMonitorConfig, len(m.activeConfigs))
		copy(configs, m.activeConfigs)
		configs[m.editIdx].Mode = item.mode
		body = m.appendPreviewSection(body, configs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

// transformSelectView shows the rotation list with a live preview: hovering a
// 90°/270° transform visibly swaps the monitor's box proportions.
func (m tuiModel) transformSelectView(h int) string {
	title := "Select rotation"
	if mon, ok := m.editedMonitor(); ok {
		title = "Select rotation for " + mon.Name
	}

	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render(title) + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	if item, ok := m.list.SelectedItem().(settingListItem); ok && m.editIdx < len(m.activeConfigs) {
		configs := make([]activeMonitorConfig, len(m.activeConfigs))
		copy(configs, m.activeConfigs)
		configs[m.editIdx].Transform = item.value
		body = m.appendPreviewSection(body, configs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

// orderEditView lists the monitors in their current order with the one being
// moved marked, and redraws the arrangement preview after every move.
func (m tuiModel) orderEditView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Reorder monitors") + "\n\n")
	for i, cfg := range m.activeConfigs {
		mon := m.monitors[cfg.Index]
		name := styleBase.Render(fmt.Sprintf("%-12s", mon.Name))
		mode := styleDimmed.Render(formatMonitorMode(cfg.Mode))
		if i == m.editIdx {
			b.WriteString("  " + styleSelected.Render("▸ "+fmt.Sprintf("%-12s", mon.Name)) + "  " + mode + "  " + styleAccent.Render("↕") + "\n")
			continue
		}
		b.WriteString("    " + name + "  " + mode + "\n")
	}

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, false, 0)
	return lipgloss.NewStyle().Height(h).Render(body)
}

// configView shows the exact lines that will be written to monitors.conf.
func (m tuiModel) configView(h int) string {
	inner := m.viewport.View()
	boxWidth := m.width - 6
	if boxWidth < 10 {
		boxWidth = 10
	}
	box := stylePreviewBox.Copy().Width(boxWidth).Render(inner)

	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Will be written to monitors.conf") + "\n\n")
	b.WriteString("  " + box + "\n")

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	return lipgloss.NewStyle().Height(h).Render(body)
}

func (m tuiModel) profileNameView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Profile name") + "\n\n")
	b.WriteString("  " + styleMuted.Render("Letters, numbers, dashes, underscores only.") + "\n\n")
	b.WriteString("  " + m.textInput.View() + "\n")
	if m.inputErr != "" {
		b.WriteString("\n  " + styleErr.Render(m.inputErr) + "\n")
	}
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) doneView(h int) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, line := range m.statusLines {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n  " + styleDimmed.Render("Press any key to exit.") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) errView(h int) string {
	msg := "An unknown error occurred."
	if m.err != nil {
		msg = m.err.Error()
	}
	boxWidth := m.width - 8
	if boxWidth < 10 {
		boxWidth = 10
	}
	box := styleErrorBox.Copy().Width(boxWidth).Render(
		styleErr.Copy().Bold(true).Render("Error") + "\n\n" + styleBase.Render(msg),
	)
	var b strings.Builder
	b.WriteString("\n\n  " + box + "\n\n")
	b.WriteString("  " + styleDimmed.Render("Press any key to exit.") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

// ── back navigation ───────────────────────────────────────────────────────────

// backAllowed returns true for states where pressing Escape navigates back.
// Editor screens return to the review hub and the hub returns to the layout
// list; states where a side-effect is in flight (writing, reloading) or where
// there is nothing behind them (detecting, done, error) do not allow it.
func backAllowed(s tuiState) bool {
	return s == tuiReview || isEditorState(s)
}

func (m tuiModel) goBack() (tuiModel, tea.Cmd) {
	if m.state == tuiReview {
		m.state = tuiLayoutSelect
		m.list = m.makeLayoutList()
		return m, nil
	}
	// Reordering mutates the live order as it goes, so cancelling has to put
	// the pre-edit order back.
	if m.state == tuiOrderEdit && m.orderBackup != nil {
		m.activeConfigs = m.orderBackup
		m.orderBackup = nil
	}
	return m.backToReview(), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m tuiModel) resizeComponents() tuiModel {
	if m.width == 0 || m.height == 0 {
		return m
	}
	contentH := m.height - 2
	// Only resize the list when it has been initialised; calling SetSize on a
	// zero-value list.Model panics because its internal paginator is nil.
	// Lists are kept compact so the live preview has room below them.
	switch m.state {
	case tuiLayoutSelect:
		m.list.SetSize(m.width-4, m.clampListHeight(layoutListHeight, len(m.monitors)+3))
	case tuiModeSelect:
		m.list.SetSize(m.width-4, m.modeListHeight(len(m.list.Items())))
	case tuiTransformSelect:
		m.list.SetSize(m.width-4, m.clampListHeight(transformListHeight, 0))
	case tuiDirectionSelect:
		m.list.SetSize(m.width-4, m.clampListHeight(directionListHeight, 0))
	}

	vpW := m.width - 10
	vpH := contentH - 8
	// The config lines are short; capping the viewport to its content leaves
	// room for the arrangement preview below the config box.
	if n := len(m.configLines); n > 0 && vpH > n {
		vpH = n
	}
	if vpW < 4 {
		vpW = 4
	}
	if vpH < 2 {
		vpH = 2
	}
	m.viewport.Width = vpW
	m.viewport.Height = vpH
	return m
}

func (m tuiModel) makeLayoutList() list.Model {
	opts := layoutOptions()
	items := make([]list.Item, 0, len(opts))
	for _, opt := range opts {
		items = append(items, layoutListItem{opt: opt})
	}
	l := m.newStyledList(items)
	l.SetSize(m.width-4, m.clampListHeight(layoutListHeight, len(m.monitors)+3))
	return l
}

// modeShortcutCount is the number of strategy shortcut entries prepended to
// every TUI mode list; modeListIdx offsets restored cursors by this amount.
const modeShortcutCount = 3

// Fixed list heights: item count + 1 spare row, which keeps the bubbles
// paginator hidden so every option is visible at once. Compact lists leave
// room for the live preview below them.
const (
	layoutListHeight    = 7 // 6 layout options
	transformListHeight = 9 // 8 transforms
)

// previewReserve is the vertical room kept free below variable-height lists
// (the mode list) so the live preview fits: 3 lines of chrome plus a 5-line
// row of boxes.
const previewReserve = 8

// modeListHeight caps the mode list so it never swallows the space reserved
// for the preview, while long mode lists still paginate.
func (m tuiModel) modeListHeight(itemCount int) int {
	maxH := m.height - 2 - 4 - previewReserve
	if maxH > itemCount+1 {
		maxH = itemCount + 1
	}
	if maxH < 4 {
		maxH = 4
	}
	return maxH
}

// clampListHeight bounds a fixed list height so the list plus its title
// chrome always fits the terminal (lists paginate when shrunk). extraLines
// accounts for content rendered above the list beyond the standard 3 title
// lines, e.g. the detected-monitors block on the layout screen.
func (m tuiModel) clampListHeight(desired, extraLines int) int {
	maxH := m.height - 2 - 3 - extraLines
	if desired > maxH {
		// A clamped list paginates, and the bubbles paginator renders one
		// line beyond the set height — shrink once more to absorb it.
		desired = maxH - 1
	}
	if desired < 2 {
		desired = 2
	}
	return desired
}

func (m tuiModel) makeModeList(mon monitor, modes []monitorMode) list.Model {
	curr := currentMonitorMode(mon)
	items := make([]list.Item, 0, modeShortcutCount+len(modes))
	items = append(items,
		modeListItem{shortcut: "Preferred", mode: resolveModeStrategy(mon, modeStrategyPreferred)},
		modeListItem{shortcut: "Highest resolution", mode: resolveModeStrategy(mon, modeStrategyHighres)},
		modeListItem{shortcut: "Highest refresh", mode: resolveModeStrategy(mon, modeStrategyHighrr)},
	)
	for _, mode := range modes {
		isCurr := mode.Width == curr.Width && mode.Height == curr.Height && mode.RefreshRate == curr.RefreshRate
		items = append(items, modeListItem{mode: mode, current: isCurr})
	}
	l := m.newStyledList(items)
	l.SetSize(m.width-4, m.modeListHeight(len(items)))
	return l
}

// hyprland transform values: 0-3 rotate counter-clockwise in 90° steps,
// 4-7 are the same rotations of a flipped (mirrored) image.
func (m tuiModel) makeTransformList(current int) list.Model {
	items := make([]list.Item, 0, len(transformLabels))
	for value, label := range transformLabels {
		items = append(items, settingListItem{label: label, value: value, current: value == current})
	}
	l := m.newStyledList(items)
	l.SetSize(m.width-4, m.clampListHeight(transformListHeight, 0))
	l.Select(current)
	return l
}

// directionListHeight is the fixed list height for the direction screen.
// The list has exactly 4 items; one extra row keeps the paginator hidden so
// all four directions are visible at once, while staying compact enough for
// the live preview to render below on the same screen.
const directionListHeight = 5

func (m tuiModel) makeDirectionList() list.Model {
	items := make([]list.Item, 0, 4)
	for _, dir := range []layoutDirection{leftToRight, rightToLeft, topToBottom, bottomToTop} {
		items = append(items, dirListItem{name: directionLabel(dir), dir: dir})
	}
	l := m.newStyledList(items)
	l.SetSize(m.width-4, m.clampListHeight(directionListHeight, 0))
	return l
}

func (m tuiModel) newStyledList(items []list.Item) list.Model {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetHeight(1)
	d.SetSpacing(0)
	d.Styles.SelectedTitle = styleSelected.Copy().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorAccent).
		PaddingLeft(1)
	d.Styles.NormalTitle = styleBase.Copy().PaddingLeft(2)
	d.Styles.DimmedTitle = styleDimmed.Copy().PaddingLeft(2)

	contentH := m.height - 2
	listH := contentH - 4
	if listH < 2 {
		listH = 5
	}

	l := list.New(items, d, m.width-4, listH)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.PaginationStyle = styleDimmed.Copy().PaddingLeft(2)
	l.KeyMap.Quit = key.NewBinding()
	l.KeyMap.ForceQuit = key.NewBinding()
	return l
}

// directionListIdx returns the list index for a given direction constant,
// matching the order in makeDirectionList.
func directionListIdx(d layoutDirection) int {
	switch d {
	case rightToLeft:
		return 1
	case topToBottom:
		return 2
	case bottomToTop:
		return 3
	}
	return 0 // leftToRight
}

// modeListIdx returns the list index of the given mode in a mode list built by
// makeModeList (offset past the strategy shortcuts), or 0 if not found.
func modeListIdx(selected monitorMode, modes []monitorMode) int {
	for i, m := range modes {
		if m.Width == selected.Width && m.Height == selected.Height && m.RefreshRate == selected.RefreshRate {
			return modeShortcutCount + i
		}
	}
	return 0
}
