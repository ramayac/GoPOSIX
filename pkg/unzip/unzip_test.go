package unzip

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to construct a ZIP archive dynamically for testing.
func createTestZip(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for name, content := range files {
		fh := &zip.FileHeader{
			Name:   name,
			Method: zip.Store,
		}
		if strings.HasSuffix(name, "/") {
			fh.SetMode(0755)
		} else {
			fh.SetMode(0644)
		}
		f, err := w.CreateHeader(fh)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestUnzipHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: unzip")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestUnzipMissingArg(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for missing ZIPFILE argument")
	}

	// JSON mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for missing ZIPFILE argument in JSON mode")
	}
}

func TestUnzipCorruptedZip(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"nonexistent.zip"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for nonexistent archive")
	}
}

func TestUnzipOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "unzip-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipBytes := createTestZip(t, map[string]string{
		"foo/bar":     "hello bar",
		"foo/baz/":    "", // directory
		"foo/baz/qux": "hello qux",
	})

	zipPath := filepath.Join(tempDir, "test.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: List files (-l)
	code := run([]string{"-l", "test.zip"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 on listing, got %d. Stderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("foo/bar")) || !bytes.Contains(stdout.Bytes(), []byte("foo/baz/qux")) {
		t.Errorf("Expected listing to show files, got:\n%s", stdout.String())
	}

	// Case 2: Test integrity (-t)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-t", "test.zip"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 on testing, got %d. Stderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("testing: foo/bar   OK")) {
		t.Errorf("Expected OK status on test, got:\n%s", stdout.String())
	}

	// Case 3: Print to stdout (-p)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-p", "test.zip", "foo/bar"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 on extracting to stdout, got %d. Stderr: %s", code, stderr.String())
	}
	if stdout.String() != "hello bar" {
		t.Errorf("Expected 'hello bar', got %q", stdout.String())
	}

	// Case 4: Extract files to directory
	stdout.Reset()
	stderr.Reset()
	extractDir := filepath.Join(tempDir, "out")
	code = run([]string{"-d", "out", "test.zip"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 on extracting, got %d. Stderr: %s", code, stderr.String())
	}

	barContent, err := os.ReadFile(filepath.Join(extractDir, "foo/bar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(barContent) != "hello bar" {
		t.Errorf("Expected 'hello bar', got %q", string(barContent))
	}

	// Case 5: Path filter (extract only specific paths)
	os.RemoveAll(extractDir)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-d", "out", "test.zip", "foo/baz/qux"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	// qux should be extracted because it matches exact filter "foo/baz/qux"
	quxPath := filepath.Join(extractDir, "foo/baz/qux")
	if _, err := os.Stat(quxPath); err != nil {
		t.Error("Expected foo/baz/qux to be extracted based on filters")
	}
	// bar should NOT be extracted
	barPath := filepath.Join(extractDir, "foo/bar")
	if _, err := os.Stat(barPath); err == nil {
		t.Error("Expected foo/bar NOT to be extracted based on filters")
	}

	// Case 6: Overwrite check (no overwrite flag, existing file skips)
	if err := os.WriteFile(quxPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-d", "out", "test.zip", "foo/baz/qux"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatal(code)
	}
	quxContent, _ := os.ReadFile(quxPath)
	if string(quxContent) != "original" {
		t.Errorf("Expected original file to be skipped, got %q", string(quxContent))
	}

	// Case 7: Overwrite check (overwrite flag -o)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-o", "-d", "out", "test.zip", "foo/baz/qux"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatal(code)
	}
	quxContent, _ = os.ReadFile(quxPath)
	if string(quxContent) != "hello qux" {
		t.Errorf("Expected file to be overwritten, got %q", string(quxContent))
	}

	// Case 8: JSON mode on listing
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-l", "test.zip"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatal(code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"archive"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"foo/bar"`)) {
		t.Errorf("Expected JSON list formatting, got:\n%s", stdout.String())
	}
}

func TestUnzipDirectoryTraversalGuard(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "unzip-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create zip with directory traversal prefix
	zipBytes := createTestZip(t, map[string]string{
		"../../evil.txt": "evil",
	})
	zipPath := filepath.Join(tempDir, "evil.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-d", "out", "evil.zip"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected error code when directory traversal is blocked")
	}

	// Verify the file was not written outside the extract directory
	evilPath := filepath.Join(tempDir, "evil.txt")
	if _, err := os.Stat(evilPath); err == nil {
		t.Error("Security hazard! Directory traversal file created outside destination directory")
	}
}
