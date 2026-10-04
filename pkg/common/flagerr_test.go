package common

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestHasJSONFlag(t *testing.T) {
	if !HasJSONFlag([]string{"-n", "--json", "x"}) {
		t.Fatal("expected --json to be found")
	}
	if HasJSONFlag([]string{"-n", "x"}) {
		t.Fatal("did not expect --json to be found")
	}
	if HasJSONFlag(nil) {
		t.Fatal("did not expect --json in empty args")
	}
}

func TestRenderFlagErrorPlain(t *testing.T) {
	var out bytes.Buffer
	code := RenderFlagError("tsort", []string{"--nope"}, errors.New("unknown flag: --nope"), &out, 2)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if got := out.String(); got != "tsort: unknown flag: --nope\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestRenderFlagErrorJSON(t *testing.T) {
	var out bytes.Buffer
	code := RenderFlagError("tsort", []string{"--json", "--nope"}, errors.New("unknown flag: --nope"), &out, 2)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	got := out.String()
	if !strings.Contains(got, `"code":"FLAG_ERROR"`) {
		t.Fatalf("output missing FLAG_ERROR: %q", got)
	}
	if !strings.Contains(got, `"exitCode":2`) {
		t.Fatalf("output missing exitCode: %q", got)
	}
}

func TestRenderFlagErrorJSONExitOne(t *testing.T) {
	var out bytes.Buffer
	code := RenderFlagError("unzip", []string{"--json", "--nope"}, errors.New("unknown flag: --nope"), &out, 1)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), `"exitCode":1`) {
		t.Fatalf("output missing exitCode 1: %q", out.String())
	}
}
