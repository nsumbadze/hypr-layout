package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var monitorModePattern = regexp.MustCompile(`^(\d+)x(\d+)@([0-9]+(?:\.[0-9]+)?)Hz$`)

type monitorMode struct {
	Width       int
	Height      int
	RefreshRate float64
}

// modeStrategy names an automatic mode-selection rule, mirroring Hyprland's
// preferred/highres/highrr keywords. Strategies resolve to a concrete mode so
// positioning arithmetic always works with exact dimensions.
type modeStrategy string

const (
	modeStrategyCurrent   modeStrategy = "current"
	modeStrategyBest      modeStrategy = "best"
	modeStrategyPreferred modeStrategy = "preferred"
	modeStrategyHighres   modeStrategy = "highres"
	modeStrategyHighrr    modeStrategy = "highrr"
)

func parseModeStrategy(value string) (modeStrategy, error) {
	switch modeStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case modeStrategyCurrent:
		return modeStrategyCurrent, nil
	case modeStrategyBest:
		return modeStrategyBest, nil
	case modeStrategyPreferred:
		return modeStrategyPreferred, nil
	case modeStrategyHighres:
		return modeStrategyHighres, nil
	case modeStrategyHighrr:
		return modeStrategyHighrr, nil
	default:
		return "", fmt.Errorf("unknown mode strategy %q (use current, best, preferred, highres, or highrr)", value)
	}
}

func resolveModeStrategy(mon monitor, strategy modeStrategy) monitorMode {
	switch strategy {
	case modeStrategyCurrent:
		return currentMonitorMode(mon)
	case modeStrategyPreferred:
		return preferredMonitorMode(mon)
	case modeStrategyHighres:
		return highestResolutionMode(mon)
	default:
		// best and highrr both mean highest refresh rate.
		return bestMonitorMode(mon)
	}
}

// preferredMonitorMode returns the first valid entry of availableModes, which
// DRM connectors report as the display's preferred mode. Falls back to the
// current mode when no modes are available.
func preferredMonitorMode(mon monitor) monitorMode {
	for _, raw := range mon.AvailableModes {
		if mode, err := parseMonitorMode(raw); err == nil {
			return mode
		}
	}

	return currentMonitorMode(mon)
}

// highestResolutionMode returns the mode with the most pixels, breaking ties
// by refresh rate. Falls back to the current mode when no modes are available.
func highestResolutionMode(mon monitor) monitorMode {
	modes := availableMonitorModes(mon)
	if len(modes) == 0 {
		return currentMonitorMode(mon)
	}

	best := modes[0]
	for _, mode := range modes[1:] {
		if compareModesByResolution(mode, best) > 0 {
			best = mode
		}
	}

	return best
}

func compareModesByResolution(a, b monitorMode) int {
	aPixels := a.Width * a.Height
	bPixels := b.Width * b.Height
	switch {
	case aPixels > bPixels:
		return 1
	case aPixels < bPixels:
		return -1
	}

	switch {
	case a.RefreshRate > b.RefreshRate:
		return 1
	case a.RefreshRate < b.RefreshRate:
		return -1
	}

	return 0
}

func availableMonitorModes(mon monitor) []monitorMode {
	seen := make(map[string]struct{}, len(mon.AvailableModes))
	modes := make([]monitorMode, 0, len(mon.AvailableModes))

	for _, raw := range mon.AvailableModes {
		mode, err := parseMonitorMode(raw)
		if err != nil {
			continue
		}

		key := formatMonitorMode(mode)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		modes = append(modes, mode)
	}

	return modes
}

func currentMonitorMode(mon monitor) monitorMode {
	return monitorMode{
		Width:       mon.Width,
		Height:      mon.Height,
		RefreshRate: mon.RefreshRate,
	}
}

func parseMonitorMode(value string) (monitorMode, error) {
	matches := monitorModePattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return monitorMode{}, fmt.Errorf("invalid monitor mode: %q", value)
	}

	width, err := strconv.Atoi(matches[1])
	if err != nil {
		return monitorMode{}, fmt.Errorf("parse width: %w", err)
	}

	height, err := strconv.Atoi(matches[2])
	if err != nil {
		return monitorMode{}, fmt.Errorf("parse height: %w", err)
	}

	refreshRate, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return monitorMode{}, fmt.Errorf("parse refresh rate: %w", err)
	}

	return monitorMode{
		Width:       width,
		Height:      height,
		RefreshRate: refreshRate,
	}, nil
}

func formatMonitorMode(mode monitorMode) string {
	return fmt.Sprintf("%dx%d@%s", mode.Width, mode.Height, formatFloat(mode.RefreshRate))
}
