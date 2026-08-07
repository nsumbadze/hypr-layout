package hypr

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestReloadHyprlandRunsHyprctlReload(t *testing.T) {
	var gotName string
	var gotArgs []string

	runner := func(_ context.Context, name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := Reload(context.Background(), runner); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}

	if gotName != "hyprctl" {
		t.Fatalf("unexpected command name: %q", gotName)
	}

	wantArgs := []string{"reload"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", gotArgs, wantArgs)
	}
}

func TestReloadHyprlandWrapsRunnerError(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) error {
		return errors.New("boom")
	}

	err := Reload(context.Background(), runner)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "run hyprctl reload") {
		t.Fatalf("unexpected error: %v", err)
	}
}
