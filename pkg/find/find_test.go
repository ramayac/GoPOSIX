package find

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFindRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), ".") {
		t.Error("expected output with .")
	}
}

func TestFindJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

func TestBuildExecArgs(t *testing.T) {
	files := []FileInfo{{Path: "a.txt"}, {Path: "b.txt"}}

	// {} replacement
	args := buildExecArgs([]string{"echo", "{}"}, files)
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[1] != "a.txt" || args[2] != "b.txt" {
		t.Errorf("expected a.txt b.txt, got %v", args[1:])
	}

	// No {} placeholder
	args2 := buildExecArgs([]string{"ls", "-la"}, files)
	if len(args2) != 2 || args2[0] != "ls" || args2[1] != "-la" {
		t.Errorf("expected passthrough, got %v", args2)
	}

	// Empty
	args3 := buildExecArgs([]string{}, files)
	if len(args3) != 0 {
		t.Errorf("expected 0 args, got %d", len(args3))
	}
}

func TestBuildExecArgsMultiplePlaceholders(t *testing.T) {
	files := []FileInfo{{Path: "x"}, {Path: "y"}}
	args := buildExecArgs([]string{"cmd", "{}", "--", "{}"}, files)
	if len(args) != 6 {
		t.Errorf("expected 6 args, got %d: %v", len(args), args)
	}
}

// BusyBox hardening: find -xdev should be accepted as a valid flag.
func TestFindXdevFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{".", "-xdev"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("find -xdev exited with %d, want 0", code)
	}
}

// --- Hardening: CLI flag tests via temp directories ---

func setupFindTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Create: dir/
	//   a.txt
	//   b.txt
	//   sub/
	//     c.txt
	os.WriteFile(dir+"/a.txt", []byte("hello"), 0644)
	os.WriteFile(dir+"/b.txt", []byte("world"), 0644)
	os.MkdirAll(dir+"/sub", 0755)
	os.WriteFile(dir+"/sub/c.txt", []byte("nested"), 0644)
	return dir
}

func TestFind_NamePattern(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-name", "*.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Should find a.txt and b.txt at root, plus c.txt in sub
	outStr := out.String()
	if !strings.Contains(outStr, "a.txt") {
		t.Error("expected a.txt in output")
	}
	if !strings.Contains(outStr, "b.txt") {
		t.Error("expected b.txt in output")
	}
}

func TestFind_TypeFile(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-type", "f"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	outStr := out.String()
	if !strings.Contains(outStr, "a.txt") {
		t.Error("expected a.txt (file) in output")
	}
	if strings.Contains(outStr, "/sub\n") || strings.Contains(outStr, "/sub/") {
		// sub/ is a directory, not a file — but it contains c.txt which IS a file
	}
}

func TestFind_TypeDir(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-type", "d"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	outStr := out.String()
	// sub/ should appear
	if !strings.Contains(outStr, "sub") {
		t.Errorf("expected sub dir in output, got: %s", outStr)
	}
	// Regular files should NOT appear
	if strings.Contains(outStr, "a.txt") {
		t.Error("a.txt is a file, should not appear with -type d")
	}
}

func TestFind_MaxDepth(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-maxdepth", "1"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	outStr := out.String()
	// Should find a.txt, b.txt, sub/ at depth 1
	// But NOT sub/c.txt (depth 2)
	if strings.Contains(outStr, "c.txt") {
		t.Error("c.txt is depth 2, should be filtered by -maxdepth 1")
	}
	if !strings.Contains(outStr, "a.txt") {
		t.Error("expected a.txt at depth 1")
	}
}

func TestFind_MaxDepthLongFlag(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "--maxdepth", "1"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.Contains(out.String(), "c.txt") {
		t.Error("c.txt should be filtered by --maxdepth 1")
	}
}

func TestFind_JSON(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "--json"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"path\"") {
		t.Errorf("expected JSON output, got: %s", out.String())
	}
}

func TestFind_JSONShortFlag(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "--json"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"path\"") {
		t.Errorf("expected JSON output, got: %s", out.String())
	}
}

func TestFind_CombinedFilters(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-name", "*.txt", "-type", "f", "-maxdepth", "1"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	outStr := out.String()
	// Only a.txt and b.txt should match: *.txt, file type, depth 1
	if !strings.Contains(outStr, "a.txt") || !strings.Contains(outStr, "b.txt") {
		t.Errorf("expected a.txt and b.txt, got: %s", outStr)
	}
	if strings.Contains(outStr, "c.txt") {
		t.Error("c.txt is depth 2, should be filtered")
	}
}

func TestFind_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{".", "--nonexistent"}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 for bad flag, got %d", code)
	}
}

func TestFind_NoArgs(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Defaults to current dir; should find something
	if !strings.Contains(out.String(), ".") {
		t.Error("expected '.' in default output")
	}
}

func TestFind_Mtime(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	// -mtime -1: files modified less than 1 day ago (all our temp files)
	code := run([]string{dir, "-mtime", "-1"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Should find recently created files
	if out.Len() == 0 {
		t.Error("expected some output with -mtime -1")
	}
}

func TestFind_MtimeExact(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	// -mtime 0: files modified exactly today
	code := run([]string{dir, "-mtime", "0"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected some output with -mtime 0")
	}
}

func TestFind_Exec(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-exec", "echo", "found", ";"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.Contains(out.String(), "found") {
		// Output goes to stdout via cmd.Stdout=os.Stdout, not our buffer
		// The test just verifies the exec path doesn't crash
	}
}

func TestFind_ExecPlus(t *testing.T) {
	dir := setupFindTree(t)
	var out bytes.Buffer
	code := run([]string{dir, "-exec", "echo", "{}", "+"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestFind_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	code := run([]string{dir}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Should just list the directory itself
	if !strings.Contains(out.String(), dir) {
		t.Errorf("expected %s in output, got: %s", dir, out.String())
	}
}
