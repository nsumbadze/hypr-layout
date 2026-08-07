package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
)

func wizardTestModel(w, h int) tuiModel {
	m := newTUIModel()
	m.width, m.height = w, h
	m.monitors = []hypr.Monitor{
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1, Focused: true,
			AvailableModes: []string{"2560x1440@165.00Hz", "2560x1440@144.00Hz", "1920x1080@60.00Hz"}},
		{Name: "HDMI-A-2", Width: 2560, Height: 1440, RefreshRate: 144, Scale: 1,
			AvailableModes: []string{"2560x1440@144.00Hz", "1920x1080@60.00Hz"}},
	}
	m.activeIndexes = []int{0, 1}
	m.direction = layout.LeftToRight
	m.activeConfigs = layout.BuildConfigs(m.monitors, m.activeIndexes, nil)
	m = m.resizeComponents()
	return m
}

// reviewTestModel returns the model sitting on the review hub, which is where
// the wizard lands after a layout is picked.
func reviewTestModel(w, h int) tuiModel {
	m := wizardTestModel(w, h)
	m.state = tuiReview
	return m
}

// wizardScreens returns the model rendered in each preview-capable state.
func wizardScreens(w, h int) map[string]tuiModel {
	screens := make(map[string]tuiModel)

	layoutPick := wizardTestModel(w, h)
	layoutPick.state = tuiLayoutSelect
	layoutPick.list = layoutPick.makeLayoutList()
	layoutPick.list.Select(2) // Dual horizontal — available with the test monitors
	screens["layout"] = layoutPick

	screens["review"] = reviewTestModel(w, h)

	mode := wizardTestModel(w, h)
	mode.state = tuiModeSelect
	mode.editIdx = 0
	mode.list = mode.makeModeList(mode.monitors[0], layout.AvailableModes(mode.monitors[0]))
	screens["mode"] = mode

	transform := wizardTestModel(w, h)
	transform.state = tuiTransformSelect
	transform.list = transform.makeTransformList(0)
	screens["transform"] = transform

	vrr := wizardTestModel(w, h)
	vrr.state = tuiVRRSelect
	vrr.list = vrr.makeVRRList(0)
	screens["vrr"] = vrr

	direction := wizardTestModel(w, h)
	direction.state = tuiDirectionSelect
	direction.list = direction.makeDirectionList()
	screens["direction"] = direction

	order := wizardTestModel(w, h)
	order.state = tuiOrderEdit
	screens["order"] = order

	config := wizardTestModel(w, h)
	config.state = tuiConfigView
	config.configLines = layout.PositionedLines(config.monitors, config.activeConfigs, layout.LeftToRight)
	config = config.resizeComponents()
	config.viewport.SetContent(strings.Join(config.configLines, "\n"))
	screens["config-view"] = config

	return screens
}

// Every wizard screen must fit the terminal: an oversized body gets clipped
// by the renderer, which hides content and the footer.
func TestWizardScreensFitTerminal(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 40}, {60, 15}, {60, 12}} {
		for name, m := range wizardScreens(size.w, size.h) {
			view := m.View()
			if got := strings.Count(view, "\n") + 1; got > size.h {
				t.Fatalf("%s at %dx%d: view is %d lines, exceeds terminal height", name, size.w, size.h, got)
			}
		}
	}
}

// Every preview-capable screen should show the live preview when the
// terminal has room for it.
func TestWizardScreensShowPreview(t *testing.T) {
	for name, m := range wizardScreens(100, 40) {
		view := m.View()
		if !strings.Contains(view, "Preview") || !strings.Contains(view, "╭") {
			t.Fatalf("%s at 100x40: no preview rendered:\n%s", name, view)
		}
	}
}

func TestDirectionScreenShowsPreviewAndAllDirections(t *testing.T) {
	m := wizardScreens(80, 24)["direction"]
	view := m.View()

	for _, want := range []string{"Left → right", "Right → left", "Top → bottom", "Bottom → top", "Preview", "DP-1", "HDMI-A-2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("direction view missing %q:\n%s", want, view)
		}
	}
}

func TestDirectionScreenFitsAllDirections(t *testing.T) {
	var model tea.Model = wizardScreens(80, 24)["direction"]
	for i := 0; i < 4; i++ {
		view := model.(tuiModel).View()
		if got := strings.Count(view, "\n") + 1; got > 24 {
			t.Fatalf("direction %d: view is %d lines, exceeds terminal height", i, got)
		}
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
}

func TestMirrorLayoutShowsMirrorPreview(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.mirrored = true

	view := m.View()
	if !strings.Contains(view, "mirrored by HDMI-A-2") {
		t.Fatalf("expected mirror caption in view:\n%s", view)
	}
}

func TestTransformHoverSwapsPreviewProportions(t *testing.T) {
	m := wizardScreens(100, 40)["transform"]

	normal := m.View()
	m.list.Select(1) // 90°
	rotated := m.View()

	if normal == rotated {
		t.Fatal("expected preview to change when hovering a 90° transform")
	}
}

// ── review hub ────────────────────────────────────────────────────────────────

// Picking a layout must land on the hub showing what the monitors are running
// now — the wizard proposes nothing until the user changes something.
func TestLayoutSelectionLandsOnReviewShowingCurrentSettings(t *testing.T) {
	m := wizardTestModel(100, 40)
	m.state = tuiLayoutSelect
	m.list = m.makeLayoutList()
	m.list.Select(2) // Dual horizontal

	next, _ := m.updateLayoutSelect(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(tuiModel)

	if got.state != tuiReview {
		t.Fatalf("expected review state after layout selection, got %v", got.state)
	}
	if len(got.activeConfigs) != 2 {
		t.Fatalf("expected 2 active configs, got %d", len(got.activeConfigs))
	}
	for _, cfg := range got.activeConfigs {
		mon := got.monitors[cfg.Index]
		if cfg.Mode != layout.CurrentMode(mon) {
			t.Fatalf("%s: expected the current mode %v, got %v",
				mon.Name, layout.CurrentMode(mon), cfg.Mode)
		}
		if cfg.Transform != mon.Transform {
			t.Fatalf("%s: expected the current rotation %d, got %d", mon.Name, mon.Transform, cfg.Transform)
		}
		if cfg.VRR != currentVRR(mon) {
			t.Fatalf("%s: expected the current VRR %d, got %d", mon.Name, currentVRR(mon), cfg.VRR)
		}
	}
	if got.direction != layout.LeftToRight {
		t.Fatalf("expected default direction left→right, got %q", got.direction)
	}
}

// Applying straight off the hub must reproduce the monitors' current modes, so
// an accidental apply is a no-op rather than a surprise mode change.
func TestApplyingWithoutEditingKeepsCurrentModes(t *testing.T) {
	m := reviewTestModel(100, 40)

	lines := strings.Join(m.buildConfigLines(), "\n")
	for _, mon := range m.monitors {
		want := mon.Name + ", " + layout.FormatMode(layout.CurrentMode(mon))
		if !strings.Contains(lines, want) {
			t.Fatalf("expected %q in the generated config:\n%s", want, lines)
		}
	}
}

// Changed and unchanged values must occupy the same width, so highlighting a
// pending change never shifts the columns. (Colour itself is stripped in tests,
// which have no terminal profile, so only the layout is asserted here.)
func TestPendingValuePadsToAStableWidth(t *testing.T) {
	for _, changed := range []bool{false, true} {
		if got := lipgloss.Width(pendingValue("2560x1440@165", 16, changed)); got != 16 {
			t.Fatalf("changed=%v: expected width 16, got %d", changed, got)
		}
	}
	if got := lipgloss.Width(pendingValue("a-very-long-value-past-the-column", 4, false)); got != 33 {
		t.Fatalf("expected an over-long value to keep its own width, got %d", got)
	}
}

func TestReviewRowsAreOneSettingEach(t *testing.T) {
	rows := reviewTestModel(100, 40).reviewRows()

	var got []reviewRowKind
	for _, row := range rows {
		if row.kind.selectable() {
			got = append(got, row.kind)
		}
	}
	want := []reviewRowKind{
		reviewRowMode, reviewRowRotation, reviewRowVRR,
		reviewRowMode, reviewRowRotation, reviewRowVRR,
		reviewRowDirection, reviewRowOrder,
		reviewRowSaveProfile, reviewRowShowConfig,
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d selectable rows, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d: expected kind %v, got %v", i, want[i], got[i])
		}
	}
}

// Every monitor's settings must sit under a header naming that monitor.
func TestReviewGroupsSettingsUnderMonitorHeaders(t *testing.T) {
	rows := reviewTestModel(100, 40).reviewRows()

	var headers []string
	for _, row := range rows {
		if row.kind == reviewRowHeader {
			headers = append(headers, row.label)
		}
	}
	want := []string{"DP-1", "HDMI-A-2", "Layout", "Actions"}
	if len(headers) != len(want) {
		t.Fatalf("expected headers %v, got %v", want, headers)
	}
	for i := range want {
		if headers[i] != want[i] {
			t.Fatalf("expected header %q, got %q", want[i], headers[i])
		}
	}
}

// Mirrored layouts have no direction or order, so that group must not appear.
func TestReviewRowsOmitLayoutGroupWhenMirrored(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.mirrored = true

	for _, row := range m.reviewRows() {
		if row.kind == reviewRowDirection || row.kind == reviewRowOrder {
			t.Fatalf("mirrored layout should have no layout rows, got kind %v", row.kind)
		}
		if row.kind == reviewRowHeader && row.label == "Layout" {
			t.Fatal("mirrored layout should have no Layout header")
		}
	}
}

// The cursor must never come to rest on a header or a blank line.
func TestReviewCursorSkipsUnselectableRows(t *testing.T) {
	m := reviewTestModel(100, 40)
	rows := m.reviewRows()
	m.reviewCursor = firstSelectableRow(rows)

	for step := 0; step < len(rows); step++ {
		next, _ := m.updateReview(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(tuiModel)
		if !rows[m.reviewCursor].kind.selectable() {
			t.Fatalf("cursor landed on unselectable row %d (kind %v)", m.reviewCursor, rows[m.reviewCursor].kind)
		}
	}
	for step := 0; step < len(rows); step++ {
		next, _ := m.updateReview(tea.KeyMsg{Type: tea.KeyUp})
		m = next.(tuiModel)
		if !rows[m.reviewCursor].kind.selectable() {
			t.Fatalf("cursor landed on unselectable row %d (kind %v)", m.reviewCursor, rows[m.reviewCursor].kind)
		}
	}
}

func TestReviewLabelsEverySetting(t *testing.T) {
	view := reviewTestModel(100, 40).View()
	for _, want := range []string{
		"DP-1", "HDMI-A-2", "Mode", "Rotation", "VRR",
		"Layout", "Direction", "Order", "Actions", "Save as profile", "Show config",
		"2560x1440@165", "Normal", "Off",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("review view missing %q:\n%s", want, view)
		}
	}
}

func TestEnterOpensTheEditorForEachRow(t *testing.T) {
	tests := []struct {
		name string
		kind reviewRowKind
		want tuiState
	}{
		{"mode", reviewRowMode, tuiModeSelect},
		{"rotation", reviewRowRotation, tuiTransformSelect},
		{"vrr", reviewRowVRR, tuiVRRSelect},
		{"direction", reviewRowDirection, tuiDirectionSelect},
		{"order", reviewRowOrder, tuiOrderEdit},
		{"save profile", reviewRowSaveProfile, tuiProfileName},
		{"show config", reviewRowShowConfig, tuiConfigView},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := reviewTestModel(100, 40)
			m.reviewCursor = rowIndexOfKind(t, m, tc.kind)

			next, _ := m.updateReview(tea.KeyMsg{Type: tea.KeyEnter})
			if got := next.(tuiModel).state; got != tc.want {
				t.Fatalf("expected state %v, got %v", tc.want, got)
			}
		})
	}
}

func rowIndexOfKind(t *testing.T, m tuiModel, kind reviewRowKind) int {
	t.Helper()
	for i, row := range m.reviewRows() {
		if row.kind == kind {
			return i
		}
	}
	t.Fatalf("no row of kind %v", kind)
	return 0
}

// Every editor returns to the hub rather than to another step.
func TestEditorsReturnToReview(t *testing.T) {
	tests := []struct {
		name  string
		setup func(tuiModel) tuiModel
		apply func(tuiModel) (tea.Model, tea.Cmd)
	}{
		{
			name: "mode",
			setup: func(m tuiModel) tuiModel {
				m.state = tuiModeSelect
				m.list = m.makeModeList(m.monitors[0], layout.AvailableModes(m.monitors[0]))
				return m
			},
			apply: func(m tuiModel) (tea.Model, tea.Cmd) {
				return m.updateModeSelect(tea.KeyMsg{Type: tea.KeyEnter})
			},
		},
		{
			name: "rotation",
			setup: func(m tuiModel) tuiModel {
				m.state = tuiTransformSelect
				m.list = m.makeTransformList(0)
				return m
			},
			apply: func(m tuiModel) (tea.Model, tea.Cmd) {
				return m.updateTransformSelect(tea.KeyMsg{Type: tea.KeyEnter})
			},
		},
		{
			name: "direction",
			setup: func(m tuiModel) tuiModel {
				m.state = tuiDirectionSelect
				m.list = m.makeDirectionList()
				return m
			},
			apply: func(m tuiModel) (tea.Model, tea.Cmd) {
				return m.updateDirectionSelect(tea.KeyMsg{Type: tea.KeyEnter})
			},
		},
		{
			name:  "order",
			setup: func(m tuiModel) tuiModel { m.state = tuiOrderEdit; return m },
			apply: func(m tuiModel) (tea.Model, tea.Cmd) {
				return m.updateOrderEdit(tea.KeyMsg{Type: tea.KeyEnter})
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next, _ := tc.apply(tc.setup(reviewTestModel(100, 40)))
			if got := next.(tuiModel).state; got != tuiReview {
				t.Fatalf("expected return to review, got state %v", got)
			}
		})
	}
}

// ── reordering ────────────────────────────────────────────────────────────────

func TestOrderEditMovesMonitorWithShiftKeys(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.state = tuiOrderEdit
	m.editIdx = 1

	next, _ := m.updateOrderEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = next.(tuiModel)

	if m.activeConfigs[0].Index != 1 || m.activeConfigs[1].Index != 0 {
		t.Fatalf("expected monitors swapped, got %d then %d",
			m.activeConfigs[0].Index, m.activeConfigs[1].Index)
	}
	// The cursor follows the monitor it is moving.
	if m.editIdx != 0 {
		t.Fatalf("expected cursor to follow the moved monitor to 0, got %d", m.editIdx)
	}
}

func TestOrderEditClampsAtEnds(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.state = tuiOrderEdit
	m.editIdx = 0

	next, _ := m.updateOrderEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = next.(tuiModel)

	if m.activeConfigs[0].Index != 0 || m.editIdx != 0 {
		t.Fatal("moving the first monitor up should be a no-op")
	}
}

// Escaping out of a reorder must restore the order it started with.
func TestOrderEditCancelRestoresOrder(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.reviewCursor = rowIndexOfKind(t, m, reviewRowOrder)

	opened, _ := m.updateReview(tea.KeyMsg{Type: tea.KeyEnter})
	moved, _ := opened.(tuiModel).updateOrderEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	if got := moved.(tuiModel).activeConfigs[0].Index; got != 1 {
		t.Fatalf("expected monitors swapped before cancelling, got first index %d", got)
	}

	cancelled, _ := moved.(tuiModel).goBack()
	if cancelled.state != tuiReview {
		t.Fatalf("expected return to review after cancelling, got %v", cancelled.state)
	}
	if cancelled.activeConfigs[0].Index != 0 || cancelled.activeConfigs[1].Index != 1 {
		t.Fatalf("expected original order restored, got %d then %d",
			cancelled.activeConfigs[0].Index, cancelled.activeConfigs[1].Index)
	}
}

// ── navigation ────────────────────────────────────────────────────────────────

func TestBackFromReviewReturnsToLayoutSelect(t *testing.T) {
	back, _ := reviewTestModel(100, 40).goBack()
	if back.state != tuiLayoutSelect {
		t.Fatalf("expected layout select, got %v", back.state)
	}
}

func TestBackAllowedOnlyWhereThereIsSomethingToGoBackTo(t *testing.T) {
	allowed := []tuiState{tuiReview, tuiModeSelect, tuiTransformSelect,
		tuiDirectionSelect, tuiOrderEdit, tuiConfigView, tuiProfileName}
	for _, s := range allowed {
		if !backAllowed(s) {
			t.Fatalf("expected back to be allowed in state %v", s)
		}
	}
	for _, s := range []tuiState{tuiDetecting, tuiLayoutSelect, tuiApplying, tuiReloading, tuiDone, tuiErr} {
		if backAllowed(s) {
			t.Fatalf("expected back to be blocked in state %v", s)
		}
	}
}

// ── apply ─────────────────────────────────────────────────────────────────────

// "a" writes and reloads in one step, with no confirmation in between.
func TestApplyWritesAndReloads(t *testing.T) {
	m := reviewTestModel(100, 40)

	next, cmd := m.updateReview(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	got := next.(tuiModel)

	if got.state != tuiApplying {
		t.Fatalf("expected applying state, got %v", got.state)
	}
	if cmd == nil {
		t.Fatal("expected a write command")
	}
	if len(got.configLines) == 0 {
		t.Fatal("expected config lines to be rendered")
	}
}

func TestBuildConfigLinesFollowsLayoutKind(t *testing.T) {
	m := reviewTestModel(100, 40)
	positioned := strings.Join(m.buildConfigLines(), "\n")
	if !strings.Contains(positioned, "2560x0") {
		t.Fatalf("expected the second monitor offset to the right:\n%s", positioned)
	}

	m.mirrored = true
	if mirrored := strings.Join(m.buildConfigLines(), "\n"); !strings.Contains(mirrored, "mirror") {
		t.Fatalf("expected mirror keyword for a mirrored layout:\n%s", mirrored)
	}
}

// The hub footer must fit even a narrow terminal without being truncated.
func TestReviewFooterFitsNarrowTerminals(t *testing.T) {
	for _, w := range []int{60, 84, 120} {
		m := reviewTestModel(w, 26)
		if got := lipgloss.Width(m.footerHints()) + 2; got > w {
			t.Fatalf("at width %d: footer occupies %d columns", w, got)
		}
	}
}

// A long monitor list must scroll rather than push the preview off screen.
func TestReviewScrollsInsteadOfHidingPreview(t *testing.T) {
	m := reviewTestModel(100, 24)
	rows := m.reviewRows()

	first, last := m.reviewWindow(rows)
	if last-first >= len(rows) {
		t.Skip("everything fits at this size; nothing to scroll")
	}
	if !strings.Contains(m.View(), "Preview") {
		t.Fatalf("expected the preview to survive scrolling:\n%s", m.View())
	}
}

// Scrolling must always keep the cursor within the drawn window.
func TestReviewWindowKeepsCursorVisible(t *testing.T) {
	m := reviewTestModel(100, 20)
	rows := m.reviewRows()

	for i, row := range rows {
		if !row.kind.selectable() {
			continue
		}
		m.reviewCursor = i
		first, last := m.reviewWindow(rows)
		if i < first || i >= last {
			t.Fatalf("cursor %d outside drawn window [%d,%d)", i, first, last)
		}
	}
}
