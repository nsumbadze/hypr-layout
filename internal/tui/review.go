package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

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
		modes := layout.AvailableModes(mon)
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
		m.orderBackup = append([]layout.MonitorConfig(nil), m.activeConfigs...)
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
		return layout.MirroredLines(m.monitors, m.activeConfigs, layout.MirrorSourceIndex(m.monitors, m.activeIndexes))
	}
	return layout.PositionedLines(m.monitors, m.activeConfigs, m.direction)
}

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
		mon := m.monitors[cfg.Index]
		label = mon.Name
		value = pendingValue(layout.FormatMode(cfg.Mode), 16, cfg.Mode != layout.CurrentMode(mon)) +
			"  " + pendingValue(transformLabel(cfg.Transform), 14, cfg.Transform != mon.Transform) +
			"  " + pendingValue(vrrLabel(cfg.VRR), 0, cfg.VRR != currentVRR(mon))
	case reviewRowDirection:
		label = "Direction"
		value = ui.Dimmed.Render(directionLabel(m.direction))
	case reviewRowOrder:
		label = "Order"
		value = ui.Dimmed.Render(m.orderSummary())
	}

	marker := "  "
	labelStyle := ui.Base
	if selected {
		marker = ui.Accent.Render("▸ ")
		labelStyle = ui.Selected
	}
	return "  " + marker + labelStyle.Render(fmt.Sprintf("%-*s", nameW, label)) + "  " + value + "\n"
}

// currentVRR maps a monitor's VRR flag onto the 0/1/2 config value.
func currentVRR(mon hypr.Monitor) int {
	if mon.VRR {
		return 1
	}
	return 0
}

// pendingValue renders one setting, padded to width and highlighted when it
// differs from what the monitor is running now. The hub opens with everything
// at its current value, so a highlight is exactly the set of pending changes.
func pendingValue(text string, width int, changed bool) string {
	if n := width - lipgloss.Width(text); n > 0 {
		text += strings.Repeat(" ", n)
	}
	if changed {
		return ui.Accent.Render(text)
	}
	return ui.Dimmed.Render(text)
}

// orderSummary describes the current monitor order as a single line.
func (m tuiModel) orderSummary() string {
	names := make([]string, 0, len(m.activeConfigs))
	for _, cfg := range m.activeConfigs {
		names = append(names, m.monitors[cfg.Index].Name)
	}
	return strings.Join(names, " → ")
}

func directionLabel(d layout.Direction) string {
	switch d {
	case layout.RightToLeft:
		return "Right → left"
	case layout.TopToBottom:
		return "Top → bottom"
	case layout.BottomToTop:
		return "Bottom → top"
	}
	return "Left → right"
}
