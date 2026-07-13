package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// profileSetVersion is the format version written to exported bundles so the
// format can evolve without breaking older files.
const profileSetVersion = 1

type profileSet struct {
	Version  int            `json:"version"`
	Profiles []profileEntry `json:"profiles"`
}

type profileEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// exportProfileSet bundles every saved profile (raw file content, including
// the metadata header) into a profileSet.
func exportProfileSet(profilesDir string) (profileSet, error) {
	names, err := listProfiles(profilesDir)
	if err != nil {
		return profileSet{}, err
	}

	set := profileSet{Version: profileSetVersion, Profiles: make([]profileEntry, 0, len(names))}
	for _, name := range names {
		data, err := os.ReadFile(profilePathFor(profilesDir, name))
		if err != nil {
			return profileSet{}, fmt.Errorf("read profile %q: %w", name, err)
		}

		set.Profiles = append(set.Profiles, profileEntry{Name: name, Content: string(data)})
	}

	return set, nil
}

func encodeProfileSet(set profileSet) ([]byte, error) {
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode profile set: %w", err)
	}

	return append(data, '\n'), nil
}

func decodeProfileSet(data []byte) (profileSet, error) {
	var set profileSet
	if err := json.Unmarshal(data, &set); err != nil {
		return profileSet{}, fmt.Errorf("decode profile set: %w", err)
	}

	if set.Version != profileSetVersion {
		return profileSet{}, fmt.Errorf("unsupported profile set version: %d", set.Version)
	}

	return set, nil
}

type importResult struct {
	Imported []string
	Skipped  []string
}

// importProfileSet writes each bundled profile into profilesDir. Profiles with
// invalid names are rejected; existing profiles are skipped unless force is
// set.
func importProfileSet(profilesDir string, set profileSet, force bool) (importResult, error) {
	var result importResult

	for _, entry := range set.Profiles {
		if err := validateProfileName(entry.Name); err != nil {
			return result, fmt.Errorf("invalid profile name %q: %w", entry.Name, err)
		}
	}

	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return result, fmt.Errorf("create profiles directory: %w", err)
	}

	for _, entry := range set.Profiles {
		profilePath := profilePathFor(profilesDir, entry.Name)
		if !force {
			if _, err := os.Stat(profilePath); err == nil {
				result.Skipped = append(result.Skipped, entry.Name)
				continue
			}
		}

		if err := os.WriteFile(profilePath, []byte(entry.Content), monitorConfigMode); err != nil {
			return result, fmt.Errorf("write profile %q: %w", entry.Name, err)
		}

		result.Imported = append(result.Imported, entry.Name)
	}

	return result, nil
}
