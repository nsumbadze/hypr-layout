package layout

import (
	"sort"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
)

// CurrentArrangement orders the active monitors the way they sit on the desk
// now and names the direction that reproduces that arrangement, so the hub
// opens on the real layout rather than on detection order.
//
// Generated layouts anchor the first monitor at 0x0 and stack the rest after
// it, so when the monitor at the origin is the last one along the axis the
// stack grew towards negative coordinates: a right-to-left or bottom-to-top
// layout. Monitors reporting no useful positions fall back to detection order
// and left-to-right.
func CurrentArrangement(monitors []hypr.Monitor, activeIndexes []int) ([]int, Direction) {
	order := append([]int(nil), activeIndexes...)
	if len(order) < 2 {
		return order, LeftToRight
	}

	vertical := spread(monitors, order, func(m hypr.Monitor) int { return m.Y }) >
		spread(monitors, order, func(m hypr.Monitor) int { return m.X })
	along := func(idx int) int {
		if vertical {
			return monitors[idx].Y
		}
		return monitors[idx].X
	}
	sort.SliceStable(order, func(i, j int) bool { return along(order[i]) < along(order[j]) })

	first, last := monitors[order[0]], monitors[order[len(order)-1]]
	reversed := atOrigin(last) && !atOrigin(first)
	if reversed {
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}

	switch {
	case vertical && reversed:
		return order, BottomToTop
	case vertical:
		return order, TopToBottom
	case reversed:
		return order, RightToLeft
	default:
		return order, LeftToRight
	}
}

func atOrigin(mon hypr.Monitor) bool {
	return mon.X == 0 && mon.Y == 0
}

// spread is the distance between the smallest and largest coordinate along
// one axis across the given monitors.
func spread(monitors []hypr.Monitor, indexes []int, coord func(hypr.Monitor) int) int {
	lo, hi := coord(monitors[indexes[0]]), coord(monitors[indexes[0]])
	for _, idx := range indexes[1:] {
		v := coord(monitors[idx])
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}

	return hi - lo
}
