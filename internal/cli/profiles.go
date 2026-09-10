package cli

import (
	"fmt"
	"os"

	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/profile"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

// exportFileMode is the permission the profile bundle is written with.
const exportFileMode = 0o644

func runListProfiles() error {
	profilesDir, err := profile.DefaultDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}
	profiles, err := profile.Summaries(profilesDir)
	if err != nil {
		return fmt.Errorf("could not list profiles: %w", err)
	}
	fmt.Print(renderProfileList(profiles))
	return nil
}

// runExportProfiles writes all saved profiles as a JSON bundle to the given
// file, or to stdout when no file is given (kept unstyled so it can be piped).
func runExportProfiles(exportPath string) error {
	profilesDir, err := profile.DefaultDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}
	set, err := profile.Export(profilesDir)
	if err != nil {
		return fmt.Errorf("could not export profiles: %w", err)
	}
	data, err := profile.Encode(set)
	if err != nil {
		return err
	}
	if exportPath == "" {
		fmt.Print(string(data))
		return nil
	}
	if err := os.WriteFile(exportPath, data, exportFileMode); err != nil {
		return fmt.Errorf("could not write export file: %w", err)
	}
	fmt.Print(renderInlineStatus(ui.Success.Render("✓"), fmt.Sprintf("Exported %d profile(s) to %s", len(set.Profiles), exportPath)))
	return nil
}

func runImportProfiles(importPath string, args []string) error {
	force := false
	for _, arg := range args {
		if arg != "--force" {
			return fmt.Errorf("unknown flag: %s", arg)
		}
		force = true
	}
	profilesDir, err := profile.DefaultDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}
	data, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("could not read import file: %w", err)
	}
	set, err := profile.Decode(data)
	if err != nil {
		return err
	}
	result, err := profile.Import(profilesDir, set, force)
	if err != nil {
		return fmt.Errorf("could not import profiles: %w", err)
	}
	for _, name := range result.Imported {
		fmt.Print(renderInlineStatus(ui.Success.Render("✓"), "Imported "+name))
	}
	for _, name := range result.Skipped {
		fmt.Print(renderInlineStatus(ui.Dimmed.Render("○"), "Skipped "+name+" (already exists, use --force to overwrite)"))
	}
	if len(result.Imported) == 0 && len(result.Skipped) == 0 {
		fmt.Print(renderStatusDim("No profiles in bundle.") + "\n")
	}
	return nil
}

func runApplyProfile(profileName string, args []string) error {
	options, err := parseCommandOptions(args)
	if err != nil {
		return err
	}
	profilesDir, err := profile.DefaultDir()
	if err != nil {
		return fmt.Errorf("could not determine profiles directory: %w", err)
	}
	lines, err := profile.Load(profilesDir, profileName)
	if err != nil {
		return fmt.Errorf("could not load profile %q: %w", profileName, err)
	}
	rules, err := layout.ParseConfLines(lines)
	if err != nil {
		return fmt.Errorf("could not read profile %q: %w", profileName, err)
	}
	return applyPreviewFlow("Profile: "+profileName, rules, options)
}
