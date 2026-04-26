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

const (
	tuiDetecting tuiState = iota
	tuiLayoutSelect
	tuiModeSelect
	tuiDirectionSelect
	tuiOrderInput
	tuiPreview
	tuiSaveConfirm
	tuiProfileName
	tuiApplyConfirm
	tuiApplying
	tuiReloadConfirm
	tuiReloading
	tuiDone
	tuiErr
)

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
	mode    monitorMode
	current bool
}

func (i modeListItem) Title() string {
	s := formatMonitorMode(i.mode)
	if i.current {
		return s + "  " + styleDimmed.Render("current")
	}
	return s
}
func (i modeListItem) Description() string { return "" }
func (i modeListItem) FilterValue() string { return formatMonitorMode(i.mode) }

type dirListItem struct {
	name string
	dir  layoutDirection
}

func (i dirListItem) Title() string       { return i.name }
func (i dirListItem) Description() string { return "" }
func (i dirListItem) FilterValue() string { return i.name }

// ── key map ───────────────────────────────────────────────────────────────────

type tuiKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Yes   key.Binding
	No    key.Binding
	Back  key.Binding
	Quit  key.Binding
}

var tuiKeys = tuiKeyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Yes:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
	No:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "no")),
	Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// ── model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	state  tuiState
	width  int
	height int

	// pipeline data
	monitors       []monitor
	activeIndexes  []int
	selectedModes  map[int]monitorMode
	direction      layoutDirection
	activeConfigs  []activeMonitorConfig
	configLines    []string
	applyResult    applyResult
	currentModeIdx int

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
		state:         tuiDetecting,
		spinner:       s,
		textInput:     ti,
		selectedModes: make(map[int]monitorMode),
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
		m.state = tuiReloadConfirm
		return m, nil

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
		if msg.err != nil {
			m.inputErr = "Failed to save profile: " + msg.err.Error()
		} else {
			m.statusLines = append(m.statusLines,
				styleSuccess.Render("✓")+"  Profile saved.",
			)
		}
		m.state = tuiApplyConfirm
		return m, nil
	}

	switch m.state {
	case tuiLayoutSelect:
		return m.updateLayoutSelect(msg)
	case tuiModeSelect:
		return m.updateModeSelect(msg)
	case tuiDirectionSelect:
		return m.updateDirectionSelect(msg)
	case tuiOrderInput:
		return m.updateOrderInput(msg)
	case tuiPreview:
		return m.updatePreview(msg)
	case tuiSaveConfirm:
		return m.updateSaveConfirm(msg)
	case tuiProfileName:
		return m.updateProfileName(msg)
	case tuiApplyConfirm:
		return m.updateApplyConfirm(msg)
	case tuiReloadConfirm:
		return m.updateReloadConfirm(msg)
	case tuiDone, tuiErr:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	return m, nil
}

// ── state update helpers ──────────────────────────────────────────────────────

func (m tuiModel) updateLayoutSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(layoutListItem)
		if !ok {
			return m, nil
		}
		if item.opt.ID == 5 {
			return m, tea.Quit
		}
		indexes, err := activeIndexesForLayout(m.monitors, item.opt.ID)
		if err != nil {
			m.err = err
			m.state = tuiErr
			return m, nil
		}
		m.activeIndexes = indexes
		m.currentModeIdx = 0
		return m.advanceModeSelect()
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// advanceModeSelect skips monitors with no available modes (auto-selecting current)
// and either shows the next mode list or moves on to direction selection.
func (m tuiModel) advanceModeSelect() (tuiModel, tea.Cmd) {
	for m.currentModeIdx < len(m.activeIndexes) {
		idx := m.activeIndexes[m.currentModeIdx]
		mon := m.monitors[idx]
		modes := availableMonitorModes(mon)
		if len(modes) == 0 {
			m.selectedModes[idx] = currentMonitorMode(mon)
			m.currentModeIdx++
			continue
		}
		m.state = tuiModeSelect
		m.list = m.makeModeList(mon, modes)
		return m, nil
	}
	m.state = tuiDirectionSelect
	m.list = m.makeDirectionList()
	return m, nil
}

func (m tuiModel) updateModeSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(modeListItem)
		if !ok {
			return m, nil
		}
		m.selectedModes[m.activeIndexes[m.currentModeIdx]] = item.mode
		m.currentModeIdx++
		return m.advanceModeSelect()
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
		m.activeConfigs = buildActiveMonitorConfigs(m.monitors, m.activeIndexes, m.selectedModes)
		m.state = tuiOrderInput
		m.textInput.SetValue("")
		m.textInput.Placeholder = orderPlaceholder(m.activeConfigs)
		m.inputErr = ""
		m.textInput.Focus()
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) updateOrderInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		ordered, err := reorderActiveConfigs(m.textInput.Value(), m.activeConfigs)
		if err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.activeConfigs = ordered
		m.configLines = renderPositionedConfigLines(m.monitors, m.activeConfigs, m.direction)
		m.state = tuiPreview
		m.inputErr = ""
		m.viewport.SetContent(strings.Join(m.configLines, "\n"))
		return m, nil
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m tuiModel) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, tuiKeys.Yes):
			m.state = tuiSaveConfirm
			return m, nil
		case key.Matches(km, tuiKeys.No):
			m.state = tuiApplyConfirm
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m tuiModel) updateSaveConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.Yes):
		m.state = tuiProfileName
		m.textInput.SetValue("")
		m.textInput.Placeholder = "my-profile"
		m.inputErr = ""
		m.textInput.Focus()
		return m, textinput.Blink
	case key.Matches(km, tuiKeys.No):
		m.state = tuiApplyConfirm
		return m, nil
	}
	return m, nil
}

func (m tuiModel) updateProfileName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		name := strings.TrimSpace(m.textInput.Value())
		if err := validateProfileName(name); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		lines := m.configLines
		return m, func() tea.Msg {
			dir, err := defaultProfilesDir()
			if err != nil {
				return profileSavedMsg{err: err}
			}
			return profileSavedMsg{err: saveProfile(dir, name, lines)}
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m tuiModel) updateApplyConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.Yes):
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
	case key.Matches(km, tuiKeys.No):
		m.statusLines = append(m.statusLines, styleDimmed.Render("  No changes applied."))
		m.state = tuiDone
		return m, nil
	}
	return m, nil
}

func (m tuiModel) updateReloadConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.Yes):
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
	case key.Matches(km, tuiKeys.No):
		m.statusLines = append(m.statusLines,
			styleDimmed.Render("  Run ")+styleAccent.Render("hyprctl reload")+styleDimmed.Render(" manually to apply."),
		)
		m.state = tuiDone
		return m, nil
	}
	return m, nil
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if m.width == 0 {
		return ""
	}
	if m.width < 40 || m.height < 10 {
		return "  Terminal too small — resize to continue.\n"
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.bodyView(),
		m.footerView(),
	)
}

func (m tuiModel) headerView() string {
	appName := styleAccent.Copy().Bold(true).Render("hypr-layout")
	step := styleMuted.Render(m.stepLabel())
	inner := "  " + appName + "  " + styleDimmed.Render("·") + "  " + step
	pad := m.width - lipgloss.Width(inner)
	if pad > 0 {
		inner += strings.Repeat(" ", pad)
	}
	return styleBarBg.Copy().Width(m.width).Render(inner)
}

func (m tuiModel) footerView() string {
	inner := "  " + m.footerHints()
	pad := m.width - lipgloss.Width(inner)
	if pad > 0 {
		inner += strings.Repeat(" ", pad)
	}
	return styleBarBg.Copy().Width(m.width).Render(inner)
}

func (m tuiModel) stepLabel() string {
	switch m.state {
	case tuiDetecting:
		return "Detecting monitors"
	case tuiLayoutSelect:
		return "Select layout"
	case tuiModeSelect:
		if len(m.activeIndexes) > 0 && m.currentModeIdx < len(m.activeIndexes) {
			mon := m.monitors[m.activeIndexes[m.currentModeIdx]]
			return fmt.Sprintf("Mode for %s  (%d/%d)", mon.Name, m.currentModeIdx+1, len(m.activeIndexes))
		}
		return "Select mode"
	case tuiDirectionSelect:
		return "Layout direction"
	case tuiOrderInput:
		return "Monitor order"
	case tuiPreview:
		return "Preview"
	case tuiSaveConfirm:
		return "Save profile?"
	case tuiProfileName:
		return "Profile name"
	case tuiApplyConfirm:
		return "Apply?"
	case tuiApplying:
		return "Writing config…"
	case tuiReloadConfirm:
		return "Reload Hyprland?"
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
	case tuiModeSelect, tuiDirectionSelect:
		return dim("↑/↓") + " navigate" + sep + acc("enter") + " select" + sep + back + sep + acc("q") + " quit"
	case tuiOrderInput, tuiProfileName:
		return acc("enter") + " confirm" + sep + back + sep + acc("q") + " quit"
	case tuiPreview:
		return dim("↑/↓") + " scroll" + sep + acc("y") + " save profile" + sep + acc("n") + " skip" + sep + back + sep + acc("q") + " quit"
	case tuiSaveConfirm, tuiApplyConfirm:
		return acc("y") + " yes" + sep + acc("n") + " no" + sep + back + sep + acc("q") + " quit"
	case tuiReloadConfirm:
		return acc("y") + " yes" + sep + acc("n") + " no" + sep + acc("q") + " quit"
	case tuiDone, tuiErr:
		return dim("any key") + " exit"
	default:
		return ""
	}
}

func (m tuiModel) bodyView() string {
	contentH := m.height - 2 // one line each for header and footer
	switch m.state {
	case tuiDetecting:
		return m.spinnerView(contentH, "Detecting monitors…")
	case tuiLayoutSelect:
		return m.layoutSelectView(contentH)
	case tuiModeSelect:
		title := "Select mode"
		if len(m.activeIndexes) > 0 && m.currentModeIdx < len(m.activeIndexes) {
			mon := m.monitors[m.activeIndexes[m.currentModeIdx]]
			title = fmt.Sprintf("Select mode for %s", mon.Name)
			if len(m.activeIndexes) > 1 {
				title += fmt.Sprintf("  %s", styleDimmed.Render(fmt.Sprintf("(%d of %d)", m.currentModeIdx+1, len(m.activeIndexes))))
			}
		}
		return m.listView(contentH, title)
	case tuiDirectionSelect:
		return m.directionSelectView(contentH)
	case tuiOrderInput:
		return m.orderInputView(contentH)
	case tuiPreview:
		return m.previewView(contentH)
	case tuiSaveConfirm:
		return m.yesNoView(contentH, "Save this layout as a profile?")
	case tuiProfileName:
		return m.profileNameView(contentH)
	case tuiApplyConfirm:
		return m.yesNoView(contentH, "Apply this layout to ~/.config/hypr/monitors.conf?")
	case tuiApplying:
		return m.spinnerView(contentH, "Writing config…")
	case tuiReloadConfirm:
		return m.reloadConfirmView(contentH)
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

func (m tuiModel) directionSelectView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Select layout direction") + "\n\n")
	b.WriteString(m.list.View())

	// Live preview: recalculate and redraw for whichever item is highlighted.
	if item, ok := m.list.SelectedItem().(dirListItem); ok && len(m.activeConfigs) > 0 {
		previewW := m.width - 6
		if previewW < 24 {
			previewW = 24
		}
		preview := renderDirectionPreview(m.monitors, m.activeConfigs, item.dir, previewW)
		if preview != "" {
			b.WriteString("\n\n  " + styleTitle.Render("Preview") + "\n\n")
			b.WriteString(indentBlock(preview, 2))
			b.WriteByte('\n')
		}
	}

	return lipgloss.NewStyle().Height(h).Render(b.String())
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
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) listView(h int, title string) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render(title) + "\n\n")
	b.WriteString(m.list.View())
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) orderInputView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Set monitor order") + "\n\n")
	for i, cfg := range m.activeConfigs {
		mon := m.monitors[cfg.Index]
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			styleDimmed.Render(fmt.Sprintf("(%d)", i+1)),
			styleBase.Render(mon.Name),
			styleDimmed.Render(formatMonitorMode(cfg.Mode)),
		))
	}
	b.WriteString("\n  " + styleMuted.Render("Enter indices separated by spaces:") + "\n")
	b.WriteString("  " + m.textInput.View() + "\n")
	if m.inputErr != "" {
		b.WriteString("\n  " + styleErr.Render(m.inputErr) + "\n")
	}
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) previewView(h int) string {
	inner := m.viewport.View()
	boxWidth := m.width - 6
	if boxWidth < 10 {
		boxWidth = 10
	}
	box := stylePreviewBox.Copy().Width(boxWidth).Render(inner)

	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Config preview") + "\n\n")
	b.WriteString("  " + box + "\n")
	b.WriteString("\n  " + styleMuted.Render("Save as profile?  ") +
		styleAccent.Render("y") + styleDimmed.Render(" yes  ") +
		styleAccent.Render("n") + styleDimmed.Render(" skip") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) yesNoView(h int, question string) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, line := range m.statusLines {
		b.WriteString("  " + line + "\n")
	}
	if len(m.statusLines) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("  " + styleBase.Render(question) + "\n\n")
	b.WriteString("  " + stylePromptGlyph.Render("❯ ") +
		styleAccent.Render("y") + styleDimmed.Render("  /  ") +
		styleAccent.Render("n") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) reloadConfirmView(h int) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, line := range m.statusLines {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n  " + styleBase.Render("Reload Hyprland now?") + "\n\n")
	b.WriteString("  " + stylePromptGlyph.Render("❯ ") +
		styleAccent.Render("y") + styleDimmed.Render("  /  ") +
		styleAccent.Render("n") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
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
// States where a side-effect has already occurred (writing config, reloading)
// or where there is no previous step (detecting, done, error) do not allow it.
func backAllowed(s tuiState) bool {
	switch s {
	case tuiModeSelect, tuiDirectionSelect, tuiOrderInput,
		tuiPreview, tuiSaveConfirm, tuiProfileName, tuiApplyConfirm:
		return true
	}
	return false
}

func (m tuiModel) goBack() (tuiModel, tea.Cmd) {
	switch m.state {
	case tuiModeSelect:
		if m.currentModeIdx == 0 {
			m.state = tuiLayoutSelect
			m.list = m.makeLayoutList()
			return m, nil
		}
		return m.rewindModeSelect()

	case tuiDirectionSelect:
		return m.rewindToLastModeOrLayout()

	case tuiOrderInput:
		m.textInput.Blur()
		m.inputErr = ""
		m.state = tuiDirectionSelect
		m.list = m.makeDirectionList()
		m.list.Select(directionListIdx(m.direction))
		return m, nil

	case tuiPreview:
		m.state = tuiOrderInput
		m.textInput.SetValue("")
		m.textInput.Placeholder = orderPlaceholder(m.activeConfigs)
		m.inputErr = ""
		m.textInput.Focus()
		return m, textinput.Blink

	case tuiSaveConfirm:
		m.state = tuiPreview
		return m, nil

	case tuiProfileName:
		m.textInput.Blur()
		m.inputErr = ""
		m.state = tuiSaveConfirm
		return m, nil

	case tuiApplyConfirm:
		m.state = tuiPreview
		return m, nil
	}
	return m, nil
}

// rewindModeSelect moves back to the previous monitor that has selectable modes,
// skipping any monitors that were auto-selected (no available modes).
// The list cursor is restored to the previously chosen mode for that monitor.
func (m tuiModel) rewindModeSelect() (tuiModel, tea.Cmd) {
	for i := m.currentModeIdx - 1; i >= 0; i-- {
		idx := m.activeIndexes[i]
		mon := m.monitors[idx]
		if modes := availableMonitorModes(mon); len(modes) > 0 {
			m.currentModeIdx = i
			m.state = tuiModeSelect
			m.list = m.makeModeList(mon, modes)
			if prev, ok := m.selectedModes[idx]; ok {
				m.list.Select(modeListIdx(prev, modes))
			}
			return m, nil
		}
	}
	// No previous monitor has selectable modes — go all the way back to layout.
	m.state = tuiLayoutSelect
	m.list = m.makeLayoutList()
	return m, nil
}

// rewindToLastModeOrLayout moves back from direction select to the last
// monitor that had selectable modes, or to layout select if all were auto-selected.
// The list cursor is restored to the previously chosen mode for that monitor.
func (m tuiModel) rewindToLastModeOrLayout() (tuiModel, tea.Cmd) {
	for i := len(m.activeIndexes) - 1; i >= 0; i-- {
		idx := m.activeIndexes[i]
		mon := m.monitors[idx]
		if modes := availableMonitorModes(mon); len(modes) > 0 {
			m.currentModeIdx = i
			m.state = tuiModeSelect
			m.list = m.makeModeList(mon, modes)
			if prev, ok := m.selectedModes[idx]; ok {
				m.list.Select(modeListIdx(prev, modes))
			}
			return m, nil
		}
	}
	m.state = tuiLayoutSelect
	m.list = m.makeLayoutList()
	return m, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m tuiModel) resizeComponents() tuiModel {
	if m.width == 0 || m.height == 0 {
		return m
	}
	contentH := m.height - 2
	listH := contentH - 4
	if listH < 2 {
		listH = 2
	}
	// Only resize the list when it has been initialised; calling SetSize on a
	// zero-value list.Model panics because its internal paginator is nil.
	switch m.state {
	case tuiLayoutSelect, tuiModeSelect:
		m.list.SetSize(m.width-4, listH)
	case tuiDirectionSelect:
		// Direction list has exactly 4 fixed items. Keep it compact so the
		// live preview has room below it on the same screen.
		m.list.SetSize(m.width-4, directionListHeight)
	}

	vpW := m.width - 10
	vpH := contentH - 8
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
	return m.newStyledList(items)
}

func (m tuiModel) makeModeList(mon monitor, modes []monitorMode) list.Model {
	curr := currentMonitorMode(mon)
	items := make([]list.Item, 0, len(modes))
	for _, mode := range modes {
		isCurr := mode.Width == curr.Width && mode.Height == curr.Height && mode.RefreshRate == curr.RefreshRate
		items = append(items, modeListItem{mode: mode, current: isCurr})
	}
	return m.newStyledList(items)
}

// directionListHeight is the fixed list height for the direction screen.
// The list has exactly 4 items; keeping it compact lets the preview render
// on the same screen without being pushed off the bottom.
const directionListHeight = 4

func (m tuiModel) makeDirectionList() list.Model {
	items := []list.Item{
		dirListItem{name: "Left → right", dir: leftToRight},
		dirListItem{name: "Right → left", dir: rightToLeft},
		dirListItem{name: "Top → bottom", dir: topToBottom},
		dirListItem{name: "Bottom → top", dir: bottomToTop},
	}
	l := m.newStyledList(items)
	l.SetSize(m.width-4, directionListHeight)
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

// modeListIdx returns the list index of the given mode in a slice of modes,
// or 0 if not found.
func modeListIdx(selected monitorMode, modes []monitorMode) int {
	for i, m := range modes {
		if m.Width == selected.Width && m.Height == selected.Height && m.RefreshRate == selected.RefreshRate {
			return i
		}
	}
	return 0
}

func orderPlaceholder(configs []activeMonitorConfig) string {
	parts := make([]string, len(configs))
	for i := range configs {
		parts[i] = fmt.Sprintf("%d", i+1)
	}
	return strings.Join(parts, " ")
}
