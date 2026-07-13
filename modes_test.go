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

func TestParseModeStrategy(t *testing.T) {
	tests := []struct {
		value   string
		want    modeStrategy
		wantErr bool
	}{
		{value: "current", want: modeStrategyCurrent},
		{value: "best", want: modeStrategyBest},
		{value: "preferred", want: modeStrategyPreferred},
		{value: "highres", want: modeStrategyHighres},
		{value: "highrr", want: modeStrategyHighrr},
		{value: "fastest", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := parseModeStrategy(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("parseModeStrategy returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("unexpected strategy: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestResolveModeStrategy(t *testing.T) {
	mon := monitor{
		Width:       1920,
		Height:      1080,
		RefreshRate: 60,
		AvailableModes: []string{
			"2560x1440@60.00Hz",
			"1920x1080@144.00Hz",
			"2560x1440@120.00Hz",
		},
	}

	tests := []struct {
		strategy modeStrategy
		want     monitorMode
	}{
		{strategy: modeStrategyCurrent, want: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{strategy: modeStrategyPreferred, want: monitorMode{Width: 2560, Height: 1440, RefreshRate: 60}},
		{strategy: modeStrategyHighres, want: monitorMode{Width: 2560, Height: 1440, RefreshRate: 120}},
		{strategy: modeStrategyHighrr, want: monitorMode{Width: 1920, Height: 1080, RefreshRate: 144}},
		{strategy: modeStrategyBest, want: monitorMode{Width: 1920, Height: 1080, RefreshRate: 144}},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			got := resolveModeStrategy(mon, tt.strategy)
			if got != tt.want {
				t.Fatalf("unexpected mode: got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveModeStrategyFallsBackToCurrent(t *testing.T) {
	mon := monitor{Width: 2880, Height: 1800, RefreshRate: 120}
	current := monitorMode{Width: 2880, Height: 1800, RefreshRate: 120}

	for _, strategy := range []modeStrategy{modeStrategyPreferred, modeStrategyHighres, modeStrategyHighrr} {
		if got := resolveModeStrategy(mon, strategy); got != current {
			t.Fatalf("strategy %q: unexpected fallback mode: got %#v want %#v", strategy, got, current)
		}
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

	lines := renderPositionedConfigLines(monitors, buildActiveMonitorConfigs(monitors, []int{0}, nil), leftToRight)
	want := []string{"monitor = DP-1, 2560x1440@165, 0x0, 1"}

	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("unexpected rendered lines: got %#v want %#v", lines, want)
	}
}
