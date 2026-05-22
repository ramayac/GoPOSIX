package tree

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreeBasic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tree_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested structure:
	// tmpDir/
	//   dir1/
	//     file1.txt
	//   dir2/
	//   file2.txt
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	if err := os.Mkdir(dir1, 0755); err != nil {
		t.Fatalf("failed to create dir1: %v", err)
	}
	if err := os.Mkdir(dir2, 0755); err != nil {
		t.Fatalf("failed to create dir2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "file1.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write file1.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write file2.txt: %v", err)
	}

	var stdout, stderr bytes.Buffer
	gotExit := run([]string{tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d. stderr: %s", gotExit, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "dir1") || !strings.Contains(out, "file1.txt") || !strings.Contains(out, "file2.txt") {
		t.Errorf("stdout missing expected nodes: %s", out)
	}
	if !strings.Contains(out, "2 directories, 2 files") {
		t.Errorf("stdout missing correct report summary, got: %s", out)
	}
}

func TestTreeEdgeCases(t *testing.T) {
	// 1. Invalid Flag
	var stdout, stderr bytes.Buffer
	gotExit := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for invalid flag, got %d", gotExit)
	}

	// 2. Non-existent dir error opening
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"nonexistent_dir"}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for nonexistent dir, got %d", gotExit)
	}
	if !strings.Contains(stdout.String(), "nonexistent_dir [error opening dir]") {
		t.Errorf("expected error message in stdout, got: %s", stdout.String())
	}

	// 3. JSON mode
	tmpDir, _ := os.MkdirTemp("", "tree_json_test")
	defer os.RemoveAll(tmpDir)
	_ = os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("hi"), 0644)

	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"--json", tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0 for json mode, got %d", gotExit)
	}
	if !strings.Contains(stdout.String(), `"type":"file"`) || !strings.Contains(stdout.String(), `"name":"hello.txt"`) {
		t.Errorf("expected valid JSON schema, got: %s", stdout.String())
	}
}
