package main

import "testing"

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
	mon := monitor{
		Width:          2560,
		Height:         1440,
		RefreshRate:    60,
		AvailableModes: []string{"2560x1440@60.00Hz", "1920x1080@144.00Hz", "2560x1440@165.00Hz"},
	}

	got := bestMonitorMode(mon)
	want := monitorMode{Width: 2560, Height: 1440, RefreshRate: 165}

	if got != want {
		t.Fatalf("unexpected best mode: got %#v want %#v", got, want)
	}
}

func TestBestMonitorModeFallsBackToCurrent(t *testing.T) {
	mon := monitor{
		Width:       2880,
		Height:      1800,
		RefreshRate: 120,
	}

	got := bestMonitorMode(mon)
	want := monitorMode{Width: 2880, Height: 1800, RefreshRate: 120}

	if got != want {
		t.Fatalf("unexpected fallback mode: got %#v want %#v", got, want)
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
