package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/monconf"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

func runQuickPreset(preset string, args []string) error {
	options, err := parseQuickOptions(args)
	if err != nil {
		return err
	}
	layoutID, err := quickPresetLayoutID(preset)
	if err != nil {
		return err
	}
	monitors, err := hypr.Detect()
	if err != nil {
		return err
	}
	if len(monitors) == 0 {
		return fmt.Errorf("no monitors detected")
	}
	activeIndexes, err := layout.ActiveIndexes(monitors, layoutID)
	if err != nil {
		return fmt.Errorf("could not determine active monitors: %w", err)
	}
	activeConfigs := layout.BuildConfigs(monitors, activeIndexes, layout.AutoSelectModes(monitors, activeIndexes, options.Mode))
	applyDisplayOverrides(activeConfigs, options.Transform, options.VRR)
	if options.Order != "" {
		activeConfigs, err = layout.Reorder(options.Order, activeConfigs)
		if err != nil {
			return fmt.Errorf("invalid --order: %w", err)
		}
	}
	var lines []string
	if layoutID == layout.Mirror {
		lines = layout.MirroredLines(monitors, activeConfigs, layout.MirrorSourceIndex(monitors, activeIndexes))
	} else {
		lines = layout.PositionedLines(monitors, activeConfigs, options.Direction)
	}
	fmt.Print(renderPreview("Quick: "+preset, lines))
	return applyPreviewFlow(lines, options.commandOptions)
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
	configPath, err := monconf.DefaultPath()
	if err != nil {
		return fmt.Errorf("could not determine monitor config path: %w", err)
	}
	result, err := monconf.Apply(configPath, lines, time.Now())
	if err != nil {
		return fmt.Errorf("failed to apply layout: %w", err)
	}
	if result.BackupPath != "" {
		fmt.Print(renderInlineStatus(ui.Dimmed.Render(""), "Backup: "+result.BackupPath))
	}
	fmt.Print(renderInlineStatus(ui.Success.Render("✓"), "Layout written to "+result.ConfigPath))
	return reloadFlow(result, options)
}

func reloadFlow(result monconf.Result, options commandOptions) error {
	if options.NoReload {
		fmt.Print(renderInlineStatus(ui.Dimmed.Render(""), "Run "+ui.Accent.Render("hyprctl reload")+" manually to apply."))
		return nil
	}
	reloadNow, err := promptReloadConfirmation(os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to read reload confirmation: %w", err)
	}
	if !reloadNow {
		fmt.Print(renderInlineStatus(ui.Dimmed.Render(""), "Run "+ui.Accent.Render("hyprctl reload")+" manually to apply."))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := hypr.Reload(ctx, hypr.SystemRunner); err != nil {
		rbErr := monconf.Rollback(result)
		if rbErr != nil {
			return fmt.Errorf("hyprland reload failed: %v; rollback failed: %w", err, rbErr)
		}
		if result.BackupPath != "" {
			return fmt.Errorf("hyprland reload failed: %v; rollback complete, restored backup from %s", err, result.BackupPath)
		}
		return fmt.Errorf("hyprland reload failed: %v; rollback complete, removed newly created config", err)
	}
	fmt.Print(renderInlineStatus(ui.Success.Render("✓"), "Hyprland reloaded."))
	return nil
}
