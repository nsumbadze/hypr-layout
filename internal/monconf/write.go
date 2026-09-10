// Package monconf writes the generated monitor config to the file Hyprland
// reads. Every write backs up whatever was there before, so a failed reload
// can be rolled back.
package monconf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nsumbadze/hypr-layout/internal/layout"
)

const fileMode = 0o644

// Target is the file the layout is written to and the syntax it expects.
type Target struct {
	Path   string
	Format layout.Format
}

type Result struct {
	ConfigPath        string
	BackupPath        string
	HadPreviousConfig bool
}

// DetectTarget picks the file Hyprland actually reads. A hyprland.lua means
// the Lua config is in use (Hyprland 0.55+, Omarchy Quattro), so the layout
// goes to monitors.lua; otherwise it goes to the classic monitors.conf.
func DetectTarget() (Target, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Target{}, fmt.Errorf("resolve home directory: %w", err)
	}

	return TargetIn(filepath.Join(homeDir, ".config", "hypr")), nil
}

func TargetIn(hyprDir string) Target {
	if _, err := os.Stat(filepath.Join(hyprDir, "hyprland.lua")); err == nil {
		return Target{Path: filepath.Join(hyprDir, "monitors.lua"), Format: layout.FormatLua}
	}

	return Target{Path: filepath.Join(hyprDir, "monitors.conf"), Format: layout.FormatConf}
}

// Apply writes lines, already rendered in the target's format, to the target
// file, backing up the previous file first.
func Apply(target Target, lines []string, now time.Time) (Result, error) {
	if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
		return Result{}, fmt.Errorf("create config directory: %w", err)
	}

	previous, backupPath, hadPreviousConfig, err := backupExisting(target.Path, now)
	if err != nil {
		return Result{}, err
	}

	content := generateContent(target.Format, previous, lines)
	if err := os.WriteFile(target.Path, []byte(content), fileMode); err != nil {
		return Result{}, fmt.Errorf("write config file: %w", err)
	}

	if err := os.Chmod(target.Path, fileMode); err != nil {
		return Result{}, fmt.Errorf("set config permissions: %w", err)
	}

	return Result{
		ConfigPath:        target.Path,
		BackupPath:        backupPath,
		HadPreviousConfig: hadPreviousConfig,
	}, nil
}

// backupExisting copies the current config aside and returns its content.
func backupExisting(configPath string, now time.Time) (string, string, bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", false, nil
		}

		return "", "", false, fmt.Errorf("read existing config: %w", err)
	}

	backupPath := backupPathFor(configPath, now)
	if err := os.WriteFile(backupPath, data, fileMode); err != nil {
		return "", "", false, fmt.Errorf("create backup file: %w", err)
	}

	if err := os.Chmod(backupPath, fileMode); err != nil {
		return "", "", false, fmt.Errorf("set backup permissions: %w", err)
	}

	return string(data), backupPath, true, nil
}

func backupPathFor(configPath string, now time.Time) string {
	return fmt.Sprintf("%s.backup-%s", configPath, now.Format("20060102-150405"))
}

func generateContent(format layout.Format, previous string, lines []string) string {
	if format == layout.FormatLua {
		return luaContent(previous, lines)
	}
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}

const luaHeader = "-- Written by hypr-layout. Rerun it to change the layout."

// luaContent puts the rules under a short header. Environment settings from
// the previous file are carried over: Omarchy keeps GDK_SCALE in
// monitors.lua, and dropping it would change how GTK apps render.
func luaContent(previous string, lines []string) string {
	out := []string{luaHeader}
	if env := preservedLuaLines(previous); len(env) > 0 {
		out = append(out, "")
		out = append(out, env...)
	}
	if len(lines) > 0 {
		out = append(out, "")
		out = append(out, lines...)
	}

	return strings.Join(out, "\n") + "\n"
}

// preservedLuaLines returns the hl.env calls of the previous file. A value
// written as tostring(name) is resolved against the file's `local name = ...`
// lines, because those locals are not carried over: Omarchy's default file
// sets GDK_SCALE from `local omarchy_gdk_scale = 2`.
func preservedLuaLines(previous string) []string {
	locals := make(map[string]string)
	var kept []string
	for _, line := range strings.Split(previous, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := luaLocalPattern.FindStringSubmatch(trimmed); m != nil {
			locals[m[1]] = strings.Trim(m[2], `"`)
			continue
		}
		if strings.HasPrefix(trimmed, "hl.env(") {
			kept = append(kept, luaToStringPattern.ReplaceAllStringFunc(trimmed, func(call string) string {
				name := luaToStringPattern.FindStringSubmatch(call)[1]
				if value, ok := locals[name]; ok {
					return `"` + value + `"`
				}
				return call
			}))
		}
	}

	return kept
}

var (
	luaLocalPattern    = regexp.MustCompile(`^local\s+(\w+)\s*=\s*("[^"]*"|[^\s-]+)`)
	luaToStringPattern = regexp.MustCompile(`tostring\((\w+)\)`)
)
