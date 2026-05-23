package makedevs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMakedevsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-h"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: makedevs")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestMakedevsMissingArg(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for missing ROOTDIR")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure in JSON mode for missing ROOTDIR")
	}
}

func TestMakedevsFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag", "."}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure on invalid flags")
	}
}

func TestMakedevsTableParsing(t *testing.T) {
	// Mock functions to avoid requiring root for tests!
	oldMknod := mknodFn
	oldChown := chownFn
	defer func() {
		mknodFn = oldMknod
		chownFn = oldChown
	}()

	mknodFn = func(path string, mode uint32, dev int) error {
		return nil
	}
	chownFn = func(path string, uid, gid int) error {
		return nil
	}

	tempDir, err := os.MkdirTemp("", "makedevs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tableContent := `
# A sample device table comment
/dev		d	755	0	0	-	-	-	-	-
/dev/fb		c	640	0	5	29	0	0	32	2
/dev/null	c	666	0	0	1	3	0	0	-
/dev/ram	b	640	0	0	1	0	0	1	3
/dev/pipe	p	600	0	0	-	-	-	-	-
/etc/hosts	f	644	0	0	-	-	-	-	-
`
	stdin := bytes.NewReader([]byte(tableContent))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-d", "-", "."}, stdin, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	// Verify folders/files were created
	devDir := filepath.Join(tempDir, "dev")
	if _, err := os.Stat(devDir); err != nil {
		t.Error("Expected dev directory to be created")
	}

	pipePath := filepath.Join(tempDir, "dev/pipe")
	if _, err := os.Stat(pipePath); err != nil {
		t.Error("Expected dev/pipe FIFO to be created")
	}

	hostsPath := filepath.Join(tempDir, "etc/hosts")
	if _, err := os.Stat(hostsPath); err != nil {
		t.Error("Expected etc/hosts file to be created")
	}
}

func TestMakedevsFileBasedTable(t *testing.T) {
	oldMknod := mknodFn
	oldChown := chownFn
	defer func() {
		mknodFn = oldMknod
		chownFn = oldChown
	}()
	mknodFn = func(path string, mode uint32, dev int) error { return nil }
	chownFn = func(path string, uid, gid int) error { return nil }

	tempDir, err := os.MkdirTemp("", "makedevs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tableContent := `
/dev/null	c	666	0	0	1	3	0	0	-
/dev/invalid	x	666	0	0	-	-	-	-	-
`
	tablePath := filepath.Join(tempDir, "table.txt")
	if err := os.WriteFile(tablePath, []byte(tableContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Standard file based run (will fail due to type 'x' being invalid)
	code := run([]string{"-d", "table.txt", "."}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected error because of invalid device type 'x'")
	}

	// Case 2: Missing table file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-d", "nonexistent-table.txt", "."}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected error for nonexistent table file")
	}

	// Case 3: JSON mode run with valid/invalid parsing
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-d", "table.txt", "."}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure in JSON mode due to type 'x'")
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"failedCount":1`)) || !bytes.Contains(stdout.Bytes(), []byte(`"unknown device type: x"`)) {
		t.Errorf("Expected valid JSON error report, got:\n%s", stdout.String())
	}
}
