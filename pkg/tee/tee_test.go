package tee

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	// Not easily testable as it binds os.Stdout and os.Stdin in run().
	// We'll test the io.MultiWriter aspect.

	dir := t.TempDir()
	file1 := filepath.Join(dir, "out1.txt")
	file2 := filepath.Join(dir, "out2.txt")

	f1, _ := os.Create(file1)
	f2, _ := os.Create(file2)

	var stdout bytes.Buffer
	writers := []io.Writer{&stdout, f1, f2}
	multiWriter := io.MultiWriter(writers...)

	io.Copy(multiWriter, strings.NewReader("hello tee\n"))
	f1.Close()
	f2.Close()

	if stdout.String() != "hello tee\n" {
		t.Error("stdout missing")
	}

	b1, _ := os.ReadFile(file1)
	if string(b1) != "hello tee\n" {
		t.Error("file1 missing")
	}
	b2, _ := os.ReadFile(file2)
	if string(b2) != "hello tee\n" {
		t.Error("file2 missing")
	}
}

func TestCountingWriter(t *testing.T) {
	var buf bytes.Buffer
	cw := &countingWriter{w: &buf}
	_, err := cw.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if cw.count != 5 {
		t.Errorf("count %d, want 5", cw.count)
	}
}

func TestTeeJSON(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.txt")

	// Simulate stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.WriteString("hello world\n")
		w.Close()
	}()

	var out bytes.Buffer
	code := run([]string{"--json", outFile}, r, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	if data["bytesWritten"].(float64) != 12 {
		t.Errorf("bytesWritten %v, want 12", data["bytesWritten"])
	}
	files := data["files"].([]interface{})
	if len(files) != 1 || files[0] != outFile {
		t.Errorf("files %v, want [%q]", files, outFile)
	}

	// Verify file was written
	b, _ := os.ReadFile(outFile)
	if string(b) != "hello world\n" {
		t.Errorf("file content %q, want 'hello world\\n'", string(b))
	}
}

func TestTeeJSONShortFlag(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.txt")

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.WriteString("test\n")
		w.Close()
	}()

	var out bytes.Buffer
	code := run([]string{"--json", outFile}, r, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["bytesWritten"].(float64) != 5 {
		t.Errorf("bytesWritten %v, want 5", data["bytesWritten"])
	}
}
