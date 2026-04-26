package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch {
	case len(args) == 0:
		return runInteractive()
	case len(args) == 1 && args[0] == "list":
		return runListProfiles()
	case len(args) >= 2 && args[0] == "apply":
		return runApplyProfile(args[1], args[2:])
	case len(args) >= 2 && args[0] == "quick":
		return runQuickPreset(args[1], args[2:])
	default:
		return fmt.Errorf("usage: hypr-layout [list | apply <profile-name> [--yes] [--no-reload] | quick <preset> [--yes] [--no-reload]]")
	}
}

func runInteractive() error {
	monitors, err := detectMonitors()
	if err != nil {
		return err
	}

	fmt.Print(renderMonitors(monitors))

	selection, err := promptLayoutSelection(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read selection: %w", err)
	}

	if selection.ID == 5 {
		return nil
	}

	activeIndexes, err := activeIndexesForLayout(monitors, selection.ID)
	if err != nil {
		return fmt.Errorf("could not determine active monitors: %w", err)
	}

	selectedModes, err := promptMonitorModes(os.Stdin, os.Stdout, monitors, activeIndexes)
	if err != nil {
		return fmt.Errorf("failed to read monitor mode selection: %w", err)
	}

	direction, err := promptLayoutDirection(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read layout direction: %w", err)
	}

	activeConfigs := buildActiveMonitorConfigs(monitors, activeIndexes, selectedModes)
	activeConfigs, err = promptMonitorOrder(os.Stdin, os.Stdout, monitors, activeConfigs)
	if err != nil {
		return fmt.Errorf("failed to read monitor order: %w", err)
	}

	lines := renderPositionedConfigLines(monitors, activeConfigs, direction)

	fmt.Print(renderPreview(selection.Name, lines))

	if err := maybeSaveProfile(lines); err != nil {
		return err
	}

	return applyPreviewFlow(lines, commandOptions{})
}

func runListProfiles() error {
	profilesDir, err := defaultProfilesDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}

	profiles, err := listProfiles(profilesDir)
	if err != nil {
		return fmt.Errorf("could not list profiles: %w", err)
	}

	fmt.Print(renderProfileList(profiles))
	return nil
}

func runApplyProfile(profileName string, args []string) error {
	options, err := parseCommandOptions(args)
	if err != nil {
		return err
	}

	profilesDir, err := defaultProfilesDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}

	lines, err := loadProfile(profilesDir, profileName)
	if err != nil {
		return fmt.Errorf("could not load profile %q: %w", profileName, err)
	}

	fmt.Print(renderPreview("Profile "+profileName, lines))
	return applyPreviewFlow(lines, options)
}

func runQuickPreset(preset string, args []string) error {
	options, err := parseCommandOptions(args)
	if err != nil {
		return err
	}

	layoutID, err := quickPresetLayoutID(preset)
	if err != nil {
		return err
	}

	monitors, err := detectMonitors()
	if err != nil {
		return err
	}

	if len(monitors) == 0 {
		return fmt.Errorf("no monitors detected")
	}

	activeIndexes, err := activeIndexesForLayout(monitors, layoutID)
	if err != nil {
		return fmt.Errorf("could not determine active monitors: %w", err)
	}

	activeConfigs := buildActiveMonitorConfigs(monitors, activeIndexes, autoSelectMonitorModes(monitors, activeIndexes))
	lines := renderPositionedConfigLines(monitors, activeConfigs, horizontalDirection)

	fmt.Print(renderPreview("Quick "+preset, lines))
	return applyPreviewFlow(lines, options)
}

func detectMonitors() ([]monitor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := readMonitorsJSON(ctx)
	if err != nil {
		return nil, friendlyMonitorError(err)
	}

	monitors, err := parseMonitors(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Hyprland monitor data: %w", err)
	}

	return monitors, nil
}

func maybeSaveProfile(lines []string) error {
	save, err := promptSaveProfileConfirmation(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read save profile confirmation: %w", err)
	}

	if !save {
		return nil
	}

	profilesDir, err := defaultProfilesDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}

	profileName, err := promptProfileName(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read profile name: %w", err)
	}

	if err := saveProfile(profilesDir, profileName, lines); err != nil {
		return fmt.Errorf("could not save profile: %w", err)
	}

	fmt.Printf("Profile saved: %s\n", profileName)
	return nil
}

func applyPreviewFlow(lines []string, options commandOptions) error {
	if !options.AutoYes {
		apply, err := promptApplyConfirmation(os.Stdin, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to read apply confirmation: %w", err)
		}

		if !apply {
			fmt.Println("No changes applied.")
			return nil
		}
	}

	configPath, err := defaultMonitorConfigPath()
	if err != nil {
		return fmt.Errorf("could not determine monitor config path: %w", err)
	}

	result, err := applyMonitorConfig(configPath, lines, time.Now())
	if err != nil {
		return fmt.Errorf("failed to apply layout: %w", err)
	}

	if result.BackupPath != "" {
		fmt.Printf("Backup created: %s\n", result.BackupPath)
	}

	fmt.Printf("Layout written to %s\n", result.ConfigPath)
	return reloadFlow(result, options)
}

func reloadFlow(result applyResult, options commandOptions) error {
	if options.NoReload {
		fmt.Println("Config written. Run `hyprctl reload` manually to apply.")
		return nil
	}

	reloadNow, err := promptReloadConfirmation(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read reload confirmation: %w", err)
	}

	if !reloadNow {
		fmt.Println("Config written. Run `hyprctl reload` manually to apply.")
		return nil
	}

	reloadCtx, reloadCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer reloadCancel()

	if err := reloadHyprland(reloadCtx, systemCommandRunner); err != nil {
		rollbackErr := rollbackMonitorConfig(result)
		if rollbackErr != nil {
			return fmt.Errorf("hyprland reload failed: %v; rollback failed: %w", err, rollbackErr)
		}

		if result.BackupPath != "" {
			return fmt.Errorf("hyprland reload failed: %v; rollback complete, restored backup from %s", err, result.BackupPath)
		}

		return fmt.Errorf("hyprland reload failed: %v; rollback complete, removed the newly created monitors.conf", err)
	}

	fmt.Println("Hyprland reloaded successfully.")
	return nil
}
