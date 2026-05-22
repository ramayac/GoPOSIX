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

	// 4. Invalid depth value
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-L", "abc", "."}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for invalid depth, got %d", gotExit)
	}
}

func TestTreeDepthLimit(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a 3-level nested structure
	// tmpDir/
	//   dir1/
	//     dir2/
	//       file3.txt
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir1", "dir2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir1", "dir2", "file3.txt"), []byte("deep"), 0644)

	var stdout, stderr bytes.Buffer
	// With -L 1, should NOT show dir2 or file3.txt
	gotExit := run([]string{"-L", "1", tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d. stderr: %s", gotExit, stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "dir2") {
		t.Errorf("expected dir2 to be hidden by -L 1, got: %s", out)
	}
	if strings.Contains(out, "file3.txt") {
		t.Errorf("expected file3.txt to be hidden by -L 1, got: %s", out)
	}
}

func TestTreeAllHidden(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".hidden_file"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "visible_file"), []byte("public"), 0644)
	os.Mkdir(filepath.Join(tmpDir, ".hidden_dir"), 0755)

	var stdout, stderr bytes.Buffer

	// Without -a: hidden entries should be absent
	gotExit := run([]string{tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d", gotExit)
	}
	if strings.Contains(stdout.String(), ".hidden_file") || strings.Contains(stdout.String(), ".hidden_dir") {
		t.Errorf("hidden entries leaked without -a: %s", stdout.String())
	}

	// With -a: hidden entries should be present
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-a", tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d", gotExit)
	}
	if !strings.Contains(stdout.String(), ".hidden_file") {
		t.Errorf("hidden file missing with -a: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), ".hidden_dir") {
		t.Errorf("hidden dir missing with -a: %s", stdout.String())
	}
}

func TestTreeDirsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("data"), 0644)

	var stdout, stderr bytes.Buffer
	gotExit := run([]string{"-d", tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d", gotExit)
	}
	out := stdout.String()
	// file.txt should NOT appear with -d
	if strings.Contains(out, "file.txt") {
		t.Errorf("file.txt should not appear with -d: %s", out)
	}
	if !strings.Contains(out, "subdir") {
		t.Errorf("subdir missing with -d: %s", out)
	}
	// Report should show 1 directory, 0 files
	if !strings.Contains(out, "1 directories, 0 files") {
		t.Errorf("report mismatch for -d: %s", out)
	}
}

func TestTreeSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "target_file"), []byte("target"), 0644)
	os.Symlink(filepath.Join(tmpDir, "target_file"), filepath.Join(tmpDir, "link_to_file"))
	os.Mkdir(filepath.Join(tmpDir, "target_dir"), 0755)
	os.Symlink(filepath.Join(tmpDir, "target_dir"), filepath.Join(tmpDir, "link_to_dir"))

	var stdout, stderr bytes.Buffer
	gotExit := run([]string{tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d", gotExit)
	}
	out := stdout.String()
	if !strings.Contains(out, "link_to_file ->") {
		t.Errorf("symlink file missing arrow indicator: %s", out)
	}
	if !strings.Contains(out, "link_to_dir ->") {
		t.Errorf("symlink dir missing arrow indicator: %s", out)
	}
}

func TestTreeUnreadableDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a dir and remove permissions to simulate readdir error
	lockedDir := filepath.Join(tmpDir, "locked")
	os.Mkdir(lockedDir, 0000)
	defer os.Chmod(lockedDir, 0755) // cleanup

	var stdout, stderr bytes.Buffer
	gotExit := run([]string{tmpDir}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for unreadable dir, got %d. stdout: %s", gotExit, stdout.String())
	}
}
