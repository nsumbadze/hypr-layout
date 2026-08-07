package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func promptReloadConfirmation(r io.Reader, w io.Writer) (bool, error) {
	return promptYesNo(r, w, "\nReload Hyprland now? (y/n) ")
}

func promptYesNo(r io.Reader, w io.Writer, prompt string) (bool, error) {
	reader := bufio.NewReader(r)

	for {
		fmt.Fprint(w, prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && input != "" {
				return parseConfirmation(input)
			}

			return false, err
		}

		value, parseErr := parseConfirmation(input)
		if parseErr != nil {
			fmt.Fprintln(w, "Invalid selection: enter y or n.")
			continue
		}

		return value, nil
	}
}

func promptApplyConfirmation(r io.Reader, w io.Writer) (bool, error) {
	return promptYesNo(r, w, "\nApply this layout to ~/.config/hypr/monitors.conf? (y/n) ")
}

func parseConfirmation(input string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("enter y or n")
	}
}
