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

func TestScanCorruptedZip(t *testing.T) {
	// Test with a minimal local file header (PK\x03\x04) with a filename.
	createLocalHeader := func(name string) []byte {
		buf := make([]byte, 30+len(name))
		buf[0] = 'P'
		buf[1] = 'K'
		buf[2] = 0x03
		buf[3] = 0x04
		// filename length at offset 26 (uint16 LE)
		buf[26] = byte(len(name))
		buf[27] = byte(len(name) >> 8)
		copy(buf[30:], name)
		return buf
	}

	t.Run("normal filename", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.zip")
		os.WriteFile(path, createLocalHeader("hello.txt"), 0644)
		firstFile, slashWarn := scanCorruptedZip(path)
		if firstFile != "hello.txt" {
			t.Errorf("expected 'hello.txt', got %q", firstFile)
		}
		if slashWarn {
			t.Error("unexpected slash warning")
		}
	})

	t.Run("slash prefix", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "slash.zip")
		os.WriteFile(path, createLocalHeader("/tmp/passwd"), 0644)
		firstFile, slashWarn := scanCorruptedZip(path)
		if firstFile != "tmp/passwd" {
			t.Errorf("expected 'tmp/passwd' (stripped), got %q", firstFile)
		}
		if !slashWarn {
			t.Error("expected slash warning")
		}
	})

	t.Run("corrupted sig byte", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "corrupt.zip")
		buf := createLocalHeader("data.bin")
		buf[3] = 0x00 // corrupted signature
		os.WriteFile(path, buf, 0644)
		firstFile, slashWarn := scanCorruptedZip(path)
		if firstFile != "data.bin" {
			t.Errorf("expected 'data.bin', got %q", firstFile)
		}
		if slashWarn {
			t.Error("unexpected slash warning")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		firstFile, slashWarn := scanCorruptedZip("/nonexistent/zip.zip")
		if firstFile != "" {
			t.Errorf("expected empty, got %q", firstFile)
		}
		if slashWarn {
			t.Error("unexpected slash warning")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.zip")
		os.WriteFile(path, []byte{}, 0644)
		firstFile, slashWarn := scanCorruptedZip(path)
		if firstFile != "" {
			t.Errorf("expected empty, got %q", firstFile)
		}
		if slashWarn {
			t.Error("unexpected slash warning")
		}
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"", ""},
		{"hello.txt", "hello.txt"},
		{"normal_file-123", "normal_file-123"},
		{"a\x00b", "a?b"},                  // NUL → ?
		{"tab\there", "tab?here"},          // TAB → ?
		{"new\nline", "new?line"},          // NL → ?
		{"esc\x1bdata", "esc?data"},        // ESC → ?
		{"del\x7fchar", "del?char"},        // DEL → ?
		{"latin\xbdchar", "latin\xbdchar"}, // extended ASCII preserved
		{"utf\xc3\xa9", "utf\xc3\xa9"},     // UTF-8 preserved
		{"mixed\x01ext\xbd", "mixed?ext\xbd"},
	}
	for _, tc := range tests {
		got := sanitizeFilename(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestUnzipCorruptedArchiveError(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("completely invalid zip", func(t *testing.T) {
		path := filepath.Join(tempDir, "bad.zip")
		os.WriteFile(path, []byte("not a zip file at all"), 0644)
		var stdout, stderr bytes.Buffer
		code := run([]string{path}, nil, &stdout, &stderr, tempDir)
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errOut := stderr.String()
		if !strings.Contains(errOut, "corrupted data") {
			t.Errorf("expected 'corrupted data' in stderr, got: %s", errOut)
		}
		if !strings.Contains(errOut, "inflate error") {
			t.Errorf("expected 'inflate error' in stderr, got: %s", errOut)
		}
	})

	t.Run("zero entries suspicious zip", func(t *testing.T) {
		// Create a zip that Go's archive/zip opens but finds zero entries (>128 bytes)
		// An empty valid zip with just EOCD is 22 bytes; a larger "empty" zip is suspicious.
		path := filepath.Join(tempDir, "suspicious.zip")
		// Write PK magic + 200 bytes of zeros (looks like a zip but isn't)
		buf := make([]byte, 200)
		buf[0] = 'P'
		buf[1] = 'K'
		os.WriteFile(path, buf, 0644)
		var stdout, stderr bytes.Buffer
		code := run([]string{path}, nil, &stdout, &stderr, tempDir)
		if code != 1 {
			t.Errorf("expected exit code 1 for suspicious empty zip, got %d", code)
		}
	})

	t.Run("corrupted deflate data", func(t *testing.T) {
		// Create a valid zip with store method, then corrupt the compressed data.
		path := filepath.Join(tempDir, "corrupt_data.zip")
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		fh := &zip.FileHeader{Name: "data.bin", Method: zip.Deflate}
		f, _ := w.CreateHeader(fh)
		f.Write([]byte("original data"))
		w.Close()
		// Corrupt the compressed data (last 10 bytes)
		raw := buf.Bytes()
		for i := len(raw) - 10; i < len(raw); i++ {
			raw[i] ^= 0xFF
		}
		os.WriteFile(path, raw, 0644)
		var stdout, stderr bytes.Buffer
		code := run([]string{path}, nil, &stdout, &stderr, tempDir)
		if code != 1 {
			t.Errorf("expected exit code 1 for corrupted deflate, got %d", code)
		}
		errOut := stderr.String()
		if !strings.Contains(errOut, "corrupted data") {
			t.Errorf("expected 'corrupted data' in stderr, got: %s", errOut)
		}
	})
}
