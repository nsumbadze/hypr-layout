package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

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
	opts := layout.Options()
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

func (m tuiModel) makeModeList(mon hypr.Monitor, modes []layout.Mode) list.Model {
	curr := layout.CurrentMode(mon)
	items := make([]list.Item, 0, modeShortcutCount+len(modes))
	items = append(items,
		modeListItem{shortcut: "Preferred", mode: layout.ResolveStrategy(mon, layout.StrategyPreferred)},
		modeListItem{shortcut: "Highest resolution", mode: layout.ResolveStrategy(mon, layout.StrategyHighres)},
		modeListItem{shortcut: "Highest refresh", mode: layout.ResolveStrategy(mon, layout.StrategyHighrr)},
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
	for _, dir := range []layout.Direction{layout.LeftToRight, layout.RightToLeft, layout.TopToBottom, layout.BottomToTop} {
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
	d.Styles.SelectedTitle = ui.Selected.Copy().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ui.ColorAccent).
		PaddingLeft(1)
	d.Styles.NormalTitle = ui.Base.Copy().PaddingLeft(2)
	d.Styles.DimmedTitle = ui.Dimmed.Copy().PaddingLeft(2)

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
	l.Styles.PaginationStyle = ui.Dimmed.Copy().PaddingLeft(2)
	l.KeyMap.Quit = key.NewBinding()
	l.KeyMap.ForceQuit = key.NewBinding()
	return l
}

// directionListIdx returns the list index for a given direction constant,
// matching the order in makeDirectionList.
func directionListIdx(d layout.Direction) int {
	switch d {
	case layout.RightToLeft:
		return 1
	case layout.TopToBottom:
		return 2
	case layout.BottomToTop:
		return 3
	}
	return 0 // leftToRight
}

// modeListIdx returns the list index of the given mode in a mode list built by
// makeModeList (offset past the strategy shortcuts), or 0 if not found.
func modeListIdx(selected layout.Mode, modes []layout.Mode) int {
	for i, m := range modes {
		if m.Width == selected.Width && m.Height == selected.Height && m.RefreshRate == selected.RefreshRate {
			return modeShortcutCount + i
		}
	}
	return 0
}
