package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, renderInlineError(err.Error()))
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
		return fmt.Errorf("usage: hypr-layout [list | apply <profile-name> [--yes] [--no-reload] | quick <preset> [--yes] [--no-reload] [--direction <dir>] [--order <indices>]]")
	}
}

// runInteractive runs the full Bubble Tea wizard for the interactive flow.
// All state management (detect → layout → mode → direction → order →
// preview → save profile → apply → reload) happens inside the TUI model.
func runInteractive() error {
	p := tea.NewProgram(newTUIModel(), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	final, ok := finalModel.(tuiModel)
	if !ok {
		return fmt.Errorf("unexpected model type after TUI exit")
	}
	return final.err
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
	fmt.Print(renderPreview("Profile: "+profileName, lines))
	return applyPreviewFlow(lines, options)
}

func runQuickPreset(preset string, args []string) error {
	options, err := parseQuickOptions(args)
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
	activeConfigs := buildActiveMonitorConfigs(monitors, activeIndexes, autoSelectMonitorModes(monitors, activeIndexes, options.Mode))
	if options.Order != "" {
		activeConfigs, err = reorderActiveConfigs(options.Order, activeConfigs)
		if err != nil {
			return fmt.Errorf("invalid --order: %w", err)
		}
	}
	lines := renderPositionedConfigLines(monitors, activeConfigs, options.Direction)
	fmt.Print(renderPreview("Quick: "+preset, lines))
	return applyPreviewFlow(lines, options.commandOptions)
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

func applyPreviewFlow(lines []string, options commandOptions) error {
	if !options.AutoYes {
		apply, err := promptApplyConfirmation(os.Stdin, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to read apply confirmation: %w", err)
		}
		if !apply {
			fmt.Print(renderStatusDim("No changes applied.") + "\n")
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
		fmt.Print(renderInlineStatus(styleDimmed.Render(""), "Backup: "+result.BackupPath))
	}
	fmt.Print(renderInlineStatus(styleSuccess.Render("✓"), "Layout written to "+result.ConfigPath))
	return reloadFlow(result, options)
}

func reloadFlow(result applyResult, options commandOptions) error {
	if options.NoReload {
		fmt.Print(renderInlineStatus(styleDimmed.Render(""), "Run "+styleAccent.Render("hyprctl reload")+" manually to apply."))
		return nil
	}
	reloadNow, err := promptReloadConfirmation(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read reload confirmation: %w", err)
	}
	if !reloadNow {
		fmt.Print(renderInlineStatus(styleDimmed.Render(""), "Run "+styleAccent.Render("hyprctl reload")+" manually to apply."))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := reloadHyprland(ctx, systemCommandRunner); err != nil {
		rbErr := rollbackMonitorConfig(result)
		if rbErr != nil {
			return fmt.Errorf("hyprland reload failed: %v; rollback failed: %w", err, rbErr)
		}
		if result.BackupPath != "" {
			return fmt.Errorf("hyprland reload failed: %v; rollback complete, restored backup from %s", err, result.BackupPath)
		}
		return fmt.Errorf("hyprland reload failed: %v; rollback complete, removed newly created config", err)
	}
	fmt.Print(renderInlineStatus(styleSuccess.Render("✓"), "Hyprland reloaded."))
	return nil
}
