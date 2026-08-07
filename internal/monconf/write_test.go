package monconf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupPathFor(t *testing.T) {
	got := backupPathFor("/tmp/monitors.conf", time.Date(2026, time.April, 26, 13, 14, 15, 0, time.UTC))
	want := "/tmp/monitors.conf.backup-20260426-131415"

	if got != want {
		t.Fatalf("unexpected backup path: got %q want %q", got, want)
	}
}

func TestGenerateConfigContent(t *testing.T) {
	lines := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = HDMI-A-1, 1920x1080@60, 2560x0, 1",
		"monitor = eDP-1, disable",
	}

	got := generateContent(lines)
	want := "monitor = DP-1, 2560x1440@165, 0x0, 1\nmonitor = HDMI-A-1, 1920x1080@60, 2560x0, 1\nmonitor = eDP-1, disable\n"

	if got != want {
		t.Fatalf("unexpected config content:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestApplyMonitorConfigCreatesBackupAndWritesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "hypr", "monitors.conf")

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := os.WriteFile(configPath, []byte("old config\n"), fileMode); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	lines := []string{
		"monitor = DP-1, 2560x1440@165, 0x0, 1",
		"monitor = eDP-1, disable",
	}
	now := time.Date(2026, time.April, 26, 13, 14, 15, 0, time.UTC)

	result, err := Apply(configPath, lines, now)
	if err != nil {
		t.Fatalf("applyMonitorConfig returned error: %v", err)
	}

	wantBackupPath := filepath.Join(dir, "hypr", "monitors.conf.backup-20260426-131415")
	if result.BackupPath != wantBackupPath {
		t.Fatalf("unexpected backup path: got %q want %q", result.BackupPath, wantBackupPath)
	}

	backupData, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("ReadFile backup returned error: %v", err)
	}

	if string(backupData) != "old config\n" {
		t.Fatalf("unexpected backup content: %q", string(backupData))
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config returned error: %v", err)
	}

	wantConfig := "monitor = DP-1, 2560x1440@165, 0x0, 1\nmonitor = eDP-1, disable\n"
	if string(configData) != wantConfig {
		t.Fatalf("unexpected config content: got %q want %q", string(configData), wantConfig)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat config returned error: %v", err)
	}

	if info.Mode().Perm() != fileMode {
		t.Fatalf("unexpected config mode: got %v want %v", info.Mode().Perm(), fileMode)
	}
}

func TestApplyMonitorConfigCreatesFileWhenMissing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "hypr", "monitors.conf")
	lines := []string{"monitor = eDP-1, 2880x1800@120, 0x0, 1.5"}

	result, err := Apply(configPath, lines, time.Date(2026, time.April, 26, 13, 14, 15, 0, time.UTC))
	if err != nil {
		t.Fatalf("applyMonitorConfig returned error: %v", err)
	}

	if result.BackupPath != "" {
		t.Fatalf("expected no backup path, got %q", result.BackupPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(data) != "monitor = eDP-1, 2880x1800@120, 0x0, 1.5\n" {
		t.Fatalf("unexpected config content: %q", string(data))
	}
}
