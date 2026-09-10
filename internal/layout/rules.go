package layout

import (
	"fmt"
	"strconv"
	"strings"
)

// Format is the syntax a monitor config is written in. Hyprland 0.55 moved
// its configuration to Lua; the classic hyprlang syntax is still read by
// older versions and by setups that never migrated.
type Format string

const (
	FormatConf Format = "conf"
	FormatLua  Format = "lua"
)

// Rule is one monitor's settings, independent of the syntax they are written
// in. Mode, Position and Scale are kept as text so a rule read back from a
// profile is reproduced exactly, including hand-written values such as
// "preferred" or "auto".
type Rule struct {
	Output   string
	Disabled bool
	Mode     string
	Position string
	Scale    string
	// Transform is a Hyprland transform value (0-7); 0 is omitted when written.
	Transform int
	// VRR is a Hyprland vrr value: 0 off, 1 on, 2 fullscreen only.
	VRR int
	// Mirror names the monitor this one mirrors; empty for a normal output.
	Mirror string
}

// Lines renders the rules in the given format, one line per rule.
func Lines(rules []Rule, format Format) []string {
	if format == FormatLua {
		return LuaLines(rules)
	}

	return ConfLines(rules)
}

// ConfLines renders the rules in the classic `monitor = ...` syntax. It is
// also the syntax profiles are stored in, so it doubles as the tool's
// portable representation of a layout.
func ConfLines(rules []Rule) []string {
	lines := make([]string, 0, len(rules))
	for _, rule := range rules {
		lines = append(lines, ConfLine(rule))
	}

	return lines
}

func ConfLine(rule Rule) string {
	if rule.Disabled {
		return fmt.Sprintf("monitor = %s, disable", rule.Output)
	}

	line := fmt.Sprintf("monitor = %s, %s, %s, %s", rule.Output, rule.Mode, rule.Position, rule.Scale)
	if rule.Mirror != "" {
		line += ", mirror, " + rule.Mirror
	}
	if rule.Transform != 0 {
		line += fmt.Sprintf(", transform, %d", rule.Transform)
	}
	if rule.VRR != 0 {
		line += fmt.Sprintf(", vrr, %d", rule.VRR)
	}

	return line
}

// LuaLines renders the rules as hl.monitor calls for Hyprland's Lua config.
func LuaLines(rules []Rule) []string {
	lines := make([]string, 0, len(rules))
	for _, rule := range rules {
		lines = append(lines, LuaLine(rule))
	}

	return lines
}

func LuaLine(rule Rule) string {
	if rule.Disabled {
		return fmt.Sprintf("hl.monitor({ output = %s, disabled = true })", luaString(rule.Output))
	}

	fields := []string{
		"output = " + luaString(rule.Output),
		"mode = " + luaString(rule.Mode),
		"position = " + luaString(rule.Position),
		"scale = " + luaScale(rule.Scale),
	}
	if rule.Mirror != "" {
		fields = append(fields, "mirror = "+luaString(rule.Mirror))
	}
	if rule.Transform != 0 {
		fields = append(fields, fmt.Sprintf("transform = %d", rule.Transform))
	}
	if rule.VRR != 0 {
		fields = append(fields, fmt.Sprintf("vrr = %d", rule.VRR))
	}

	return "hl.monitor({ " + strings.Join(fields, ", ") + " })"
}

func luaString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

// luaScale writes a numeric scale as a number and anything else ("auto") as
// a string, which is how hl.monitor accepts the field.
func luaScale(scale string) string {
	if _, err := strconv.ParseFloat(scale, 64); err == nil {
		return scale
	}

	return luaString(scale)
}

// ParseConfLines reads rules back from classic-syntax lines, as stored in
// profiles, so they can be written out in whichever format Hyprland reads.
func ParseConfLines(lines []string) ([]Rule, error) {
	rules := make([]Rule, 0, len(lines))
	for i, line := range lines {
		rule, err := ParseConfLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func ParseConfLine(line string) (Rule, error) {
	body, ok := strings.CutPrefix(strings.TrimSpace(line), "monitor")
	if ok {
		body, ok = strings.CutPrefix(strings.TrimSpace(body), "=")
	}
	if !ok {
		return Rule{}, fmt.Errorf("not a monitor rule: %q", line)
	}

	fields := strings.Split(body, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	if len(fields) < 2 || fields[0] == "" {
		return Rule{}, fmt.Errorf("incomplete monitor rule: %q", line)
	}

	rule := Rule{Output: fields[0]}
	if len(fields) == 2 && fields[1] == "disable" {
		rule.Disabled = true
		return rule, nil
	}
	if len(fields) < 4 {
		return Rule{}, fmt.Errorf("incomplete monitor rule: %q", line)
	}

	rule.Mode, rule.Position, rule.Scale = fields[1], fields[2], fields[3]
	extras := fields[4:]
	if len(extras)%2 != 0 {
		return Rule{}, fmt.Errorf("unpaired keyword in monitor rule: %q", line)
	}
	for i := 0; i < len(extras); i += 2 {
		key, value := extras[i], extras[i+1]
		switch key {
		case "mirror":
			rule.Mirror = value
		case "transform", "vrr":
			n, err := strconv.Atoi(value)
			if err != nil {
				return Rule{}, fmt.Errorf("%s must be a number in monitor rule: %q", key, line)
			}
			if key == "transform" {
				rule.Transform = n
			} else {
				rule.VRR = n
			}
		default:
			return Rule{}, fmt.Errorf("unsupported keyword %q in monitor rule: %q", key, line)
		}
	}

	return rule, nil
}
