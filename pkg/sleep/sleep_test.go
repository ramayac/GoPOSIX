package sleep

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSleepMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, nil, &out, &out, "")
	if rc != 1 {
		t.Errorf("expected 1, got %d", rc)
	}
}

func TestSleepGoDuration(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"1ms"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
}

func TestSleepFloat(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"0.001"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
}

func TestSleepJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "0.001"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	if data["requested"].(float64) <= 0 {
		t.Errorf("requested %v, want > 0", data["requested"])
	}
	if data["duration"].(float64) <= 0 {
		t.Errorf("duration %v, want > 0", data["duration"])
	}
}

func TestSleepJSONShortFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "1ms"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	if data["interrupted"] != false {
		t.Errorf("interrupted %v, want false", data["interrupted"])
	}
}
