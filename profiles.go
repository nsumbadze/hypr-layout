package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// profileMetadata records how a profile was produced. It is stored as `#`
// comment lines at the top of the profile file, so profiles stay valid
// Hyprland config fragments.
type profileMetadata struct {
	SavedAt   time.Time
	Direction string
	// Monitors lists the active monitors in their final order together with
	// the mode chosen for each, e.g. "DP-1 2560x1440@165".
	Monitors []string
}

func formatProfileHeader(meta profileMetadata) []string {
	lines := []string{"# hypr-layout profile"}
	if !meta.SavedAt.IsZero() {
		lines = append(lines, "# saved: "+meta.SavedAt.UTC().Format(time.RFC3339))
	}
	if meta.Direction != "" {
		lines = append(lines, "# direction: "+meta.Direction)
	}
	if len(meta.Monitors) > 0 {
		lines = append(lines, "# monitors: "+strings.Join(meta.Monitors, ", "))
	}

	return lines
}

func parseProfileMetadata(content string) profileMetadata {
	var meta profileMetadata

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		body := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		switch {
		case strings.HasPrefix(body, "saved: "):
			if savedAt, err := time.Parse(time.RFC3339, strings.TrimPrefix(body, "saved: ")); err == nil {
				meta.SavedAt = savedAt
			}
		case strings.HasPrefix(body, "direction: "):
			meta.Direction = strings.TrimPrefix(body, "direction: ")
		case strings.HasPrefix(body, "monitors: "):
			meta.Monitors = strings.Split(strings.TrimPrefix(body, "monitors: "), ", ")
		}
	}

	return meta
}

func defaultProfilesDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "hypr-layout", "profiles"), nil
}

func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("use only letters, numbers, dashes, and underscores")
	}

	return nil
}

func saveProfile(profilesDir, profileName string, lines []string, meta profileMetadata) error {
	if err := validateProfileName(profileName); err != nil {
		return err
	}

	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return fmt.Errorf("create profiles directory: %w", err)
	}

	content := generateConfigContent(append(formatProfileHeader(meta), lines...))
	profilePath := profilePathFor(profilesDir, profileName)
	if err := os.WriteFile(profilePath, []byte(content), monitorConfigMode); err != nil {
		return fmt.Errorf("write profile file: %w", err)
	}

	if err := os.Chmod(profilePath, monitorConfigMode); err != nil {
		return fmt.Errorf("set profile permissions: %w", err)
	}

	return nil
}

func listProfiles(profilesDir string) ([]string, error) {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".conf" {
			continue
		}

		profiles = append(profiles, strings.TrimSuffix(name, ".conf"))
	}

	sort.Strings(profiles)
	return profiles, nil
}

func loadProfile(profilesDir, profileName string) ([]string, error) {
	if err := validateProfileName(profileName); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(profilePathFor(profilesDir, profileName))
	if err != nil {
		return nil, fmt.Errorf("read profile file: %w", err)
	}

	return parseProfileContent(string(data)), nil
}

// parseProfileContent returns the config lines of a profile, skipping blank
// lines and `#` comments (including the metadata header).
func parseProfileContent(content string) []string {
	var lines []string

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lines = append(lines, trimmed)
	}

	return lines
}

type profileSummary struct {
	Name string
	Meta profileMetadata
}

// listProfileSummaries returns every saved profile together with its parsed
// metadata; profiles without a metadata header get an empty metadata value.
func listProfileSummaries(profilesDir string) ([]profileSummary, error) {
	names, err := listProfiles(profilesDir)
	if err != nil {
		return nil, err
	}

	summaries := make([]profileSummary, 0, len(names))
	for _, name := range names {
		summary := profileSummary{Name: name}
		if data, err := os.ReadFile(profilePathFor(profilesDir, name)); err == nil {
			summary.Meta = parseProfileMetadata(string(data))
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func profilePathFor(profilesDir, profileName string) string {
	return filepath.Join(profilesDir, profileName+".conf")
}
