package layout

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

var modePattern = regexp.MustCompile(`^(\d+)x(\d+)@([0-9]+(?:\.[0-9]+)?)Hz$`)

type Mode struct {
	Width       int
	Height      int
	RefreshRate float64
}

// Strategy names an automatic mode-selection rule, mirroring Hyprland's
// preferred/highres/highrr keywords. Strategies resolve to a concrete mode so
// positioning arithmetic always works with exact dimensions.
type Strategy string

const (
	StrategyCurrent   Strategy = "current"
	StrategyBest      Strategy = "best"
	StrategyPreferred Strategy = "preferred"
	StrategyHighres   Strategy = "highres"
	StrategyHighrr    Strategy = "highrr"
)

func ParseStrategy(value string) (Strategy, error) {
	switch Strategy(strings.ToLower(strings.TrimSpace(value))) {
	case StrategyCurrent:
		return StrategyCurrent, nil
	case StrategyBest:
		return StrategyBest, nil
	case StrategyPreferred:
		return StrategyPreferred, nil
	case StrategyHighres:
		return StrategyHighres, nil
	case StrategyHighrr:
		return StrategyHighrr, nil
	default:
		return "", fmt.Errorf("unknown mode strategy %q (use current, best, preferred, highres, or highrr)", value)
	}
}

func ResolveStrategy(mon hypr.Monitor, strategy Strategy) Mode {
	switch strategy {
	case StrategyCurrent:
		return CurrentMode(mon)
	case StrategyPreferred:
		return preferredMode(mon)
	case StrategyHighres:
		return highestResolutionMode(mon)
	default:
		// best and highrr both mean highest refresh rate.
		return BestMode(mon)
	}
}

// preferredMonitorMode returns the first valid entry of availableModes, which
// DRM connectors report as the display's preferred mode. Falls back to the
// current mode when no modes are available.
func preferredMode(mon hypr.Monitor) Mode {
	for _, raw := range mon.AvailableModes {
		if mode, err := ParseMode(raw); err == nil {
			return mode
		}
	}

	return CurrentMode(mon)
}

// highestResolutionMode returns the mode with the most pixels, breaking ties
// by refresh rate. Falls back to the current mode when no modes are available.
func highestResolutionMode(mon hypr.Monitor) Mode {
	modes := AvailableModes(mon)
	if len(modes) == 0 {
		return CurrentMode(mon)
	}

	best := modes[0]
	for _, mode := range modes[1:] {
		if compareByResolution(mode, best) > 0 {
			best = mode
		}
	}

	return best
}

func compareByResolution(a, b Mode) int {
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

func AvailableModes(mon hypr.Monitor) []Mode {
	seen := make(map[string]struct{}, len(mon.AvailableModes))
	modes := make([]Mode, 0, len(mon.AvailableModes))

	for _, raw := range mon.AvailableModes {
		mode, err := ParseMode(raw)
		if err != nil {
			continue
		}

		key := FormatMode(mode)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		modes = append(modes, mode)
	}

	return modes
}

func CurrentMode(mon hypr.Monitor) Mode {
	return Mode{
		Width:       mon.Width,
		Height:      mon.Height,
		RefreshRate: mon.RefreshRate,
	}
}

func ParseMode(value string) (Mode, error) {
	matches := modePattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return Mode{}, fmt.Errorf("invalid monitor mode: %q", value)
	}

	width, err := strconv.Atoi(matches[1])
	if err != nil {
		return Mode{}, fmt.Errorf("parse width: %w", err)
	}

	height, err := strconv.Atoi(matches[2])
	if err != nil {
		return Mode{}, fmt.Errorf("parse height: %w", err)
	}

	refreshRate, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return Mode{}, fmt.Errorf("parse refresh rate: %w", err)
	}

	return Mode{
		Width:       width,
		Height:      height,
		RefreshRate: refreshRate,
	}, nil
}

func FormatMode(mode Mode) string {
	return fmt.Sprintf("%dx%d@%s", mode.Width, mode.Height, ui.FormatFloat(mode.RefreshRate))
}

func AutoSelectModes(monitors []hypr.Monitor, activeIndexes []int, strategy Strategy) map[int]Mode {
	selected := make(map[int]Mode, len(activeIndexes))

	for _, idx := range activeIndexes {
		selected[idx] = ResolveStrategy(monitors[idx], strategy)
	}

	return selected
}

func BestMode(mon hypr.Monitor) Mode {
	current := CurrentMode(mon)
	best := current
	modes := AvailableModes(mon)

	if len(modes) == 0 {
		return current
	}

	best = modes[0]
	for _, mode := range modes[1:] {
		if compareModes(mode, best) > 0 {
			best = mode
		}
	}

	return best
}

func compareModes(a, b Mode) int {
	switch {
	case a.RefreshRate > b.RefreshRate:
		return 1
	case a.RefreshRate < b.RefreshRate:
		return -1
	}

	aPixels := a.Width * a.Height
	bPixels := b.Width * b.Height
	switch {
	case aPixels > bPixels:
		return 1
	case aPixels < bPixels:
		return -1
	}

	return 0
}
