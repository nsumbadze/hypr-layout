package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type layoutOption struct {
	ID   int
	Name string
}

func promptLayoutSelection(r io.Reader, w io.Writer) (layoutOption, error) {
	options := layoutOptions()
	reader := bufio.NewReader(r)

	for {
		fmt.Fprint(w, renderLayoutOptions(options))

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && strings.TrimSpace(input) != "" {
				selection, parseErr := parseLayoutSelection(input, options)
				if parseErr == nil {
					return selection, nil
				}
			}

			return layoutOption{}, err
		}

		selection, err := parseLayoutSelection(input, options)
		if err != nil {
			fmt.Fprintf(w, "Invalid selection: %v\n\n", err)
			continue
		}

		return selection, nil
	}
}

func layoutOptions() []layoutOption {
	return []layoutOption{
		{ID: 1, Name: "Laptop only"},
		{ID: 2, Name: "External only"},
		{ID: 3, Name: "Dual horizontal"},
		{ID: 4, Name: "Triple horizontal"},
		{ID: 5, Name: "Quit"},
	}
}

func parseLayoutSelection(input string, options []layoutOption) (layoutOption, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return layoutOption{}, fmt.Errorf("enter a number between 1 and %d", len(options))
	}

	choice, err := strconv.Atoi(value)
	if err != nil {
		return layoutOption{}, fmt.Errorf("enter a valid number")
	}

	for _, option := range options {
		if option.ID == choice {
			return option, nil
		}
	}

	return layoutOption{}, fmt.Errorf("enter a number between 1 and %d", len(options))
}

func activeIndexesForLayout(monitors []monitor, layoutID int) ([]int, error) {
	switch layoutID {
	case 1:
		return buildLaptopOnly(monitors)
	case 2:
		return buildExternalOnly(monitors)
	case 3:
		return buildHorizontalLayout(monitors, 2)
	case 4:
		return buildHorizontalLayout(monitors, 3)
	default:
		return nil, fmt.Errorf("unknown layout: %d", layoutID)
	}
}

func buildLaptopOnly(monitors []monitor) ([]int, error) {
	laptopIndex := findLaptopMonitorIndex(monitors)
	if laptopIndex < 0 {
		return nil, fmt.Errorf("no laptop display detected")
	}

	return []int{laptopIndex}, nil
}

func buildExternalOnly(monitors []monitor) ([]int, error) {
	activeIndexes := pickPreferredMonitorIndexes(monitors, 1, true)
	if len(activeIndexes) == 0 {
		return nil, fmt.Errorf("no external display detected")
	}

	return activeIndexes, nil
}

func buildHorizontalLayout(monitors []monitor, count int) ([]int, error) {
	activeIndexes := pickPreferredMonitorIndexes(monitors, count, false)
	if len(activeIndexes) < count {
		return nil, fmt.Errorf("need at least %d monitors for this layout", count)
	}

	return activeIndexes, nil
}

func pickPreferredMonitorIndexes(monitors []monitor, count int, externalOnly bool) []int {
	indexes := make([]int, 0, count)

	for idx, mon := range monitors {
		if !isLaptopMonitor(mon.Name) {
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
		if isLaptopMonitor(mon.Name) {
			indexes = append(indexes, idx)
			if len(indexes) == count {
				return indexes
			}
		}
	}

	return indexes
}

func findLaptopMonitorIndex(monitors []monitor) int {
	for idx, mon := range monitors {
		if isLaptopMonitor(mon.Name) {
			return idx
		}
	}

	return -1
}

func isLaptopMonitor(name string) bool {
	return strings.HasPrefix(name, "eDP") || strings.HasPrefix(name, "LVDS")
}
