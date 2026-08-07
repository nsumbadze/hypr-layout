package profile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExportAndImportProfileSetRoundTrip(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	profiles := map[string]string{
		"desk":   "# hypr-layout profile\nmonitor = DP-1, 2560x1440@165, 0x0, 1\n",
		"travel": "monitor = eDP-1, 2880x1800@120, 0x0, 1.5\n",
	}
	for name, content := range profiles {
		if err := os.WriteFile(filepath.Join(sourceDir, name+".conf"), []byte(content), fileMode); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	set, err := Export(sourceDir)
	if err != nil {
		t.Fatalf("exportProfileSet returned error: %v", err)
	}

	data, err := Encode(set)
	if err != nil {
		t.Fatalf("encodeProfileSet returned error: %v", err)
	}

	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("decodeProfileSet returned error: %v", err)
	}

	result, err := Import(targetDir, decoded, false)
	if err != nil {
		t.Fatalf("importProfileSet returned error: %v", err)
	}

	wantImported := []string{"desk", "travel"}
	if !reflect.DeepEqual(result.Imported, wantImported) {
		t.Fatalf("unexpected imported profiles: got %#v want %#v", result.Imported, wantImported)
	}

	for name, content := range profiles {
		data, err := os.ReadFile(filepath.Join(targetDir, name+".conf"))
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}

		if string(data) != content {
			t.Fatalf("profile %q content mismatch: got %q want %q", name, string(data), content)
		}
	}
}

func TestImportProfileSetSkipsExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()
	existing := "monitor = DP-1, 2560x1440@165, 0x0, 1\n"
	if err := os.WriteFile(filepath.Join(dir, "desk.conf"), []byte(existing), fileMode); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	set := Set{
		Version:  setVersion,
		Profiles: []Entry{{Name: "desk", Content: "monitor = eDP-1, disable\n"}},
	}

	result, err := Import(dir, set, false)
	if err != nil {
		t.Fatalf("importProfileSet returned error: %v", err)
	}

	if len(result.Imported) != 0 || !reflect.DeepEqual(result.Skipped, []string{"desk"}) {
		t.Fatalf("unexpected result: %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(dir, "desk.conf"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(data) != existing {
		t.Fatal("existing profile was overwritten without --force")
	}
}

func TestImportProfileSetOverwritesWithForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "desk.conf"), []byte("old\n"), fileMode); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	set := Set{
		Version:  setVersion,
		Profiles: []Entry{{Name: "desk", Content: "monitor = eDP-1, disable\n"}},
	}

	result, err := Import(dir, set, true)
	if err != nil {
		t.Fatalf("importProfileSet returned error: %v", err)
	}

	if !reflect.DeepEqual(result.Imported, []string{"desk"}) {
		t.Fatalf("unexpected result: %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(dir, "desk.conf"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if string(data) != "monitor = eDP-1, disable\n" {
		t.Fatalf("profile was not overwritten: %q", string(data))
	}
}

func TestImportProfileSetRejectsInvalidNames(t *testing.T) {
	set := Set{
		Version:  setVersion,
		Profiles: []Entry{{Name: "../evil", Content: "x\n"}},
	}

	if _, err := Import(t.TempDir(), set, false); err == nil {
		t.Fatal("expected error for invalid profile name")
	}
}

func TestDecodeProfileSetRejectsUnknownVersion(t *testing.T) {
	if _, err := Decode([]byte(`{"version": 99, "profiles": []}`)); err == nil {
		t.Fatal("expected error for unknown version")
	}
}
