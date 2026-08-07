package cli

import (
	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"testing"
)

func TestQuickPresetLayoutID(t *testing.T) {
	tests := []struct {
		preset  string
		want    int
		wantErr bool
	}{
		{preset: "laptop", want: 1},
		{preset: "external", want: 2},
		{preset: "dual", want: 3},
		{preset: "triple", want: 4},
		{preset: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			got, err := quickPresetLayoutID(tt.preset)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("quickPresetLayoutID returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("unexpected layout ID: got %d want %d", got, tt.want)
			}
		})
	}
}

func TestBestMonitorModeSelectsHighestRefresh(t *testing.T) {
	mon := hypr.Monitor{
		Width:          2560,
		Height:         1440,
		RefreshRate:    60,
		AvailableModes: []string{"2560x1440@60.00Hz", "1920x1080@144.00Hz", "2560x1440@165.00Hz"},
	}

	got := layout.BestMode(mon)
	want := layout.Mode{Width: 2560, Height: 1440, RefreshRate: 165}

	if got != want {
		t.Fatalf("unexpected best mode: got %#v want %#v", got, want)
	}
}

func TestBestMonitorModeFallsBackToCurrent(t *testing.T) {
	mon := hypr.Monitor{
		Width:       2880,
		Height:      1800,
		RefreshRate: 120,
	}

	got := layout.BestMode(mon)
	want := layout.Mode{Width: 2880, Height: 1800, RefreshRate: 120}

	if got != want {
		t.Fatalf("unexpected fallback mode: got %#v want %#v", got, want)
	}
}

func TestParseQuickOptions(t *testing.T) {
	got, err := parseQuickOptions([]string{"--yes", "--direction", "top-bottom", "--order", "2 1"})
	if err != nil {
		t.Fatalf("parseQuickOptions returned error: %v", err)
	}

	if !got.AutoYes || got.NoReload {
		t.Fatalf("unexpected shared options: %#v", got)
	}

	if got.Direction != layout.TopToBottom {
		t.Fatalf("unexpected direction: %q", got.Direction)
	}

	if got.Order != "2 1" {
		t.Fatalf("unexpected order: %q", got.Order)
	}
}

func TestParseQuickOptionsDefaultsToLeftRight(t *testing.T) {
	got, err := parseQuickOptions(nil)
	if err != nil {
		t.Fatalf("parseQuickOptions returned error: %v", err)
	}

	if got.Direction != layout.LeftToRight {
		t.Fatalf("unexpected default direction: %q", got.Direction)
	}
}

func TestParseQuickOptionsTransformAndVRR(t *testing.T) {
	got, err := parseQuickOptions([]string{"--transform", "1", "--vrr", "2"})
	if err != nil {
		t.Fatalf("parseQuickOptions returned error: %v", err)
	}

	if got.Transform != 1 || got.VRR != 2 {
		t.Fatalf("unexpected settings: %#v", got)
	}
}

func TestParseQuickOptionsKeepsCurrentSettingsByDefault(t *testing.T) {
	got, err := parseQuickOptions(nil)
	if err != nil {
		t.Fatalf("parseQuickOptions returned error: %v", err)
	}

	if got.Transform != keepCurrentSetting || got.VRR != keepCurrentSetting {
		t.Fatalf("unexpected default settings: %#v", got)
	}
}

func TestApplyDisplayOverrides(t *testing.T) {
	configs := []layout.MonitorConfig{
		{Index: 0, Transform: 3, VRR: 1},
		{Index: 1},
	}

	applyDisplayOverrides(configs, 1, keepCurrentSetting)

	if configs[0].Transform != 1 || configs[1].Transform != 1 {
		t.Fatalf("transform override not applied: %#v", configs)
	}

	if configs[0].VRR != 1 || configs[1].VRR != 0 {
		t.Fatalf("vrr should have been kept: %#v", configs)
	}
}

func TestParseQuickOptionsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"--bad"}},
		{name: "missing direction value", args: []string{"--direction"}},
		{name: "invalid direction value", args: []string{"--direction", "diagonal"}},
		{name: "missing order value", args: []string{"--order"}},
		{name: "transform out of range", args: []string{"--transform", "8"}},
		{name: "transform not a number", args: []string{"--transform", "left"}},
		{name: "vrr out of range", args: []string{"--vrr", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseQuickOptions(tt.args); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseDirectionName(t *testing.T) {
	tests := []struct {
		value string
		want  layout.Direction
	}{
		{value: "left-right", want: layout.LeftToRight},
		{value: "right-left", want: layout.RightToLeft},
		{value: "top-bottom", want: layout.TopToBottom},
		{value: "bottom-top", want: layout.BottomToTop},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := layout.ParseDirection(tt.value)
			if err != nil {
				t.Fatalf("parseDirectionName returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("unexpected direction: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestParseCommandOptions(t *testing.T) {
	got, err := parseCommandOptions([]string{"--yes", "--no-reload"})
	if err != nil {
		t.Fatalf("parseCommandOptions returned error: %v", err)
	}

	if !got.AutoYes || !got.NoReload {
		t.Fatalf("unexpected options: %#v", got)
	}

	if _, err := parseCommandOptions([]string{"--bad"}); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
