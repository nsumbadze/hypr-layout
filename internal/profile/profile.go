// Package profile stores named layouts as readable config files with a small
// comment header, and bundles them for export and import.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// fileMode is the permission profiles are written with, matching the
// monitor config the tool generates elsewhere.
const fileMode = 0o644

// renderContent joins config lines into file content with a trailing newline.
func renderContent(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Metadata records how a profile was produced. It is stored as `#`
// comment lines at the top of the profile file, so profiles stay valid
// Hyprland config fragments.
type Metadata struct {
	SavedAt   time.Time
	Direction string
	// Monitors lists the active monitors in their final order together with
	// the mode chosen for each, e.g. "DP-1 2560x1440@165".
	Monitors []string
}

func formatHeader(meta Metadata) []string {
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

func parseMetadata(content string) Metadata {
	var meta Metadata

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

func DefaultDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "hypr-layout", "profiles"), nil
}

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if !namePattern.MatchString(name) {
		return fmt.Errorf("use only letters, numbers, dashes, and underscores")
	}

	return nil
}

func Save(profilesDir, profileName string, lines []string, meta Metadata) error {
	if err := ValidateName(profileName); err != nil {
		return err
	}

	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return fmt.Errorf("create profiles directory: %w", err)
	}

	content := renderContent(append(formatHeader(meta), lines...))
	profilePath := pathFor(profilesDir, profileName)
	if err := os.WriteFile(profilePath, []byte(content), fileMode); err != nil {
		return fmt.Errorf("write profile file: %w", err)
	}

	if err := os.Chmod(profilePath, fileMode); err != nil {
		return fmt.Errorf("set profile permissions: %w", err)
	}

	return nil
}

func List(profilesDir string) ([]string, error) {
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

func Load(profilesDir, profileName string) ([]string, error) {
	if err := ValidateName(profileName); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(pathFor(profilesDir, profileName))
	if err != nil {
		return nil, fmt.Errorf("read profile file: %w", err)
	}

	return parseContent(string(data)), nil
}

// parseProfileContent returns the config lines of a profile, skipping blank
// lines and `#` comments (including the metadata header).
func parseContent(content string) []string {
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

type Summary struct {
	Name string
	Meta Metadata
}

// listProfileSummaries returns every saved profile together with its parsed
// metadata; profiles without a metadata header get an empty metadata value.
func Summaries(profilesDir string) ([]Summary, error) {
	names, err := List(profilesDir)
	if err != nil {
		return nil, err
	}

	summaries := make([]Summary, 0, len(names))
	for _, name := range names {
		summary := Summary{Name: name}
		if data, err := os.ReadFile(pathFor(profilesDir, name)); err == nil {
			summary.Meta = parseMetadata(string(data))
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func pathFor(profilesDir, profileName string) string {
	return filepath.Join(profilesDir, profileName+".conf")
}
