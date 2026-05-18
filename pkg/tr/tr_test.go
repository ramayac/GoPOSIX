package tr

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestTrTranslate(t *testing.T) {
	in := "hello world"
	var out bytes.Buffer
	Run(strings.NewReader(in), &out, "a-z", "A-Z", false, false, false)
	if out.String() != "HELLO WORLD" {
		t.Errorf("got %v", out.String())
	}
}

func TestTrDelete(t *testing.T) {
	in := "hello world"
	var out bytes.Buffer
	Run(strings.NewReader(in), &out, "l", "", true, false, false)
	if out.String() != "heo word" {
		t.Errorf("got %v", out.String())
	}
}

func TestTrSqueeze(t *testing.T) {
	in := "heeeello   world"
	var out bytes.Buffer
	Run(strings.NewReader(in), &out, "e ", "", false, true, false)
	if out.String() != "hello world" {
		t.Errorf("got %v", out.String())
	}
}

func TestTrJSONTranslate(t *testing.T) {
	in := "hello world\n"

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.WriteString(in)
		w.Close()
	}()

	var out bytes.Buffer
	code := run([]string{"--json", "a-z", "A-Z"}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	lines := data["lines"].([]interface{})
	if len(lines) != 1 || lines[0] != "HELLO WORLD" {
		t.Errorf("lines %v, want ['HELLO WORLD']", lines)
	}
	if data["bytesIn"].(float64) <= 0 {
		t.Errorf("bytesIn should be > 0, got %v", data["bytesIn"])
	}
	if data["bytesOut"].(float64) <= 0 {
		t.Errorf("bytesOut should be > 0, got %v", data["bytesOut"])
	}
}

func TestTrJSONDelete(t *testing.T) {
	in := "hello world\n"

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.WriteString(in)
		w.Close()
	}()

	var out bytes.Buffer
	code := run([]string{"--json", "-d", "l"}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	lines := data["lines"].([]interface{})
	if len(lines) != 1 || lines[0] != "heo word" {
		t.Errorf("lines %v, want ['heo word']", lines)
	}
}
