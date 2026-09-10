package layout

import (
	"reflect"
	"testing"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
)

func TestCurrentArrangement(t *testing.T) {
	tests := []struct {
		name      string
		monitors  []hypr.Monitor
		active    []int
		wantOrder []int
		wantDir   Direction
	}{
		{
			name: "left to right sorted by x regardless of detection order",
			monitors: []hypr.Monitor{
				{Name: "DP-1", X: 2560, Y: 0},
				{Name: "HDMI-A-2", X: 0, Y: 0},
			},
			active:    []int{0, 1},
			wantOrder: []int{1, 0},
			wantDir:   LeftToRight,
		},
		{
			name: "origin monitor on the right means right to left",
			monitors: []hypr.Monitor{
				{Name: "DP-1", X: 0, Y: 0},
				{Name: "HDMI-A-2", X: -2560, Y: 0},
			},
			active:    []int{0, 1},
			wantOrder: []int{0, 1},
			wantDir:   RightToLeft,
		},
		{
			name: "three monitors stacked leftwards",
			monitors: []hypr.Monitor{
				{Name: "A", X: 0},
				{Name: "B", X: -1920},
				{Name: "C", X: -4480},
			},
			active:    []int{2, 0, 1},
			wantOrder: []int{0, 1, 2},
			wantDir:   RightToLeft,
		},
		{
			name: "top to bottom",
			monitors: []hypr.Monitor{
				{Name: "DP-1", X: 0, Y: 1080},
				{Name: "HDMI-A-2", X: 0, Y: 0},
			},
			active:    []int{0, 1},
			wantOrder: []int{1, 0},
			wantDir:   TopToBottom,
		},
		{
			name: "bottom to top",
			monitors: []hypr.Monitor{
				{Name: "DP-1", X: 0, Y: 0},
				{Name: "HDMI-A-2", X: 0, Y: -1440},
			},
			active:    []int{0, 1},
			wantOrder: []int{0, 1},
			wantDir:   BottomToTop,
		},
		{
			name: "inactive monitors do not affect the axis",
			monitors: []hypr.Monitor{
				{Name: "eDP-1", X: 0, Y: 5000},
				{Name: "DP-1", X: 0, Y: 0},
				{Name: "HDMI-A-2", X: 2560, Y: 0},
			},
			active:    []int{1, 2},
			wantOrder: []int{1, 2},
			wantDir:   LeftToRight,
		},
		{
			name: "unknown positions keep detection order",
			monitors: []hypr.Monitor{
				{Name: "DP-1"},
				{Name: "HDMI-A-2"},
			},
			active:    []int{0, 1},
			wantOrder: []int{0, 1},
			wantDir:   LeftToRight,
		},
		{
			name:      "single monitor",
			monitors:  []hypr.Monitor{{Name: "eDP-1", X: 100, Y: 100}},
			active:    []int{0},
			wantOrder: []int{0},
			wantDir:   LeftToRight,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			active := append([]int(nil), tc.active...)
			order, dir := CurrentArrangement(tc.monitors, active)
			if !reflect.DeepEqual(order, tc.wantOrder) || dir != tc.wantDir {
				t.Fatalf("got order %v dir %q, want order %v dir %q", order, dir, tc.wantOrder, tc.wantDir)
			}
			if !reflect.DeepEqual(active, tc.active) {
				t.Fatalf("input slice was modified: %v", active)
			}
		})
	}
}

// Whatever CurrentArrangement reports must regenerate the positions the
// monitors already have, so applying without edits changes nothing.
func TestCurrentArrangementReproducesPositions(t *testing.T) {
	monitors := []hypr.Monitor{
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 60, Scale: 1, X: 0, Y: 0},
		{Name: "HDMI-A-2", Width: 2560, Height: 1440, RefreshRate: 60, Scale: 1, X: -2560, Y: 0},
	}

	order, dir := CurrentArrangement(monitors, []int{0, 1})
	got := ConfLines(PositionedRules(monitors, BuildConfigs(monitors, order, nil), dir))
	want := []string{
		"monitor = DP-1, 2560x1440@60, 0x0, 1",
		"monitor = HDMI-A-2, 2560x1440@60, -2560x0, 1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
