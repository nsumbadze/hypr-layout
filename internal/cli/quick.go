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
	var rules []layout.Rule
	if layoutID == layout.Mirror {
		rules = layout.MirroredRules(monitors, activeConfigs, layout.MirrorSourceIndex(monitors, activeIndexes))
	} else {
		rules = layout.PositionedRules(monitors, activeConfigs, options.Direction)
	}
	return applyPreviewFlow("Quick: "+preset, rules, options.commandOptions)
}

// applyPreviewFlow previews the rules in the syntax of the file Hyprland
// reads, then writes them there and reloads.
func applyPreviewFlow(title string, rules []layout.Rule, options commandOptions) error {
	target, err := monconf.DetectTarget()
	if err != nil {
		return fmt.Errorf("could not determine monitor config path: %w", err)
	}
	lines := layout.Lines(rules, target.Format)
	fmt.Print(renderPreview(title, lines))
	if !options.AutoYes {
		apply, err := promptApplyConfirmation(os.Stdin, os.Stdout, target.Path)
		if err != nil {
			return fmt.Errorf("failed to read apply confirmation: %w", err)
		}
		if !apply {
			fmt.Print(renderStatusDim("No changes applied.") + "\n")
			return nil
		}
	}
	result, err := monconf.Apply(target, lines, time.Now())
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
