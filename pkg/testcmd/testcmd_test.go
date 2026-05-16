package testcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateEmpty(t *testing.T) {
	result, err := Evaluate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("empty expression should be false")
	}
}

func TestEvaluateStringNonEmpty(t *testing.T) {
	result, err := Evaluate([]string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("non-empty string should be true")
	}
}

func TestEvaluateStringEmpty(t *testing.T) {
	result, err := Evaluate([]string{""})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("empty string should be false")
	}
}

func TestEvaluateStringZ(t *testing.T) {
	result, err := Evaluate([]string{"-z", ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-z '' should be true")
	}

	result, err = Evaluate([]string{"-z", "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("-z 'hello' should be false")
	}
}

func TestEvaluateStringN(t *testing.T) {
	result, err := Evaluate([]string{"-n", "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-n 'hello' should be true")
	}

	result, err = Evaluate([]string{"-n", ""})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("-n '' should be false")
	}
}

func TestEvaluateStringEquals(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"abc", "=", "abc"}, true},
		{[]string{"abc", "=", "def"}, false},
		{[]string{"abc", "!=", "def"}, true},
		{[]string{"abc", "!=", "abc"}, false},
	}
	for _, tc := range tests {
		result, err := Evaluate(tc.args)
		if err != nil {
			t.Errorf("Evaluate(%v) error: %v", tc.args, err)
			continue
		}
		if result != tc.want {
			t.Errorf("Evaluate(%v) = %v, want %v", tc.args, result, tc.want)
		}
	}
}

func TestEvaluateIntegerComparisons(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"5", "-eq", "5"}, true},
		{[]string{"5", "-eq", "3"}, false},
		{[]string{"5", "-ne", "3"}, true},
		{[]string{"3", "-lt", "5"}, true},
		{[]string{"5", "-lt", "3"}, false},
		{[]string{"5", "-le", "5"}, true},
		{[]string{"5", "-gt", "3"}, true},
		{[]string{"3", "-gt", "5"}, false},
		{[]string{"5", "-ge", "5"}, true},
		{[]string{"-1", "-lt", "0"}, true},
	}
	for _, tc := range tests {
		result, err := Evaluate(tc.args)
		if err != nil {
			t.Errorf("Evaluate(%v) error: %v", tc.args, err)
			continue
		}
		if result != tc.want {
			t.Errorf("Evaluate(%v) = %v, want %v", tc.args, result, tc.want)
		}
	}
}

func TestEvaluateNot(t *testing.T) {
	result, err := Evaluate([]string{"!", "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("! 'hello' should be false")
	}

	result, err = Evaluate([]string{"!", ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("! '' should be true")
	}
}

func TestEvaluateAnd(t *testing.T) {
	result, err := Evaluate([]string{"hello", "-a", "world"})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("'hello' -a 'world' should be true")
	}

	result, err = Evaluate([]string{"hello", "-a", ""})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("'hello' -a '' should be false")
	}
}

func TestEvaluateOr(t *testing.T) {
	result, err := Evaluate([]string{"", "-o", "world"})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("'' -o 'world' should be true")
	}

	result, err = Evaluate([]string{"", "-o", ""})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("'' -o '' should be false")
	}
}

func TestEvaluateParens(t *testing.T) {
	// ( '' -o 'x' ) -a 'y' → true
	result, err := Evaluate([]string{"(", "", "-o", "x", ")", "-a", "y"})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("( '' -o 'x' ) -a 'y' should be true")
	}
}

func TestEvaluateFileExists(t *testing.T) {
	// Test with a file that definitely exists
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Evaluate([]string{"-e", testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-e should be true for existing file")
	}

	result, err = Evaluate([]string{"-f", testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-f should be true for regular file")
	}

	result, err = Evaluate([]string{"-d", testFile})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("-d should be false for regular file")
	}
}

func TestEvaluateFileDir(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := Evaluate([]string{"-d", tmpDir})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-d should be true for directory")
	}
}

func TestEvaluateFileNotExists(t *testing.T) {
	result, err := Evaluate([]string{"-e", "/nonexistent_file_path_1234567890"})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("-e should be false for nonexistent file")
	}
}

func TestEvaluateFileSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Non-empty file
	full := filepath.Join(tmpDir, "full")
	os.WriteFile(full, []byte("data"), 0644)
	result, err := Evaluate([]string{"-s", full})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-s should be true for non-empty file")
	}

	// Empty file
	empty := filepath.Join(tmpDir, "empty")
	os.WriteFile(empty, []byte(""), 0644)
	result, err = Evaluate([]string{"-s", empty})
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("-s should be false for empty file")
	}
}

func TestEvaluateFileReadable(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readable")
	os.WriteFile(testFile, []byte("data"), 0644)

	result, err := Evaluate([]string{"-r", testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-r should be true for readable file")
	}
}

func TestEvaluateFileExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "executable")
	os.WriteFile(testFile, []byte("#!/bin/sh"), 0755)

	result, err := Evaluate([]string{"-x", testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("-x should be true for executable file")
	}
}

func TestRunTestCLI(t *testing.T) {
	var buf bytes.Buffer
	code := runTest([]string{"hello", "=", "hello"}, &buf)
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
}

func TestRunTestCLIFalse(t *testing.T) {
	var buf bytes.Buffer
	code := runTest([]string{"hello", "=", "world"}, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunTestCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := runTest([]string{"--json", "hello", "=", "hello"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["result"] != true {
		t.Errorf("got %v, want true", data["result"])
	}
}

func TestRunTestCLIJSONShort(t *testing.T) {
	var buf bytes.Buffer
	code := runTest([]string{"-j", "hello", "=", "hello"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["result"] != true {
		t.Errorf("got %v, want true", data["result"])
	}
}

func TestRunTestJSONMiddleware(t *testing.T) {
	// Test that --json can appear in the middle of expression args (edge case)
	var buf bytes.Buffer
	code := runTest([]string{"hello", "--json", "=", "hello"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0 (json extracted)", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["result"] != true {
		t.Errorf("got %v, want true", data["result"])
	}
}

func TestRunBracket(t *testing.T) {
	var buf bytes.Buffer
	code := runBracket([]string{"hello", "=", "hello", "]"}, &buf)
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
}

func TestRunBracketMissingClose(t *testing.T) {
	var buf bytes.Buffer
	code := runBracket([]string{"hello", "=", "hello"}, &buf)
	if code != 2 {
		t.Errorf("exit code %d, want 2", code)
	}
}

func TestRunBracketJSON(t *testing.T) {
	var buf bytes.Buffer
	code := runBracket([]string{"--json", "hello", "=", "hello", "]"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["result"] != true {
		t.Errorf("got %v, want true", data["result"])
	}
}

func TestEvaluateIntegerError(t *testing.T) {
	_, err := Evaluate([]string{"abc", "-eq", "5"})
	if err == nil {
		t.Error("expected integer expression error")
	}
}

func TestEvaluateDoubleNot(t *testing.T) {
	result, err := Evaluate([]string{"!", "!", "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("!! 'hello' should be true")
	}
}
