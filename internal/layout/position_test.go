package layout

import (
	"reflect"
	"testing"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
)

// ── PositionedRules ───────────────────────────────────────────────────────────

func TestRenderPositionedConfigLinesLeftToRight(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	activeConfigs := []MonitorConfig{
		{Index: 1, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, LeftToRight))
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
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	// A=DP-1 at 0x0, B=HDMI-A-1 at -1920x0 (shifts left by its own width)
	activeConfigs := []MonitorConfig{
		{Index: 1, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, RightToLeft))
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
	monitors := []hypr.Monitor{
		{Name: "A", Scale: 1},
		{Name: "B", Scale: 1},
		{Name: "C", Scale: 1},
	}
	// A at 0, B at -1920, C at -1920-2560 = -4480
	activeConfigs := []MonitorConfig{
		{Index: 0, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 1, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 2, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 144}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, RightToLeft))
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
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	activeConfigs := []MonitorConfig{
		{Index: 2, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 1, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, TopToBottom))
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
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1},
		{Name: "HDMI-A-1", Scale: 1},
	}
	// A=HDMI-A-1 at 0x0, B=DP-1 at 0x-1440 (shifts up by its own height)
	activeConfigs := []MonitorConfig{
		{Index: 2, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 1, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, BottomToTop))
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
	monitors := []hypr.Monitor{
		{Name: "A", Scale: 1},
		{Name: "B", Scale: 1},
		{Name: "C", Scale: 1},
	}
	// A at 0, B at 0x-1080, C at 0x(-1080-1440)=0x-2520
	activeConfigs := []MonitorConfig{
		{Index: 0, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 1, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{Index: 2, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 144}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, BottomToTop))
	want := []string{
		"monitor = A, 2560x1440@165, 0x0, 1",
		"monitor = B, 1920x1080@60, 0x-1080, 1",
		"monitor = C, 2560x1440@144, 0x-2520, 1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bottom-to-top three monitors:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestRenderPositionedConfigLinesEmitsTransformAndVRR(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "DP-1", Scale: 1},
		{Name: "DP-2", Scale: 1},
	}
	// DP-1 is rotated 90°: its effective width becomes its height (1440),
	// so DP-2 must start at x=1440.
	activeConfigs := []MonitorConfig{
		{Index: 0, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}, Transform: 1, VRR: 1},
		{Index: 1, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got := ConfLines(PositionedRules(monitors, activeConfigs, LeftToRight))
	want := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1, transform, 1, vrr, 1",
		"monitor = DP-2, 1920x1080@60, 1440x0, 1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("transform/vrr:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestEffectiveModeSize(t *testing.T) {
	mode := Mode{Width: 2560, Height: 1440}

	for transform, wantSwap := range map[int]bool{0: false, 1: true, 2: false, 3: true, 4: false, 5: true, 6: false, 7: true} {
		w, h := EffectiveSize(mode, transform)
		if wantSwap && (w != 1440 || h != 2560) {
			t.Fatalf("transform %d: expected swapped size, got %dx%d", transform, w, h)
		}
		if !wantSwap && (w != 2560 || h != 1440) {
			t.Fatalf("transform %d: expected original size, got %dx%d", transform, w, h)
		}
	}
}

func TestBuildActiveMonitorConfigsPreservesTransformAndVRR(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Transform: 3, VRR: true},
	}

	configs := BuildConfigs(monitors, []int{0}, nil)
	if len(configs) != 1 {
		t.Fatalf("unexpected config count: %d", len(configs))
	}

	if configs[0].Transform != 3 || configs[0].VRR != 1 {
		t.Fatalf("unexpected settings: %#v", configs[0])
	}
}

// ── renderMirroredConfigLines ─────────────────────────────────────────────────

func TestRenderMirroredConfigLines(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Scale: 1.5},
		{Name: "DP-1", Scale: 1, Focused: true},
		{Name: "HDMI-A-1", Scale: 1},
	}
	activeConfigs := []MonitorConfig{
		{Index: 0, Mode: Mode{Width: 2880, Height: 1800, RefreshRate: 120}},
		{Index: 1, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
	}

	got := ConfLines(MirroredRules(monitors, activeConfigs, 1))
	want := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = eDP-1, 2880x1800@120, 0x0, 1.5, mirror, DP-1",
		"monitor = HDMI-A-1, disable",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mirrored:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// ── reorderActiveConfigs ──────────────────────────────────────────────────────

func TestReorderActiveConfigsValid(t *testing.T) {
	configs := []MonitorConfig{
		{Index: 0, Mode: Mode{Width: 2880, Height: 1800, RefreshRate: 120}},
		{Index: 1, Mode: Mode{Width: 2560, Height: 1440, RefreshRate: 165}},
		{Index: 2, Mode: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
	}

	got, err := Reorder("2 1 3", configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []MonitorConfig{configs[1], configs[0], configs[2]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected order:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestReorderActiveConfigsDuplicateIndex(t *testing.T) {
	configs := []MonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := Reorder("1 1 2", configs); err == nil {
		t.Fatal("expected error for duplicate index, got nil")
	}
}

func TestReorderActiveConfigsMissingMonitor(t *testing.T) {
	configs := []MonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := Reorder("1 2", configs); err == nil {
		t.Fatal("expected error for missing monitor (too few indices), got nil")
	}
}

func TestReorderActiveConfigsOutOfRange(t *testing.T) {
	configs := []MonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := Reorder("1 2 4", configs); err == nil {
		t.Fatal("expected error for out-of-range index 4, got nil")
	}
}

func TestReorderActiveConfigsNonNumeric(t *testing.T) {
	configs := []MonitorConfig{
		{Index: 0},
		{Index: 1},
		{Index: 2},
	}
	if _, err := Reorder("a b c", configs); err == nil {
		t.Fatal("expected error for non-numeric input, got nil")
	}
}
