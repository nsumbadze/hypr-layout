package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestFormatMonitorMode(t *testing.T) {
	mode := monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}

	got := formatMonitorMode(mode)
	want := "2560x1440@165"

	if got != want {
		t.Fatalf("unexpected formatted mode: got %q want %q", got, want)
	}
}

func TestParseMonitorModeSelectionDefault(t *testing.T) {
	current := monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}
	modes := []monitorMode{
		{Width: 2560, Height: 1440, RefreshRate: 60},
		{Width: 2560, Height: 1440, RefreshRate: 165},
	}

	got, err := parseMonitorModeSelection("\n", modes, current)
	if err != nil {
		t.Fatalf("parseMonitorModeSelection returned error: %v", err)
	}

	if got != current {
		t.Fatalf("unexpected default mode: got %#v want %#v", got, current)
	}
}

func TestPromptMonitorModesUsesDefaultOnEnter(t *testing.T) {
	monitors := []monitor{
		{
			Name:           "DP-1",
			Width:          2560,
			Height:         1440,
			RefreshRate:    165,
			AvailableModes: []string{"2560x1440@60.00Hz", "2560x1440@165.00Hz"},
		},
	}

	var output bytes.Buffer
	selected, err := promptMonitorModes(strings.NewReader("\n"), &output, monitors, []int{0})
	if err != nil {
		t.Fatalf("promptMonitorModes returned error: %v", err)
	}

	want := map[int]monitorMode{
		0: {Width: 2560, Height: 1440, RefreshRate: 165},
	}

	if !reflect.DeepEqual(selected, want) {
		t.Fatalf("unexpected selected modes: got %#v want %#v", selected, want)
	}
}

func TestAvailableMonitorModesFallsBackToCurrentViaRenderer(t *testing.T) {
	monitors := []monitor{
		{
			Name:        "DP-1",
			Width:       2560,
			Height:      1440,
			RefreshRate: 165,
			Scale:       1,
		},
	}

	lines := renderPositionedConfigLines(monitors, buildActiveMonitorConfigs(monitors, []int{0}, nil), horizontalDirection)
	want := []string{"monitor = DP-1, 2560x1440@165, 0x0, 1"}

	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("unexpected rendered lines: got %#v want %#v", lines, want)
	}
}
