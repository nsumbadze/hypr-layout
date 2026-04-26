package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type commandRunner func(context.Context, string, ...string) error

func promptReloadConfirmation(r io.Reader, w io.Writer) (bool, error) {
	return promptYesNo(r, w, "\nReload Hyprland now? (y/n) ")
}

func reloadHyprland(ctx context.Context, run commandRunner) error {
	if err := run(ctx, "hyprctl", "reload"); err != nil {
		return fmt.Errorf("run hyprctl reload: %w", err)
	}

	return nil
}

func rollbackMonitorConfig(result applyResult) error {
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

	if err := os.WriteFile(configPath, data, monitorConfigMode); err != nil {
		return fmt.Errorf("restore config from backup: %w", err)
	}

	if err := os.Chmod(configPath, monitorConfigMode); err != nil {
		return fmt.Errorf("set restored config permissions: %w", err)
	}

	return nil
}

func systemCommandRunner(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

func promptYesNo(r io.Reader, w io.Writer, prompt string) (bool, error) {
	reader := bufio.NewReader(r)

	for {
		fmt.Fprint(w, prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && input != "" {
				return parseConfirmation(input)
			}

			return false, err
		}

		value, parseErr := parseConfirmation(input)
		if parseErr != nil {
			fmt.Fprintln(w, "Invalid selection: enter y or n.")
			continue
		}

		return value, nil
	}
}
