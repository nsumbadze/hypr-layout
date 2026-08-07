package profile

import (
	"encoding/json"
	"fmt"
	"os"
)

// profileSetVersion is the format version written to exported bundles so the
// format can evolve without breaking older files.
const setVersion = 1

type Set struct {
	Version  int     `json:"version"`
	Profiles []Entry `json:"profiles"`
}

type Entry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// exportProfileSet bundles every saved profile (raw file content, including
// the metadata header) into a profileSet.
func Export(profilesDir string) (Set, error) {
	names, err := List(profilesDir)
	if err != nil {
		return Set{}, err
	}

	set := Set{Version: setVersion, Profiles: make([]Entry, 0, len(names))}
	for _, name := range names {
		data, err := os.ReadFile(pathFor(profilesDir, name))
		if err != nil {
			return Set{}, fmt.Errorf("read profile %q: %w", name, err)
		}

		set.Profiles = append(set.Profiles, Entry{Name: name, Content: string(data)})
	}

	return set, nil
}

func Encode(set Set) ([]byte, error) {
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode profile set: %w", err)
	}

	return append(data, '\n'), nil
}

func Decode(data []byte) (Set, error) {
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return Set{}, fmt.Errorf("decode profile set: %w", err)
	}

	if set.Version != setVersion {
		return Set{}, fmt.Errorf("unsupported profile set version: %d", set.Version)
	}

	return set, nil
}

type ImportResult struct {
	Imported []string
	Skipped  []string
}

// importProfileSet writes each bundled profile into profilesDir. Profiles with
// invalid names are rejected; existing profiles are skipped unless force is
// set.
func Import(profilesDir string, set Set, force bool) (ImportResult, error) {
	var result ImportResult

	for _, entry := range set.Profiles {
		if err := ValidateName(entry.Name); err != nil {
			return result, fmt.Errorf("invalid profile name %q: %w", entry.Name, err)
		}
	}

	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return result, fmt.Errorf("create profiles directory: %w", err)
	}

	for _, entry := range set.Profiles {
		profilePath := pathFor(profilesDir, entry.Name)
		if !force {
			if _, err := os.Stat(profilePath); err == nil {
				result.Skipped = append(result.Skipped, entry.Name)
				continue
			}
		}

		if err := os.WriteFile(profilePath, []byte(entry.Content), fileMode); err != nil {
			return result, fmt.Errorf("write profile %q: %w", entry.Name, err)
		}

		result.Imported = append(result.Imported, entry.Name)
	}

	return result, nil
}
