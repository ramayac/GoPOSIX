package truefalse

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTrueExitsZero(t *testing.T) {
	var buf bytes.Buffer
	if code := runTrue(nil, &buf); code != 0 {
		t.Errorf("true: expected exit 0, got %d", code)
	}
}

func TestFalseExitsOne(t *testing.T) {
	var buf bytes.Buffer
	if code := runFalse(nil, &buf); code != 1 {
		t.Errorf("false: expected exit 1, got %d", code)
	}
}

func TestTrueJSON(t *testing.T) {
	var buf bytes.Buffer
	code := runTrue([]string{"--json"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, buf.String())
	}
	data := env["data"].(map[string]interface{})
	if data["exitCode"].(float64) != 0 {
		t.Errorf("exitCode %v, want 0", data["exitCode"])
	}
	if data["value"] != true {
		t.Errorf("value %v, want true", data["value"])
	}
}

func TestFalseJSON(t *testing.T) {
	var buf bytes.Buffer
	code := runFalse([]string{"-j"}, &buf)
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, buf.String())
	}
	data := env["data"].(map[string]interface{})
	if data["exitCode"].(float64) != 1 {
		t.Errorf("exitCode %v, want 1", data["exitCode"])
	}
	if data["value"] != false {
		t.Errorf("value %v, want false", data["value"])
	}
}

func TestTrueJSONNoArgs(t *testing.T) {
	// true with no args should still work silently (json not set)
	var buf bytes.Buffer
	if code := runTrue([]string{}, &buf); code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}
