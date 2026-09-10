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
//
// The hub is one flat list where every row is a single named setting, so
// `enter` always means the same thing: change this one. There are no per-row
// shortcuts to learn and the footer stays short.

type reviewRowKind int

const (
	reviewRowHeader reviewRowKind = iota // group label — not selectable
	reviewRowSpacer                      // blank line — not selectable
	reviewRowMode
	reviewRowRotation
	reviewRowVRR
	reviewRowDirection
	reviewRowOrder
	reviewRowSaveProfile
	reviewRowShowConfig
)

// selectable reports whether the cursor may land on a row.
func (k reviewRowKind) selectable() bool {
	return k != reviewRowHeader && k != reviewRowSpacer
}

type reviewRow struct {
	kind reviewRowKind
	// idx is the activeConfigs entry a monitor setting belongs to.
	idx int
	// label is the header text for a header row.
	label string
}

// reviewRows builds the hub's rows: each monitor's settings under its name,
// then the layout-wide settings, then the actions. Mirrored layouts stack every
// monitor on the same source, so they have no direction or order to set, and a
// single monitor has nothing to reorder.
func (m tuiModel) reviewRows() []reviewRow {
	rows := make([]reviewRow, 0, len(m.activeConfigs)*5+8)

	for i, cfg := range m.activeConfigs {
		if i > 0 {
			rows = append(rows, reviewRow{kind: reviewRowSpacer})
		}
		rows = append(rows,
			reviewRow{kind: reviewRowHeader, label: m.monitors[cfg.Index].Name},
			reviewRow{kind: reviewRowMode, idx: i},
			reviewRow{kind: reviewRowRotation, idx: i},
			reviewRow{kind: reviewRowVRR, idx: i},
		)
	}

	if !m.mirrored {
		rows = append(rows,
			reviewRow{kind: reviewRowSpacer},
			reviewRow{kind: reviewRowHeader, label: "Layout"},
			reviewRow{kind: reviewRowDirection},
		)
		if len(m.activeConfigs) > 1 {
			rows = append(rows, reviewRow{kind: reviewRowOrder})
		}
	}

	rows = append(rows,
		reviewRow{kind: reviewRowSpacer},
		reviewRow{kind: reviewRowHeader, label: "Actions"},
		reviewRow{kind: reviewRowSaveProfile},
		reviewRow{kind: reviewRowShowConfig},
	)
	return rows
}

func (m tuiModel) selectedReviewRow() (reviewRow, bool) {
	rows := m.reviewRows()
	if m.reviewCursor < 0 || m.reviewCursor >= len(rows) {
		return reviewRow{}, false
	}
	return rows[m.reviewCursor], true
}

// firstSelectableRow is where the cursor starts: the first monitor's mode.
func firstSelectableRow(rows []reviewRow) int {
	for i, row := range rows {
		if row.kind.selectable() {
			return i
		}
	}
	return 0
}

// moveReviewCursor steps the cursor by delta, skipping the group headers and
// blank lines so every keypress lands on something editable.
func (m tuiModel) moveReviewCursor(delta int) tuiModel {
	rows := m.reviewRows()
	for i := m.reviewCursor + delta; i >= 0 && i < len(rows); i += delta {
		if rows[i].kind.selectable() {
			m.reviewCursor = i
			break
		}
	}
	return m
}

func (m tuiModel) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.Up):
		return m.moveReviewCursor(-1), nil
	case key.Matches(km, tuiKeys.Down):
		return m.moveReviewCursor(1), nil
	case key.Matches(km, tuiKeys.Enter):
		return m.openReviewEditor()
	case key.Matches(km, tuiKeys.Apply):
		return m.startApply()
	}
	return m, nil
}

// openReviewEditor opens the editor for whichever single setting is highlighted.
func (m tuiModel) openReviewEditor() (tea.Model, tea.Cmd) {
	row, ok := m.selectedReviewRow()
	if !ok {
		return m, nil
	}
	switch row.kind {
	case reviewRowMode:
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

	case reviewRowRotation:
		m.editIdx = row.idx
		m.state = tuiTransformSelect
		m.list = m.makeTransformList(m.activeConfigs[row.idx].Transform)
		return m, nil

	case reviewRowVRR:
		m.editIdx = row.idx
		m.state = tuiVRRSelect
		m.list = m.makeVRRList(m.activeConfigs[row.idx].VRR)
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

	case reviewRowSaveProfile:
		return m.openProfileName()

	case reviewRowShowConfig:
		return m.openConfigView()
	}
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

// buildRules turns the current review state into monitor rules.
func (m tuiModel) buildRules() []layout.Rule {
	if m.mirrored {
		return layout.MirroredRules(m.monitors, m.activeConfigs, layout.MirrorSourceIndex(m.monitors, m.activeIndexes))
	}
	return layout.PositionedRules(m.monitors, m.activeConfigs, m.direction)
}

// buildConfigLines renders the rules in the syntax of the file being written.
func (m tuiModel) buildConfigLines() []string {
	return layout.Lines(m.buildRules(), m.target.Format)
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
		return "On"
	case 2:
		return "Fullscreen only"
	default:
		return "Off"
	}
}

// ── rendering ─────────────────────────────────────────────────────────────────

// reviewLabelWidth is the width of the setting-name column.
const reviewLabelWidth = 10

// reviewView renders the hub over a live preview of the arrangement. The row
// list scrolls when it is taller than the space left by the preview, so the
// preview never gets squeezed off the screen by a long monitor list.
func (m tuiModel) reviewView(h int) string {
	rows := m.reviewRows()
	first, last := m.reviewWindow(rows)

	var b strings.Builder
	b.WriteString("\n")
	if first > 0 {
		b.WriteString("  " + ui.Dimmed.Render("↑ more") + "\n")
	}
	for i := first; i < last; i++ {
		b.WriteString(m.reviewRowLine(rows[i], i == m.reviewCursor))
	}
	if last < len(rows) {
		b.WriteString("  " + ui.Dimmed.Render("↓ more") + "\n")
	}

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	return lipgloss.NewStyle().Height(h).Render(body)
}

// reviewWindow returns the half-open range of rows to draw, keeping the cursor
// visible and leaving room for the preview whenever the terminal allows it.
func (m tuiModel) reviewWindow(rows []reviewRow) (int, int) {
	// One line each for the header, footer and the leading blank.
	avail := m.height - 3
	if avail-len(rows) < previewReserve {
		// Not everything fits alongside the preview — scroll the list instead,
		// unless the terminal is so short that a preview is hopeless anyway.
		if h := avail - previewReserve; h >= 4 {
			avail = h
		}
	}
	if avail >= len(rows) {
		return 0, len(rows)
	}
	if avail < 1 {
		avail = 1
	}
	// Two rows are spent on the "more" markers once the list scrolls.
	avail -= 2
	if avail < 1 {
		avail = 1
	}

	first := m.reviewCursor - avail/2
	if first < 0 {
		first = 0
	}
	if first+avail > len(rows) {
		first = len(rows) - avail
	}
	return first, first + avail
}

func (m tuiModel) reviewRowLine(row reviewRow, selected bool) string {
	switch row.kind {
	case reviewRowSpacer:
		return "\n"
	case reviewRowHeader:
		return "  " + ui.Base.Render(row.label) + "\n"
	}

	label, value := m.reviewRowContent(row)

	marker := "    "
	labelStyle := ui.Muted
	if selected {
		marker = "  " + ui.Accent.Render("▸ ")
		labelStyle = ui.Selected
	}
	return marker + labelStyle.Render(fmt.Sprintf("%-*s", reviewLabelWidth, label)) + " " + value + "\n"
}

// reviewRowContent returns a row's name and its rendered value, highlighting
// the value when it differs from what the monitor is running now.
func (m tuiModel) reviewRowContent(row reviewRow) (string, string) {
	switch row.kind {
	case reviewRowMode:
		cfg := m.activeConfigs[row.idx]
		mon := m.monitors[cfg.Index]
		return "Mode", pendingValue(layout.FormatMode(cfg.Mode), 0, cfg.Mode != layout.CurrentMode(mon))
	case reviewRowRotation:
		cfg := m.activeConfigs[row.idx]
		mon := m.monitors[cfg.Index]
		return "Rotation", pendingValue(transformLabel(cfg.Transform), 0, cfg.Transform != mon.Transform)
	case reviewRowVRR:
		cfg := m.activeConfigs[row.idx]
		mon := m.monitors[cfg.Index]
		return "VRR", pendingValue(vrrLabel(cfg.VRR), 0, cfg.VRR != currentVRR(mon))
	case reviewRowDirection:
		return "Direction", ui.Dimmed.Render(directionLabel(m.direction))
	case reviewRowOrder:
		return "Order", ui.Dimmed.Render(m.orderSummary())
	case reviewRowSaveProfile:
		return "Save as profile", ""
	case reviewRowShowConfig:
		return "Show config", ""
	}
	return "", ""
}

// currentVRR maps a monitor's VRR flag onto the 0/1/2 config value.
func currentVRR(mon hypr.Monitor) int {
	if mon.VRR {
		return 1
	}
	return 0
}

// pendingValue renders a setting, padded to width and highlighted when it
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
