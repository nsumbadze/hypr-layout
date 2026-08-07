// Package monconf writes the generated monitor config to disk. Every write
// backs up whatever was there before, so a failed reload can be rolled back.
package monconf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fileMode = 0o644

type Result struct {
	ConfigPath        string
	BackupPath        string
	HadPreviousConfig bool
}

func DefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "hypr", "monitors.conf"), nil
}

func Apply(configPath string, lines []string, now time.Time) (Result, error) {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create config directory: %w", err)
	}

	backupPath, hadPreviousConfig, err := backupExisting(configPath, now)
	if err != nil {
		return Result{}, err
	}

	content := generateContent(lines)
	if err := os.WriteFile(configPath, []byte(content), fileMode); err != nil {
		return Result{}, fmt.Errorf("write config file: %w", err)
	}

	if err := os.Chmod(configPath, fileMode); err != nil {
		return Result{}, fmt.Errorf("set config permissions: %w", err)
	}

	return Result{
		ConfigPath:        configPath,
		BackupPath:        backupPath,
		HadPreviousConfig: hadPreviousConfig,
	}, nil
}

func backupExisting(configPath string, now time.Time) (string, bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("read existing config: %w", err)
	}

	backupPath := backupPathFor(configPath, now)
	if err := os.WriteFile(backupPath, data, fileMode); err != nil {
		return "", false, fmt.Errorf("create backup file: %w", err)
	}

	if err := os.Chmod(backupPath, fileMode); err != nil {
		return "", false, fmt.Errorf("set backup permissions: %w", err)
	}

	return backupPath, true, nil
}

func backupPathFor(configPath string, now time.Time) string {
	return fmt.Sprintf("%s.backup-%s", configPath, now.Format("20060102-150405"))
}

func generateContent(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}
