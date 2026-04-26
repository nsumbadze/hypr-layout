package main

import (
	"reflect"
	"testing"
)

func TestRenderPositionedConfigLinesHorizontal(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}

	activeConfigs := []activeMonitorConfig{
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, horizontalDirection)
	want := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = HDMI-A-1, 1920x1080@60, 2560x0, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesVertical(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}

	activeConfigs := []activeMonitorConfig{
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, verticalDirection)
	want := []string{
		"monitor = HDMI-A-1, 1920x1080@60, 0x0, 1",
		"monitor = DP-1, 2560x1440@165, 0x1080, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestReorderActiveConfigsValidation(t *testing.T) {
	activeConfigs := []activeMonitorConfig{
		{Index: 0, Mode: monitorMode{Width: 2880, Height: 1800, RefreshRate: 120}},
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	ordered, err := reorderActiveConfigs("2 1 3", activeConfigs)
	if err != nil {
		t.Fatalf("reorderActiveConfigs returned error: %v", err)
	}

	want := []activeMonitorConfig{
		activeConfigs[1],
		activeConfigs[0],
		activeConfigs[2],
	}

	if !reflect.DeepEqual(ordered, want) {
		t.Fatalf("unexpected order:\ngot:  %#v\nwant: %#v", ordered, want)
	}

	tests := []string{"1 1 2", "1 2", "1 2 4", "a b c"}
	for _, input := range tests {
		if _, err := reorderActiveConfigs(input, activeConfigs); err == nil {
			t.Fatalf("expected validation error for input %q", input)
		}
	}
}
