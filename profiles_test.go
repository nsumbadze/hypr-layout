package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid letters", input: "work"},
		{name: "valid mixed", input: "desk_setup-1"},
		{name: "empty", input: "", wantErr: true},
		{name: "space", input: "desk setup", wantErr: true},
		{name: "slash", input: "../work", wantErr: true},
		{name: "dot", input: "work.conf", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfileName(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSaveProfileAndLoadProfile(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = eDP-1, disable",
	}
	meta := profileMetadata{
		SavedAt:   time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC),
		Direction: "left-to-right",
		Monitors:  []string{"DP-1 2560x1440@165"},
	}

	if err := saveProfile(dir, "work_setup", lines, meta); err != nil {
		t.Fatalf("saveProfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "work_setup.conf"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# hypr-layout profile\n") {
		t.Fatalf("missing metadata header: %q", content)
	}
	if !strings.HasSuffix(content, "monitor = DP-1, 2560x1440@165, 0x0, 1\nmonitor = eDP-1, disable\n") {
		t.Fatalf("unexpected config lines: %q", content)
	}

	loaded, err := loadProfile(dir, "work_setup")
	if err != nil {
		t.Fatalf("loadProfile returned error: %v", err)
	}

	if !reflect.DeepEqual(loaded, lines) {
		t.Fatalf("unexpected loaded lines: got %#v want %#v", loaded, lines)
	}
}

func TestProfileMetadataRoundTrip(t *testing.T) {
	meta := profileMetadata{
		SavedAt:   time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC),
		Direction: "top-to-bottom",
		Monitors:  []string{"DP-1 2560x1440@165", "eDP-1 2880x1800@120"},
	}

	content := strings.Join(append(formatProfileHeader(meta), "monitor = DP-1, 2560x1440@165, 0x0, 1"), "\n")
	got := parseProfileMetadata(content)

	if !got.SavedAt.Equal(meta.SavedAt) {
		t.Fatalf("unexpected saved time: got %v want %v", got.SavedAt, meta.SavedAt)
	}

	if got.Direction != meta.Direction {
		t.Fatalf("unexpected direction: got %q want %q", got.Direction, meta.Direction)
	}

	if !reflect.DeepEqual(got.Monitors, meta.Monitors) {
		t.Fatalf("unexpected monitors: got %#v want %#v", got.Monitors, meta.Monitors)
	}
}

func TestParseProfileMetadataEmptyForPlainProfile(t *testing.T) {
	got := parseProfileMetadata("monitor = DP-1, 2560x1440@165, 0x0, 1\n")

	if !got.SavedAt.IsZero() || got.Direction != "" || got.Monitors != nil {
		t.Fatalf("expected empty metadata, got %#v", got)
	}
}

func TestListProfileSummaries(t *testing.T) {
	dir := t.TempDir()
	meta := profileMetadata{Direction: "mirror", Monitors: []string{"DP-1 2560x1440@165"}}

	if err := saveProfile(dir, "docked", []string{"monitor = DP-1, 2560x1440@165, 0x0, 1"}, meta); err != nil {
		t.Fatalf("saveProfile returned error: %v", err)
	}

	summaries, err := listProfileSummaries(dir)
	if err != nil {
		t.Fatalf("listProfileSummaries returned error: %v", err)
	}

	if len(summaries) != 1 || summaries[0].Name != "docked" {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}

	if summaries[0].Meta.Direction != "mirror" {
		t.Fatalf("unexpected metadata: %#v", summaries[0].Meta)
	}
}

func TestListProfiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"zeta.conf":  "monitor = DP-1, disable\n",
		"alpha.conf": "monitor = eDP-1, 2880x1800@120, 0x0, 1.5\n",
		"notes.txt":  "ignore\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), monitorConfigMode); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	profiles, err := listProfiles(dir)
	if err != nil {
		t.Fatalf("listProfiles returned error: %v", err)
	}

	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(profiles, want) {
		t.Fatalf("unexpected profiles: got %#v want %#v", profiles, want)
	}
}

func TestLoadProfileRejectsInvalidName(t *testing.T) {
	dir := t.TempDir()

	_, err := loadProfile(dir, "../bad")
	if err == nil {
		t.Fatal("expected error")
	}
}
