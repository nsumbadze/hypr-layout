package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func wizardTestModel(w, h int) tuiModel {
	m := newTUIModel()
	m.width, m.height = w, h
	m.monitors = []monitor{
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1, Focused: true,
			AvailableModes: []string{"2560x1440@165.00Hz", "2560x1440@144.00Hz", "1920x1080@60.00Hz"}},
		{Name: "HDMI-A-2", Width: 2560, Height: 1440, RefreshRate: 144, Scale: 1,
			AvailableModes: []string{"2560x1440@144.00Hz", "1920x1080@60.00Hz"}},
	}
	m.activeIndexes = []int{0, 1}
	m.activeConfigs = buildActiveMonitorConfigs(m.monitors, m.activeIndexes, nil)
	m = m.resizeComponents()
	return m
}

// wizardScreens returns the model rendered in each preview-capable state.
func wizardScreens(w, h int) map[string]tuiModel {
	screens := make(map[string]tuiModel)

	layout := wizardTestModel(w, h)
	layout.state = tuiLayoutSelect
	layout.list = layout.makeLayoutList()
	layout.list.Select(2) // Dual horizontal — available with the test monitors
	screens["layout"] = layout

	mode := wizardTestModel(w, h)
	mode.state = tuiModeSelect
	mode.currentModeIdx = 0
	mode.list = mode.makeModeList(mode.monitors[0], availableMonitorModes(mode.monitors[0]))
	screens["mode"] = mode

	transform := wizardTestModel(w, h)
	transform.state = tuiTransformSelect
	transform.list = transform.makeTransformList(0)
	screens["transform"] = transform

	vrr := wizardTestModel(w, h)
	vrr.state = tuiVRRSelect
	vrr.list = vrr.makeVRRList(0)
	screens["vrr"] = vrr

	direction, _ := wizardTestModel(w, h).finishSettings()
	screens["direction"] = direction

	order := wizardTestModel(w, h)
	order.state = tuiOrderInput
	order.direction = leftToRight
	screens["order"] = order

	preview := wizardTestModel(w, h)
	preview.state = tuiPreview
	preview.direction = leftToRight
	preview.configLines = renderPositionedConfigLines(preview.monitors, preview.activeConfigs, leftToRight)
	preview = preview.resizeComponents()
	preview.viewport.SetContent(strings.Join(preview.configLines, "\n"))
	screens["config-preview"] = preview

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
	m, _ := wizardTestModel(80, 24).finishSettings()
	view := m.View()

	for _, want := range []string{"Left → right", "Right → left", "Top → bottom", "Bottom → top", "Preview", "DP-1", "HDMI-A-2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("direction view missing %q:\n%s", want, view)
		}
	}
}

func TestDirectionScreenFitsAllDirections(t *testing.T) {
	m, _ := wizardTestModel(80, 24).finishSettings()

	var model tea.Model = m
	for i := 0; i < 4; i++ {
		view := model.(tuiModel).View()
		if got := strings.Count(view, "\n") + 1; got > 24 {
			t.Fatalf("direction %d: view is %d lines, exceeds terminal height", i, got)
		}
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
}

func TestMirrorLayoutShowsMirrorPreview(t *testing.T) {
	m := wizardTestModel(100, 40)
	m.mirrored = true
	m.state = tuiVRRSelect
	m.list = m.makeVRRList(0)

	view := m.View()
	if !strings.Contains(view, "mirrored by HDMI-A-2") {
		t.Fatalf("expected mirror caption in view:\n%s", view)
	}
}

func TestTransformHoverSwapsPreviewProportions(t *testing.T) {
	m := wizardTestModel(100, 40)
	m.state = tuiTransformSelect
	m.list = m.makeTransformList(0)

	normal := m.View()
	m.list.Select(1) // 90°
	rotated := m.View()

	if normal == rotated {
		t.Fatal("expected preview to change when hovering a 90° transform")
	}
}
