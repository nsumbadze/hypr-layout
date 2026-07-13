package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func directionTestModel(w, h int) tuiModel {
	m := newTUIModel()
	m.width, m.height = w, h
	m.monitors = []monitor{
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
		{Name: "HDMI-A-2", Width: 2560, Height: 1440, RefreshRate: 144, Scale: 1},
	}
	m.activeIndexes = []int{0, 1}
	m.activeConfigs = buildActiveMonitorConfigs(m.monitors, m.activeIndexes, nil)
	m = m.resizeComponents()
	return m
}

// The direction screen must always fit the terminal: an oversized body gets
// clipped by the renderer, which hides the live preview and the footer.
func TestDirectionScreenFitsTerminal(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 40}, {60, 15}} {
		m, _ := directionTestModel(size.w, size.h).finishSettings()

		// Check every direction, including the tall vertical previews.
		var model tea.Model = m
		for i := 0; i < 4; i++ {
			view := model.(tuiModel).View()
			if got := strings.Count(view, "\n") + 1; got > size.h {
				t.Fatalf("%dx%d direction %d: view is %d lines, exceeds terminal height", size.w, size.h, i, got)
			}
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
	}
}

func TestDirectionScreenShowsPreviewAndAllDirections(t *testing.T) {
	m, _ := directionTestModel(80, 24).finishSettings()
	view := m.View()

	for _, want := range []string{"Left → right", "Right → left", "Top → bottom", "Bottom → top", "Preview", "DP-1", "HDMI-A-2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("direction view missing %q:\n%s", want, view)
		}
	}
}

func TestDirectionScreenSkipsPreviewWhenTooSmall(t *testing.T) {
	m, _ := directionTestModel(60, 12).finishSettings()
	view := m.View()

	if strings.Contains(view, "╭") {
		t.Fatalf("expected preview to be skipped at 12 lines:\n%s", view)
	}

	if got := strings.Count(view, "\n") + 1; got > 12 {
		t.Fatalf("view is %d lines, exceeds terminal height", got)
	}
}
