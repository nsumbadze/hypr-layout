package layout

import (
	"reflect"
	"testing"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
)

func TestFormatMonitorMode(t *testing.T) {
	mode := Mode{Width: 2560, Height: 1440, RefreshRate: 165}

	got := FormatMode(mode)
	want := "2560x1440@165"

	if got != want {
		t.Fatalf("unexpected formatted mode: got %q want %q", got, want)
	}
}

func TestParseModeStrategy(t *testing.T) {
	tests := []struct {
		value   string
		want    Strategy
		wantErr bool
	}{
		{value: "current", want: StrategyCurrent},
		{value: "best", want: StrategyBest},
		{value: "preferred", want: StrategyPreferred},
		{value: "highres", want: StrategyHighres},
		{value: "highrr", want: StrategyHighrr},
		{value: "fastest", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := ParseStrategy(tt.value)
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
	mon := hypr.Monitor{
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
		strategy Strategy
		want     Mode
	}{
		{strategy: StrategyCurrent, want: Mode{Width: 1920, Height: 1080, RefreshRate: 60}},
		{strategy: StrategyPreferred, want: Mode{Width: 2560, Height: 1440, RefreshRate: 60}},
		{strategy: StrategyHighres, want: Mode{Width: 2560, Height: 1440, RefreshRate: 120}},
		{strategy: StrategyHighrr, want: Mode{Width: 1920, Height: 1080, RefreshRate: 144}},
		{strategy: StrategyBest, want: Mode{Width: 1920, Height: 1080, RefreshRate: 144}},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			got := ResolveStrategy(mon, tt.strategy)
			if got != tt.want {
				t.Fatalf("unexpected mode: got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveModeStrategyFallsBackToCurrent(t *testing.T) {
	mon := hypr.Monitor{Width: 2880, Height: 1800, RefreshRate: 120}
	current := Mode{Width: 2880, Height: 1800, RefreshRate: 120}

	for _, strategy := range []Strategy{StrategyPreferred, StrategyHighres, StrategyHighrr} {
		if got := ResolveStrategy(mon, strategy); got != current {
			t.Fatalf("strategy %q: unexpected fallback mode: got %#v want %#v", strategy, got, current)
		}
	}
}

func TestAvailableMonitorModesFallsBackToCurrentViaRenderer(t *testing.T) {
	monitors := []hypr.Monitor{
		{
			Name:        "DP-1",
			Width:       2560,
			Height:      1440,
			RefreshRate: 165,
			Scale:       1,
		},
	}

	lines := PositionedLines(monitors, BuildConfigs(monitors, []int{0}, nil), LeftToRight)
	want := []string{"monitor = DP-1, 2560x1440@165, 0x0, 1"}

	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("unexpected rendered lines: got %#v want %#v", lines, want)
	}
}
