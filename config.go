package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const monitorConfigMode = 0o644

type applyResult struct {
	ConfigPath        string
	BackupPath        string
	HadPreviousConfig bool
}

func promptApplyConfirmation(r io.Reader, w io.Writer) (bool, error) {
	return promptYesNo(r, w, "\nApply this layout to ~/.config/hypr/monitors.conf? (y/n) ")
}

func parseConfirmation(input string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("enter y or n")
	}
}

func defaultMonitorConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "hypr", "monitors.conf"), nil
}

func applyMonitorConfig(configPath string, lines []string, now time.Time) (applyResult, error) {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return applyResult{}, fmt.Errorf("create config directory: %w", err)
	}

	backupPath, hadPreviousConfig, err := backupExistingConfig(configPath, now)
	if err != nil {
		return applyResult{}, err
	}

	content := generateConfigContent(lines)
	if err := os.WriteFile(configPath, []byte(content), monitorConfigMode); err != nil {
		return applyResult{}, fmt.Errorf("write config file: %w", err)
	}

	if err := os.Chmod(configPath, monitorConfigMode); err != nil {
		return applyResult{}, fmt.Errorf("set config permissions: %w", err)
	}

	return applyResult{
		ConfigPath:        configPath,
		BackupPath:        backupPath,
		HadPreviousConfig: hadPreviousConfig,
	}, nil
}

func backupExistingConfig(configPath string, now time.Time) (string, bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("read existing config: %w", err)
	}

	backupPath := backupPathFor(configPath, now)
	if err := os.WriteFile(backupPath, data, monitorConfigMode); err != nil {
		return "", false, fmt.Errorf("create backup file: %w", err)
	}

	if err := os.Chmod(backupPath, monitorConfigMode); err != nil {
		return "", false, fmt.Errorf("set backup permissions: %w", err)
	}

	return backupPath, true, nil
}

func backupPathFor(configPath string, now time.Time) string {
	return fmt.Sprintf("%s.backup-%s", configPath, now.Format("20060102-150405"))
}

func generateConfigContent(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}
