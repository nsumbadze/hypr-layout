package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
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

	if err := saveProfile(dir, "work_setup", lines); err != nil {
		t.Fatalf("saveProfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "work_setup.conf"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	wantContent := "monitor = DP-1, 2560x1440@165, 0x0, 1\nmonitor = eDP-1, disable\n"
	if string(data) != wantContent {
		t.Fatalf("unexpected saved content: got %q want %q", string(data), wantContent)
	}

	loaded, err := loadProfile(dir, "work_setup")
	if err != nil {
		t.Fatalf("loadProfile returned error: %v", err)
	}

	if !reflect.DeepEqual(loaded, lines) {
		t.Fatalf("unexpected loaded lines: got %#v want %#v", loaded, lines)
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
