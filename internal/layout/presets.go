package layout

import (
	"fmt"
	"strings"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
)

type Option struct {
	ID   int
	Name string
}

const (
	LaptopOnly       = 1
	ExternalOnly     = 2
	DualHorizontal   = 3
	TripleHorizontal = 4
	Mirror           = 5
	Quit             = 6
)

func Options() []Option {
	return []Option{
		{ID: LaptopOnly, Name: "Laptop only"},
		{ID: ExternalOnly, Name: "External only"},
		{ID: DualHorizontal, Name: "Dual horizontal"},
		{ID: TripleHorizontal, Name: "Triple horizontal"},
		{ID: Mirror, Name: "Mirror all displays"},
		{ID: Quit, Name: "Quit"},
	}
}

func ActiveIndexes(monitors []hypr.Monitor, layoutID int) ([]int, error) {
	switch layoutID {
	case LaptopOnly:
		return buildLaptopOnly(monitors)
	case ExternalOnly:
		return buildExternalOnly(monitors)
	case DualHorizontal:
		return buildHorizontal(monitors, 2)
	case TripleHorizontal:
		return buildHorizontal(monitors, 3)
	case Mirror:
		return buildMirror(monitors)
	default:
		return nil, fmt.Errorf("unknown layout: %d", layoutID)
	}
}

// buildMirrorLayout activates every connected monitor; one acts as the mirror
// source and the rest mirror it.
func buildMirror(monitors []hypr.Monitor) ([]int, error) {
	if len(monitors) < 2 {
		return nil, fmt.Errorf("need at least 2 monitors to mirror")
	}

	indexes := make([]int, len(monitors))
	for idx := range monitors {
		indexes[idx] = idx
	}

	return indexes, nil
}

// mirrorSourceIndex picks the monitor the others mirror: the focused monitor
// when it is active, otherwise the first active monitor.
func MirrorSourceIndex(monitors []hypr.Monitor, activeIndexes []int) int {
	for _, idx := range activeIndexes {
		if monitors[idx].Focused {
			return idx
		}
	}

	return activeIndexes[0]
}

func buildLaptopOnly(monitors []hypr.Monitor) ([]int, error) {
	laptopIndex := findLaptopIndex(monitors)
	if laptopIndex < 0 {
		return nil, fmt.Errorf("no laptop display detected")
	}

	return []int{laptopIndex}, nil
}

func buildExternalOnly(monitors []hypr.Monitor) ([]int, error) {
	activeIndexes := pickPreferredIndexes(monitors, 1, true)
	if len(activeIndexes) == 0 {
		return nil, fmt.Errorf("no external display detected")
	}

	return activeIndexes, nil
}

func buildHorizontal(monitors []hypr.Monitor, count int) ([]int, error) {
	activeIndexes := pickPreferredIndexes(monitors, count, false)
	if len(activeIndexes) < count {
		return nil, fmt.Errorf("need at least %d monitors for this layout", count)
	}

	return activeIndexes, nil
}

func pickPreferredIndexes(monitors []hypr.Monitor, count int, externalOnly bool) []int {
	indexes := make([]int, 0, count)

	for idx, mon := range monitors {
		if !isLaptop(mon.Name) {
			indexes = append(indexes, idx)
			if len(indexes) == count {
				return indexes
			}
		}
	}

	if externalOnly {
		return indexes
	}

	for idx, mon := range monitors {
		if isLaptop(mon.Name) {
			indexes = append(indexes, idx)
			if len(indexes) == count {
				return indexes
			}
		}
	}

	return indexes
}

func findLaptopIndex(monitors []hypr.Monitor) int {
	for idx, mon := range monitors {
		if isLaptop(mon.Name) {
			return idx
		}
	}

	return -1
}

func isLaptop(name string) bool {
	return strings.HasPrefix(name, "eDP") || strings.HasPrefix(name, "LVDS")
}
