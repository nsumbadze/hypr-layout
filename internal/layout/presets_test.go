package layout

import (
	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"testing"
)

func TestActiveIndexesForLayoutDualHorizontal(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Width: 2880, Height: 1800, RefreshRate: 120, Scale: 1.5},
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
		{Name: "HDMI-A-1", Width: 1920, Height: 1080, RefreshRate: 60, Scale: 1},
	}

	indexes, err := ActiveIndexes(monitors, 3)
	if err != nil {
		t.Fatalf("activeIndexesForLayout returned error: %v", err)
	}

	want := []int{1, 2}
	if len(indexes) != len(want) {
		t.Fatalf("unexpected index count: got %d want %d", len(indexes), len(want))
	}

	for i := range want {
		if indexes[i] != want[i] {
			t.Fatalf("unexpected indexes: got %#v want %#v", indexes, want)
		}
	}
}

func TestActiveIndexesForLayoutTripleHorizontalNeedsThreeMonitors(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1", Width: 2880, Height: 1800, RefreshRate: 120, Scale: 1.5},
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
	}

	_, err := ActiveIndexes(monitors, 4)
	if err == nil {
		t.Fatal("expected error for missing third monitor")
	}
}

func TestActiveIndexesForLayoutMirror(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1"},
		{Name: "DP-1"},
	}

	indexes, err := ActiveIndexes(monitors, Mirror)
	if err != nil {
		t.Fatalf("activeIndexesForLayout returned error: %v", err)
	}

	want := []int{0, 1}
	if len(indexes) != len(want) {
		t.Fatalf("unexpected index count: got %d want %d", len(indexes), len(want))
	}

	for i := range want {
		if indexes[i] != want[i] {
			t.Fatalf("unexpected indexes: got %#v want %#v", indexes, want)
		}
	}
}

func TestActiveIndexesForLayoutMirrorNeedsTwoMonitors(t *testing.T) {
	monitors := []hypr.Monitor{{Name: "eDP-1"}}

	if _, err := ActiveIndexes(monitors, Mirror); err == nil {
		t.Fatal("expected error for single monitor mirror")
	}
}

func TestMirrorSourceIndexPrefersFocused(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1"},
		{Name: "DP-1", Focused: true},
	}

	if got := MirrorSourceIndex(monitors, []int{0, 1}); got != 1 {
		t.Fatalf("unexpected source index: got %d want 1", got)
	}
}

func TestMirrorSourceIndexFallsBackToFirstActive(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "eDP-1"},
		{Name: "DP-1"},
	}

	if got := MirrorSourceIndex(monitors, []int{0, 1}); got != 0 {
		t.Fatalf("unexpected source index: got %d want 0", got)
	}
}
