package main

import (
	"reflect"
	"testing"
)

func TestParseLayoutSelection(t *testing.T) {
	options := layoutOptions()

	selection, err := parseLayoutSelection("3\n", options)
	if err != nil {
		t.Fatalf("parseLayoutSelection returned error: %v", err)
	}

	if selection.ID != 3 || selection.Name != "Dual horizontal" {
		t.Fatalf("unexpected selection: %+v", selection)
	}
}

func TestBuildPreviewConfigDualHorizontal(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Width: 2880, Height: 1800, RefreshRate: 120, Scale: 1.5},
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
		{Name: "HDMI-A-1", Width: 1920, Height: 1080, RefreshRate: 60, Scale: 1},
	}

	lines, err := buildPreviewConfig(monitors, 3)
	if err != nil {
		t.Fatalf("buildPreviewConfig returned error: %v", err)
	}

	want := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = HDMI-A-1, 1920x1080@60, 2560x0, 1",
		"monitor = eDP-1, disable",
	}

	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("unexpected lines:\nwant: %#v\ngot:  %#v", want, lines)
	}
}

func TestBuildPreviewConfigTripleHorizontalNeedsThreeMonitors(t *testing.T) {
	monitors := []monitor{
		{Name: "eDP-1", Width: 2880, Height: 1800, RefreshRate: 120, Scale: 1.5},
		{Name: "DP-1", Width: 2560, Height: 1440, RefreshRate: 165, Scale: 1},
	}

	_, err := buildPreviewConfig(monitors, 4)
	if err == nil {
		t.Fatal("expected error for missing third monitor")
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
