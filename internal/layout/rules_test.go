package layout

import (
	"reflect"
	"testing"
)

func TestLuaLinesRenderEveryField(t *testing.T) {
	rules := []Rule{
		{Output: "DP-1", Mode: "2560x1440@165", Position: "0x0", Scale: "1", Transform: 1, VRR: 2},
		{Output: "HDMI-A-1", Mode: "1920x1080@60", Position: "0x0", Scale: "1.5", Mirror: "DP-1"},
		{Output: "eDP-1", Disabled: true},
	}

	got := LuaLines(rules)
	want := []string{
		`hl.monitor({ output = "DP-1", mode = "2560x1440@165", position = "0x0", scale = 1, transform = 1, vrr = 2 })`,
		`hl.monitor({ output = "HDMI-A-1", mode = "1920x1080@60", position = "0x0", scale = 1.5, mirror = "DP-1" })`,
		`hl.monitor({ output = "eDP-1", disabled = true })`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lua lines:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestLuaLineQuotesNonNumericScale(t *testing.T) {
	got := LuaLine(Rule{Output: "DP-1", Mode: "preferred", Position: "auto", Scale: "auto"})
	want := `hl.monitor({ output = "DP-1", mode = "preferred", position = "auto", scale = "auto" })`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLinesPicksFormat(t *testing.T) {
	rules := []Rule{{Output: "eDP-1", Disabled: true}}
	if got := Lines(rules, FormatConf); got[0] != "monitor = eDP-1, disable" {
		t.Fatalf("conf: got %q", got[0])
	}
	if got := Lines(rules, FormatLua); got[0] != `hl.monitor({ output = "eDP-1", disabled = true })` {
		t.Fatalf("lua: got %q", got[0])
	}
}

// Profiles are stored as classic lines, so every line the tool can write must
// read back into the rule that produced it.
func TestParseConfLinesRoundTrip(t *testing.T) {
	rules := []Rule{
		{Output: "DP-1", Mode: "2560x1440@165.08", Position: "0x0", Scale: "1", Transform: 3, VRR: 1},
		{Output: "HDMI-A-2", Mode: "2560x1440@144", Position: "-2560x0", Scale: "1.25"},
		{Output: "eDP-1", Mode: "2880x1800@120", Position: "0x0", Scale: "2", Mirror: "DP-1"},
		{Output: "DP-2", Disabled: true},
	}

	got, err := ParseConfLines(ConfLines(rules))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, rules) {
		t.Fatalf("round trip:\ngot:  %#v\nwant: %#v", got, rules)
	}
}

func TestParseConfLineAcceptsHandWrittenSpacing(t *testing.T) {
	got, err := ParseConfLine("monitor=DP-1,preferred,auto,auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Rule{Output: "DP-1", Mode: "preferred", Position: "auto", Scale: "auto"}
	if got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestParseConfLineRejectsOtherRules(t *testing.T) {
	for _, line := range []string{
		"workspace = 1, monitor:DP-1",
		"monitor = DP-1",
		"monitor = DP-1, 2560x1440@165, 0x0",
		"monitor = DP-1, 2560x1440@165, 0x0, 1, transform",
		"monitor = DP-1, 2560x1440@165, 0x0, 1, transform, ninety",
		"monitor = DP-1, 2560x1440@165, 0x0, 1, bitdepth, 10",
	} {
		if _, err := ParseConfLine(line); err == nil {
			t.Fatalf("expected error for %q", line)
		}
	}
}
