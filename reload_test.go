package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReloadHyprlandRunsHyprctlReload(t *testing.T) {
	var gotName string
	var gotArgs []string

	runner := func(_ context.Context, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := reloadHyprland(context.Background(), runner); err != nil {
		t.Fatalf("reloadHyprland returned error: %v", err)
	}

	if gotName != "hyprctl" {
		t.Fatalf("unexpected command name: %q", gotName)
	}

	wantArgs := []string{"reload"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", gotArgs, wantArgs)
	}
}

func TestReloadHyprlandWrapsRunnerError(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) error {
		return errors.New("boom")
	}

	err := reloadHyprland(context.Background(), runner)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "run hyprctl reload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRollbackMonitorConfigRestoresBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "monitors.conf")
	backupPath := filepath.Join(dir, "monitors.conf.backup-20260426-131415")

	if err := os.WriteFile(configPath, []byte("new config\n"), monitorConfigMode); err != nil {
		t.Fatalf("WriteFile config returned error: %v", err)
	}

	if err := os.WriteFile(backupPath, []byte("old config\n"), monitorConfigMode); err != nil {
		t.Fatalf("WriteFile backup returned error: %v", err)
	}

	err := rollbackMonitorConfig(applyResult{
		ConfigPath:        configPath,
		BackupPath:        backupPath,
		HadPreviousConfig: true,
	})
	if err != nil {
		t.Fatalf("rollbackMonitorConfig returned error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(data) != "old config\n" {
		t.Fatalf("unexpected restored config: %q", string(data))
	}
}

func TestRollbackMonitorConfigRemovesNewFileWhenNoBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "monitors.conf")

	if err := os.WriteFile(configPath, []byte("new config\n"), monitorConfigMode); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := rollbackMonitorConfig(applyResult{
		ConfigPath:        configPath,
		HadPreviousConfig: false,
	})
	if err != nil {
		t.Fatalf("rollbackMonitorConfig returned error: %v", err)
	}

	_, statErr := os.Stat(configPath)
	if !os.IsNotExist(statErr) {
		t.Fatalf("expected config file to be removed, got error %v", statErr)
	}
}
