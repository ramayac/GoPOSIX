package tar

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarCreateExtract(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file inside a test directory
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)
	testFile := filepath.Join(srcDir, "test.txt")
	content := "hello tar"
	os.WriteFile(testFile, []byte(content), 0644)

	archiveFile := filepath.Join(tmpDir, "archive.tar")

	// Create archive
	var buf bytes.Buffer
	code := run([]string{"-c", "-f", archiveFile, "-C", tmpDir, "src"}, &buf)
	if code != 0 {
		t.Fatalf("create exit code %d", code)
	}

	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		t.Fatalf("archive file not created")
	}

	// Extract archive into a new directory
	destDir := filepath.Join(tmpDir, "dest")
	os.MkdirAll(destDir, 0755)

	var buf2 bytes.Buffer
	code = run([]string{"-x", "-f", archiveFile, "-C", destDir}, &buf2)
	if code != 0 {
		t.Fatalf("extract exit code %d", code)
	}

	// Check if file exists in the extracted location
	// The path inside the tar will be absolute since we passed an absolute path (srcDir)
	// So it gets extracted to destDir/tmp/...
	// This is standard tar behavior for absolute paths if not stripped.
	// Wait, filepath.Walk uses absolute paths if given absolute paths.
	// Let's check where it got extracted.
	// Actually, just find the file.
	found := false
	filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && info.Name() == "test.txt" {
			data, _ := os.ReadFile(path)
			if string(data) == content {
				found = true
			}
		}
		return nil
	})

	if !found {
		t.Errorf("extracted file not found or content mismatch")
	}
}

func TestTarList(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)
	testFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(testFile, []byte("hello tar"), 0644)

	archiveFile := filepath.Join(tmpDir, "archive.tar")
	var buf bytes.Buffer
	code := run([]string{"-c", "-f", archiveFile, "-C", tmpDir, "src"}, &buf)
	if code != 0 {
		t.Fatalf("create exit code %d", code)
	}

	var buf2 bytes.Buffer
	code = run([]string{"-t", "-f", archiveFile}, &buf2)
	if code != 0 {
		t.Fatalf("list exit code %d", code)
	}

	out := buf2.String()
	if !strings.Contains(out, "test.txt") {
		t.Errorf("list output missing filename: %s", out)
	}
}

func TestTarGzip(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)
	testFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(testFile, []byte(strings.Repeat("a", 1000)), 0644)

	archiveFile := filepath.Join(tmpDir, "archive.tar.gz")
	
	// Create with -z
	var buf bytes.Buffer
	code := run([]string{"-c", "-z", "-f", archiveFile, "-C", tmpDir, "src"}, &buf)
	if code != 0 {
		t.Fatalf("create exit code %d", code)
	}

	// Verify it's gzipped by checking magic number
	f, _ := os.Open(archiveFile)
	magic := make([]byte, 2)
	f.Read(magic)
	f.Close()
	if magic[0] != 0x1f || magic[1] != 0x8b {
		t.Errorf("file is not gzipped")
	}

	// List with -z
	var buf2 bytes.Buffer
	code = run([]string{"-t", "-z", "-f", archiveFile}, &buf2)
	if code != 0 {
		t.Fatalf("list exit code %d", code)
	}
}

func TestTarJSON(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)
	testFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(testFile, []byte("hello tar"), 0644)

	archiveFile := filepath.Join(tmpDir, "archive.tar")
	var buf bytes.Buffer
	code := run([]string{"--json", "-c", "-f", archiveFile, "-C", tmpDir, "src"}, &buf)
	if code != 0 {
		t.Fatalf("create exit code %d", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].([]interface{})
	if len(data) == 0 {
		t.Errorf("expected files in json output")
	}
}

func TestTarMissingArgs(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-c"}, &buf)
	if code != 1 {
		t.Errorf("should fail without -f")
	}

	var buf2 bytes.Buffer
	code = run([]string{"-f", "test.tar"}, &buf2)
	if code != 1 {
		t.Errorf("should fail without mode (-c, -x, -t)")
	}

	var buf3 bytes.Buffer
	code = run([]string{"-c", "-x", "-f", "test.tar"}, &buf3)
	if code != 1 {
		t.Errorf("should fail with multiple modes")
	}
}

// BusyBox hardening: extracting into a location where the original dir was read-only.
func TestTarExtractReadOnlyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tar-readonly-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "input_dir")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "input_file"), []byte("hello"), 0644)
	os.Chmod(srcDir, 0550) // read-only

	// Create archive
	archivePath := filepath.Join(tmpDir, "test.tar")
	createOut := &bytes.Buffer{}
	code := run(append([]string{"-c", "-f", archivePath, "-C", tmpDir}, "input_dir"), createOut)
	if code != 0 {
		t.Fatalf("tar create exited with %d, want 0: %s", code, createOut.String())
	}

	// Make dir writable and remove originals
	os.Chmod(srcDir, 0770)
	os.RemoveAll(srcDir)

	// Extract
	extractOut := &bytes.Buffer{}
	code = run([]string{"-x", "-f", archivePath, "-C", tmpDir}, extractOut)
	if code != 0 {
		t.Fatalf("tar extract exited with %d, want 0: %s", code, extractOut.String())
	}

	// Verify extracted file exists and is readable
	data, err := os.ReadFile(filepath.Join(srcDir, "input_file"))
	if err != nil {
		t.Fatalf("extracted file not readable: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("file content: %q, want 'hello'", string(data))
	}
}

// --- CLI and edge-case hardening ---

func tarTestDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	os.WriteFile(d+"/x.txt", []byte("x"), 0644)
	os.WriteFile(d+"/y.txt", []byte("yy"), 0644)
	return d
}

func TestTar_OldStyleFlags(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	// Old-style: "cf" instead of "-c -f"
	code := run([]string{"cf", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("old-style create exit %d", code)
	}
	if _, err := os.Stat(arc); os.IsNotExist(err) {
		t.Fatal("archive not created with old-style flags")
	}

	// Old-style list: "tf"
	var out2 bytes.Buffer
	code = run([]string{"tf", arc}, &out2)
	if code != 0 {
		t.Fatalf("old-style list exit %d", code)
	}
	if !strings.Contains(out2.String(), "x.txt") {
		t.Errorf("expected x.txt in list, got: %s", out2.String())
	}
}

func TestTar_OldStyleVerbose(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "tvf.tar")
	var out bytes.Buffer
	code := run([]string{"cvf", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("old-style verbose create exit %d", code)
	}
	// Verbose output should contain filename
	if !strings.Contains(out.String(), "x.txt") {
		t.Errorf("expected verbose output with filename, got: %s", out.String())
	}
}

func TestTar_CreateLongFlags(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "archive.tar")
	var out bytes.Buffer
	code := run([]string{"--create", "--file", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("--create --file exit %d", code)
	}
	if _, err := os.Stat(arc); os.IsNotExist(err) {
		t.Fatal("archive not created with long flags")
	}
}

func TestTar_ExtractLongFlags(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "arc.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, &bytes.Buffer{})

	dest := filepath.Join(dir, "out")
	os.Mkdir(dest, 0755)
	var out bytes.Buffer
	code := run([]string{"--extract", "--file", arc, "--directory", dest}, &out)
	if code != 0 {
		t.Fatalf("--extract exit %d", code)
	}
}

func TestTar_ListLongFlags(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "a.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, &bytes.Buffer{})

	var out bytes.Buffer
	code := run([]string{"--list", "--file", arc}, &out)
	if code != 0 {
		t.Fatalf("--list exit %d", code)
	}
	if !strings.Contains(out.String(), "x.txt") {
		t.Errorf("expected x.txt in --list output: %s", out.String())
	}
}

func TestTar_GzipLongFlag(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "a.tgz")
	var out bytes.Buffer
	code := run([]string{"--create", "--gzip", "--file", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("--gzip create exit %d", code)
	}
	f, _ := os.Open(arc)
	magic := make([]byte, 2)
	f.Read(magic)
	f.Close()
	if magic[0] != 0x1f || magic[1] != 0x8b {
		t.Error("file is not gzipped with --gzip")
	}
}

func TestTar_Verbose(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "v.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-v", "-f", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("verbose create exit %d", code)
	}
	if !strings.Contains(out.String(), "x.txt") {
		t.Errorf("verbose should print filenames, got: %s", out.String())
	}
}

func TestTar_ToStdout(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "o.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, &bytes.Buffer{})

	var out bytes.Buffer
	code := run([]string{"-x", "-O", "-f", arc}, &out)
	if code != 0 {
		t.Fatalf("-O extract exit %d", code)
	}
	if out.String() != "x" {
		t.Errorf("expected file content 'x', got %q", out.String())
	}
}

func TestTar_Overwrite(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "over.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, &bytes.Buffer{})

	// Change file content and extract with --overwrite
	os.WriteFile(dir+"/x.txt", []byte("new-x"), 0644)
	var out bytes.Buffer
	code := run([]string{"-x", "--overwrite", "-f", arc, "-C", dir}, &out)
	if code != 0 {
		t.Fatalf("--overwrite extract exit %d", code)
	}
	// Overwrite should have restored original content "x"
	data, _ := os.ReadFile(dir + "/x.txt")
	if string(data) != "x" {
		t.Errorf("expected overwritten content 'x', got %q", string(data))
	}
}

func TestTar_JSONList(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "list.json.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt", "y.txt"}, &bytes.Buffer{})

	var out bytes.Buffer
	code := run([]string{"--json", "-t", "-f", arc}, &out)
	if code != 0 {
		t.Fatalf("JSON list exit %d", code)
	}
	if !strings.Contains(out.String(), "\"name\"") {
		t.Errorf("expected JSON list output, got: %s", out.String())
	}
}

func TestTar_JSONCreate(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "create.json.tar")
	var out bytes.Buffer
	code := run([]string{"--json", "-c", "-f", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("JSON create exit %d", code)
	}
	if !strings.Contains(out.String(), "\"name\"") {
		t.Errorf("expected JSON create output, got: %s", out.String())
	}
}

func TestTar_BadCDir(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-c", "-f", "/tmp/test.tar", "-C", "/nonexistent/path/zzz", "dummy"}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for bad -C, got %d", code)
	}
}

func TestTar_StdinArchive(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "stdin.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, &bytes.Buffer{})

	// Read the archive and pipe it to stdin
	data, _ := os.ReadFile(arc)
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.Write(data)
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	dest := filepath.Join(dir, "stdout")
	os.Mkdir(dest, 0755)
	var out bytes.Buffer
	code := run([]string{"-x", "-f", "-", "-C", dest}, &out)
	if code != 0 {
		t.Fatalf("stdin extract exit %d", code)
	}
}

func TestTar_ExcludePattern(t *testing.T) {
	dir := tarTestDir(t)
	// Create archive with both files
	arc := filepath.Join(dir, "exclude.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt", "y.txt"}, &bytes.Buffer{})

	// Create exclude file
	excludeFile := filepath.Join(dir, "exclude.txt")
	os.WriteFile(excludeFile, []byte("y.txt\n"), 0644)

	// Test exclude with list mode (-t)
	var out bytes.Buffer
	code := run([]string{"-t", "-v", "-f", arc, "-X", excludeFile}, &out)
	if code != 0 {
		t.Fatalf("exclude list exit %d", code)
	}
	// y.txt should be excluded from listing
	if strings.Contains(out.String(), "y.txt") {
		t.Error("y.txt should be excluded from list by -X")
	}
	if !strings.Contains(out.String(), "x.txt") {
		t.Error("x.txt should still appear in list")
	}
}

func TestTar_ExtractExclude(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "ext.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt", "y.txt"}, &bytes.Buffer{})

	excludeFile := filepath.Join(dir, "ex.txt")
	os.WriteFile(excludeFile, []byte("y.txt\n"), 0644)

	dest := filepath.Join(dir, "dest")
	os.Mkdir(dest, 0755)
	var out bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", dest, "-X", excludeFile}, &out)
	if code != 0 {
		t.Fatalf("extract exclude exit %d", code)
	}
	// y.txt should not be extracted
	if _, err := os.Stat(filepath.Join(dest, "y.txt")); err == nil {
		t.Error("y.txt should be excluded from extract")
	}
	if _, err := os.Stat(filepath.Join(dest, "x.txt")); os.IsNotExist(err) {
		t.Error("x.txt should be extracted")
	}
}

func TestTar_MultipleFileArgs(t *testing.T) {
	dir := tarTestDir(t)
	os.WriteFile(dir+"/z.txt", []byte("zzz"), 0644)
	arc := filepath.Join(dir, "multi.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-v", "-f", arc, "-C", dir, "x.txt", "y.txt", "z.txt"}, &out)
	if code != 0 {
		t.Fatalf("multi-file create exit %d", code)
	}
	if !strings.Contains(out.String(), "x.txt") {
		t.Error("expected x.txt")
	}
	if !strings.Contains(out.String(), "z.txt") {
		t.Error("expected z.txt")
	}
}

func TestTar_OldStyleGzip(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "czf.tar.gz")
	var out bytes.Buffer
	code := run([]string{"czf", arc, "-C", dir, "x.txt"}, &out)
	if code != 0 {
		t.Fatalf("old-style czf exit %d", code)
	}
	f, _ := os.Open(arc)
	magic := make([]byte, 2)
	f.Read(magic)
	f.Close()
	if magic[0] != 0x1f || magic[1] != 0x8b {
		t.Error("not gzipped")
	}
}
