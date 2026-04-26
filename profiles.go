package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func promptSaveProfileConfirmation(r io.Reader, w io.Writer) (bool, error) {
	return promptYesNo(r, w, "\nSave this layout as a profile? (y/n) ")
}

func promptProfileName(r io.Reader, w io.Writer) (string, error) {
	reader := bufio.NewReader(r)

	for {
		fmt.Fprint(w, "Profile name: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && strings.TrimSpace(input) != "" {
				name := strings.TrimSpace(input)
				if validateErr := validateProfileName(name); validateErr == nil {
					return name, nil
				}
			}

			return "", err
		}

		name := strings.TrimSpace(input)
		if err := validateProfileName(name); err != nil {
			fmt.Fprintf(w, "Invalid profile name: %v\n", err)
			continue
		}

		return name, nil
	}
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

func saveProfile(profilesDir, profileName string, lines []string) error {
	if err := validateProfileName(profileName); err != nil {
		return err
	}

	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return fmt.Errorf("create profiles directory: %w", err)
	}

	profilePath := profilePathFor(profilesDir, profileName)
	if err := os.WriteFile(profilePath, []byte(generateConfigContent(lines)), monitorConfigMode); err != nil {
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

func parseProfileContent(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "\n")
}

func profilePathFor(profilesDir, profileName string) string {
	return filepath.Join(profilesDir, profileName+".conf")
}
