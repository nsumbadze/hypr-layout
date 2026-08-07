package monconf

import (
	"fmt"
	"os"
)

func Rollback(result Result) error {
	if result.HadPreviousConfig {
		return restoreBackup(result.ConfigPath, result.BackupPath)
	}

	if err := os.Remove(result.ConfigPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove newly created config: %w", err)
	}

	return nil
}

func restoreBackup(configPath, backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	if err := os.WriteFile(configPath, data, fileMode); err != nil {
		return fmt.Errorf("restore config from backup: %w", err)
	}

	if err := os.Chmod(configPath, fileMode); err != nil {
		return fmt.Errorf("set restored config permissions: %w", err)
	}

	return nil
}
