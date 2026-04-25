package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := readMonitorsJSON(ctx)
	if err != nil {
		exitWithFriendlyMonitorError(err)
	}

	monitors, err := parseMonitors(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse Hyprland monitor data: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(renderMonitors(monitors))

	selection, err := promptLayoutSelection(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read selection: %v\n", err)
		os.Exit(1)
	}

	if selection.ID == 5 {
		return
	}

	lines, err := buildPreviewConfig(monitors, selection.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not build preview config: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(renderPreview(selection.Name, lines))

	apply, err := promptApplyConfirmation(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read apply confirmation: %v\n", err)
		os.Exit(1)
	}

	if !apply {
		fmt.Println("No changes applied.")
		return
	}

	configPath, err := defaultMonitorConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not determine monitor config path: %v\n", err)
		os.Exit(1)
	}

	result, err := applyMonitorConfig(configPath, lines, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to apply layout: %v\n", err)
		os.Exit(1)
	}

	if result.BackupPath != "" {
		fmt.Printf("Backup created: %s\n", result.BackupPath)
	}

	fmt.Printf("Layout written to %s\n", result.ConfigPath)
}
