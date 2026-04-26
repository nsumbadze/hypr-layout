package main

import (
	"reflect"
	"testing"
)

// ── renderPositionedConfigLines ───────────────────────────────────────────────

func TestRenderPositionedConfigLinesLeftToRight(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	activeConfigs := []activeMonitorConfig{
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, leftToRight)
	want := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = HDMI-A-1, 1920x1080@60, 2560x0, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("left-to-right:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesRightToLeft(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	// A=DP-1 at 0x0, B=HDMI-A-1 at -1920x0 (shifts left by its own width)
	activeConfigs := []activeMonitorConfig{
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, rightToLeft)
	want := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = HDMI-A-1, 1920x1080@60, -1920x0, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("right-to-left:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesRightToLeftThreeMonitors(t *testing.T) {
	monitors := []monitor{
		{Name: "A", Scale: 1},
		{Name: "B", Scale: 1},
		{Name: "C", Scale: 1},
	}
	// A at 0, B at -1920, C at -1920-2560 = -4480
	activeConfigs := []activeMonitorConfig{
		{Index: 0, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 1, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 2, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 144}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, rightToLeft)
	want := []string{
		"monitor = A, 2560x1440@165, 0x0, 1",
		"monitor = B, 1920x1080@60, -1920x0, 1",
		"monitor = C, 2560x1440@144, -4480x0, 1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("right-to-left three monitors:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesTopToBottom(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	activeConfigs := []activeMonitorConfig{
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, topToBottom)
	want := []string{
		"monitor = HDMI-A-1, 1920x1080@60, 0x0, 1",
		"monitor = DP-1, 2560x1440@165, 0x1080, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("top-to-bottom:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesBottomToTop(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	// A=HDMI-A-1 at 0x0, B=DP-1 at 0x-1440 (shifts up by its own height)
	activeConfigs := []activeMonitorConfig{
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, bottomToTop)
	want := []string{
		"monitor = HDMI-A-1, 1920x1080@60, 0x0, 1",
		"monitor = DP-1, 2560x1440@165, 0x-1440, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bottom-to-top:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesBottomToTopThreeMonitors(t *testing.T) {
	monitors := []monitor{
		{Name: "A", Scale: 1},
		{Name: "B", Scale: 1},
		{Name: "C", Scale: 1},
	}
	// A at 0, B at 0x-1080, C at 0x(-1080-1440)=0x-2520
	activeConfigs := []activeMonitorConfig{
		{Index: 0, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 1, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 2, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 144}},
	}

	got := renderPositionedConfigLines(monitors, activeConfigs, bottomToTop)
	want := []string{
		"monitor = A, 2560x1440@165, 0x0, 1",
		"monitor = B, 1920x1080@60, 0x-1080, 1",
		"monitor = C, 2560x1440@144, 0x-2520, 1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bottom-to-top three monitors:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// ── reorderActiveConfigs ──────────────────────────────────────────────────────

func TestReorderActiveConfigsValid(t *testing.T) {
	configs := []activeMonitorConfig{
		{Index: 0, Mode: monitorMode{Width: 2880, Height: 1800, RefreshRate: 120}},
		{Index: 1, Mode: monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: monitorMode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got, err := reorderActiveConfigs("2 1 3", configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []activeMonitorConfig{configs[1], configs[0], configs[2]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected order:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestReorderActiveConfigsDuplicateIndex(t *testing.T) {
	configs := []activeMonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := reorderActiveConfigs("1 1 2", configs); err == nil {
		t.Fatal("expected error for duplicate index, got nil")
	}
}

func TestReorderActiveConfigsMissingMonitor(t *testing.T) {
	configs := []activeMonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := reorderActiveConfigs("1 2", configs); err == nil {
		t.Fatal("expected error for missing monitor (too few indices), got nil")
	}
}

func TestReorderActiveConfigsOutOfRange(t *testing.T) {
	configs := []activeMonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := reorderActiveConfigs("1 2 4", configs); err == nil {
		t.Fatal("expected error for out-of-range index 4, got nil")
	}
}

func TestReorderActiveConfigsNonNumeric(t *testing.T) {
	configs := []activeMonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := reorderActiveConfigs("a b c", configs); err == nil {
		t.Fatal("expected error for non-numeric input, got nil")
	}
}
