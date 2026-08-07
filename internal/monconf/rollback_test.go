package monconf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollbackMonitorConfigRestoresBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "monitors.conf")
	backupPath := filepath.Join(dir, "monitors.conf.backup-20260426-131415")

	if err := os.WriteFile(configPath, []byte("new config\n"), fileMode); err != nil {
		t.Fatalf("WriteFile config returned error: %v", err)
	}

	if err := os.WriteFile(backupPath, []byte("old config\n"), fileMode); err != nil {
		t.Fatalf("WriteFile backup returned error: %v", err)
	}

	err := Rollback(Result{
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

	if err := os.WriteFile(configPath, []byte("new config\n"), fileMode); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := Rollback(Result{
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
