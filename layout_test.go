package main

import "testing"

func TestActiveIndexesForLayoutDualHorizontal(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Width: 2880, Height: 1800, RefreshRate: 120, Scale: 1.5},
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
		{Name: "HDMI-A-1", Width: 1920, Height: 1080, RefreshRate: 60, Scale: 1},
	}

	indexes, err := activeIndexesForLayout(monitors, 3)
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
	monitors := []monitor{
		{Name: "eDP-1", Width: 2880, Height: 1800, RefreshRate: 120, Scale: 1.5},
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
	}

	_, err := activeIndexesForLayout(monitors, 4)
	if err == nil {
		t.Fatal("expected error for missing third monitor")
	}
}

func TestActiveIndexesForLayoutMirror(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1"},
		{Name: "DP-1"},
	}

	indexes, err := activeIndexesForLayout(monitors, layoutMirror)
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
	monitors := []monitor{{Name: "eDP-1"}}

	if _, err := activeIndexesForLayout(monitors, layoutMirror); err == nil {
		t.Fatal("expected error for single monitor mirror")
	}
}

func TestMirrorSourceIndexPrefersFocused(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1"},
		{Name: "DP-1", Focused: true},
	}

	if got := mirrorSourceIndex(monitors, []int{0, 1}); got != 1 {
		t.Fatalf("unexpected source index: got %d want 1", got)
	}
}

func TestMirrorSourceIndexFallsBackToFirstActive(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1"},
		{Name: "DP-1"},
	}

	if got := mirrorSourceIndex(monitors, []int{0, 1}); got != 0 {
		t.Fatalf("unexpected source index: got %d want 0", got)
	}
}

func TestParseConfirmation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{name: "yes short", input: "y\n", want: true},
		{name: "yes long", input: "yes", want: true},
		{name: "no short", input: "n", want: false},
		{name: "no long", input: "no\n", want: false},
		{name: "invalid", input: "maybe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfirmation(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("parseConfirmation returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("unexpected confirmation: got %v want %v", got, tt.want)
			}
		})
	}
}
