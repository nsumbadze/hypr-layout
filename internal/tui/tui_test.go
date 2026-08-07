package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/ui"
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
	m.activeConfigs = layout.BuildConfigs(m.monitors, m.activeIndexes,
		defaultModeSelections(m.monitors, m.activeIndexes))
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

// Picking a layout must land on the hub with every setting defaulted, so the
// layout is applyable without visiting a single editor.
func TestLayoutSelectionLandsOnReviewWithDefaults(t *testing.T) {
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
	// Highest refresh rate is the default, matching `quick`'s "best" mode.
	if rr := got.activeConfigs[0].Mode.RefreshRate; rr != 165 {
		t.Fatalf("expected DP-1 to default to 165Hz, got %v", rr)
	}
	if got.direction != layout.LeftToRight {
		t.Fatalf("expected default direction left→right, got %q", got.direction)
	}
}

func TestReviewRowsCoverMonitorsAndLayout(t *testing.T) {
	rows := reviewTestModel(100, 40).reviewRows()
	if len(rows) != 4 {
		t.Fatalf("expected 2 monitor rows + direction + order, got %d", len(rows))
	}
	if rows[2].kind != reviewRowDirection || rows[3].kind != reviewRowOrder {
		t.Fatalf("expected direction and order rows last, got %+v", rows)
	}
}

// Mirrored layouts have no direction or order, so those rows must not appear.
func TestReviewRowsOmitLayoutRowsWhenMirrored(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.mirrored = true

	for _, row := range m.reviewRows() {
		if row.kind != reviewRowMonitor {
			t.Fatalf("mirrored layout should only have monitor rows, got kind %v", row.kind)
		}
	}
}

func TestReviewShowsEveryMonitorWithItsSettings(t *testing.T) {
	view := reviewTestModel(100, 40).View()
	for _, want := range []string{"DP-1", "HDMI-A-2", "2560x1440@165", "Normal", "vrr off", "Direction", "Order"} {
		if !strings.Contains(view, want) {
			t.Fatalf("review view missing %q:\n%s", want, view)
		}
	}
}

// VRR cycles in place on the hub rather than costing a screen of its own.
func TestReviewCyclesVRRInPlace(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.reviewCursor = 1

	for _, want := range []int{1, 2, 0} {
		next, _ := m.updateReview(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		m = next.(tuiModel)
		if got := m.activeConfigs[1].VRR; got != want {
			t.Fatalf("expected VRR %d after cycling, got %d", want, got)
		}
		if m.state != tuiReview {
			t.Fatalf("cycling VRR should stay on the hub, got state %v", m.state)
		}
	}
}

// The rotation and VRR shortcuts only act on monitor rows.
func TestReviewShortcutsIgnoreLayoutRows(t *testing.T) {
	m := reviewTestModel(100, 40)
	m.reviewCursor = 2 // direction row

	next, _ := m.updateReview(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if got := next.(tuiModel).state; got != tuiReview {
		t.Fatalf("rotate on a layout row should stay on the hub, got state %v", got)
	}
}

func TestReviewOpensEditorForHighlightedRow(t *testing.T) {
	tests := []struct {
		name   string
		cursor int
		key    tea.KeyMsg
		want   tuiState
	}{
		{"mode", 0, tea.KeyMsg{Type: tea.KeyEnter}, tuiModeSelect},
		{"rotation", 0, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}, tuiTransformSelect},
		{"direction", 2, tea.KeyMsg{Type: tea.KeyEnter}, tuiDirectionSelect},
		{"order", 3, tea.KeyMsg{Type: tea.KeyEnter}, tuiOrderEdit},
		{"config", 0, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}, tuiConfigView},
		{"profile", 0, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, tuiProfileName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := reviewTestModel(100, 40)
			m.reviewCursor = tc.cursor

			next, _ := m.updateReview(tc.key)
			if got := next.(tuiModel).state; got != tc.want {
				t.Fatalf("expected state %v, got %v", tc.want, got)
			}
		})
	}
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
	m.reviewCursor = 3 // order row

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

// "w" writes without reloading; "a" reloads afterwards. Neither asks first.
func TestApplyAndWriteSetReloadIntent(t *testing.T) {
	for _, tc := range []struct {
		key        rune
		wantReload bool
	}{{'a', true}, {'w', false}} {
		m := reviewTestModel(100, 40)
		next, cmd := m.updateReview(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
		got := next.(tuiModel)

		if got.state != tuiApplying {
			t.Fatalf("%q: expected applying state, got %v", tc.key, got.state)
		}
		if got.reloadAfterWrite != tc.wantReload {
			t.Fatalf("%q: expected reloadAfterWrite=%v", tc.key, tc.wantReload)
		}
		if cmd == nil {
			t.Fatalf("%q: expected a write command", tc.key)
		}
		if len(got.configLines) == 0 {
			t.Fatalf("%q: expected config lines to be rendered", tc.key)
		}
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

// The hub carries more shortcuts than a narrow terminal fits, so the optional
// ones must drop off rather than be truncated mid-word.
func TestReviewHintsFitTerminalWidth(t *testing.T) {
	for _, w := range []int{50, 60, 84, 120} {
		m := reviewTestModel(w, 26)
		hints := m.reviewHints(ui.Dimmed.Render("  ·  "))

		if got := lipgloss.Width(hints) + 2; got > w {
			t.Fatalf("at width %d: hints occupy %d columns: %q", w, got, hints)
		}
		for _, essential := range []string{"edit", "apply", "quit"} {
			if !strings.Contains(hints, essential) {
				t.Fatalf("at width %d: dropped essential hint %q: %q", w, essential, hints)
			}
		}
	}
}
