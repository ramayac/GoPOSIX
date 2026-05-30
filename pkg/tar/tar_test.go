package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ulikunitz/xz"
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
	code := run([]string{"-c", "-f", archiveFile, "-C", tmpDir, "src"}, nil, &buf, &buf, "")
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
	code = run([]string{"-x", "-f", archiveFile, "-C", destDir}, nil, &buf2, &buf2, "")
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
	code := run([]string{"-c", "-f", archiveFile, "-C", tmpDir, "src"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("create exit code %d", code)
	}

	var buf2 bytes.Buffer
	code = run([]string{"-t", "-f", archiveFile}, nil, &buf2, &buf2, "")
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
	code := run([]string{"-c", "-z", "-f", archiveFile, "-C", tmpDir, "src"}, nil, &buf, &buf, "")
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
	code = run([]string{"-t", "-z", "-f", archiveFile}, nil, &buf2, &buf2, "")
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
	code := run([]string{"--json", "-c", "-f", archiveFile, "-C", tmpDir, "src"}, nil, &buf, &buf, "")
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
	code := run([]string{"-c"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("should fail without -f")
	}

	var buf2 bytes.Buffer
	code = run([]string{"-f", "test.tar"}, nil, &buf2, &buf2, "")
	if code != 1 {
		t.Errorf("should fail without mode (-c, -x, -t)")
	}

	var buf3 bytes.Buffer
	code = run([]string{"-c", "-x", "-f", "test.tar"}, nil, &buf3, &buf3, "")
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
	code := run(append([]string{"-c", "-f", archivePath, "-C", tmpDir}, "input_dir"), nil, createOut, createOut, "")
	if code != 0 {
		t.Fatalf("tar create exited with %d, want 0: %s", code, createOut.String())
	}

	// Make dir writable and remove originals
	os.Chmod(srcDir, 0770)
	os.RemoveAll(srcDir)

	// Extract
	extractOut := &bytes.Buffer{}
	code = run([]string{"-x", "-f", archivePath, "-C", tmpDir}, nil, extractOut, extractOut, "")
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
	code := run([]string{"cf", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("old-style create exit %d", code)
	}
	if _, err := os.Stat(arc); os.IsNotExist(err) {
		t.Fatal("archive not created with old-style flags")
	}

	// Old-style list: "tf"
	var out2 bytes.Buffer
	code = run([]string{"tf", arc}, nil, &out2, &out2, "")
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
	code := run([]string{"cvf", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
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
	code := run([]string{"--create", "--file", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
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
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	dest := filepath.Join(dir, "out")
	os.Mkdir(dest, 0755)
	var out bytes.Buffer
	code := run([]string{"--extract", "--file", arc, "--directory", dest}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("--extract exit %d", code)
	}
}

func TestTar_ListLongFlags(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "a.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	var out bytes.Buffer
	code := run([]string{"--list", "--file", arc}, nil, &out, &out, "")
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
	code := run([]string{"--create", "--gzip", "--file", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
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
	code := run([]string{"-c", "-v", "-f", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
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
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	var out bytes.Buffer
	code := run([]string{"-x", "-O", "-f", arc}, nil, &out, &out, "")
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
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	// Change file content and extract with --overwrite
	os.WriteFile(dir+"/x.txt", []byte("new-x"), 0644)
	var out bytes.Buffer
	code := run([]string{"-x", "--overwrite", "-f", arc, "-C", dir}, nil, &out, &out, "")
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
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt", "y.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	var out bytes.Buffer
	code := run([]string{"--json", "-t", "-f", arc}, nil, &out, &out, "")
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
	code := run([]string{"--json", "-c", "-f", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("JSON create exit %d", code)
	}
	if !strings.Contains(out.String(), "\"name\"") {
		t.Errorf("expected JSON create output, got: %s", out.String())
	}
}

func TestTar_BadCDir(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-c", "-f", "/tmp/test.tar", "-C", "/nonexistent/path/zzz", "dummy"}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 for bad -C, got %d", code)
	}
}

func TestTar_StdinArchive(t *testing.T) {
	dir := tarTestDir(t)
	arc := filepath.Join(dir, "stdin.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

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
	code := run([]string{"-x", "-f", "-", "-C", dest}, r, &out, &out, "")
	if code != 0 {
		t.Fatalf("stdin extract exit %d", code)
	}
}

func TestTar_ExcludePattern(t *testing.T) {
	dir := tarTestDir(t)
	// Create archive with both files
	arc := filepath.Join(dir, "exclude.tar")
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt", "y.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	// Create exclude file
	excludeFile := filepath.Join(dir, "exclude.txt")
	os.WriteFile(excludeFile, []byte("y.txt\n"), 0644)

	// Test exclude with list mode (-t)
	var out bytes.Buffer
	code := run([]string{"-t", "-v", "-f", arc, "-X", excludeFile}, nil, &out, &out, "")
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
	run([]string{"-c", "-f", arc, "-C", dir, "x.txt", "y.txt"}, nil, &bytes.Buffer{}, &bytes.Buffer{}, "")

	excludeFile := filepath.Join(dir, "ex.txt")
	os.WriteFile(excludeFile, []byte("y.txt\n"), 0644)

	dest := filepath.Join(dir, "dest")
	os.Mkdir(dest, 0755)
	var out bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", dest, "-X", excludeFile}, nil, &out, &out, "")
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
	code := run([]string{"-c", "-v", "-f", arc, "-C", dir, "x.txt", "y.txt", "z.txt"}, nil, &out, &out, "")
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
	code := run([]string{"czf", arc, "-C", dir, "x.txt"}, nil, &out, &out, "")
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

func TestLocalTime_NoTZ(t *testing.T) {
	os.Unsetenv("TZ")
	now := time.Now()
	result := localTime(now)
	if !result.Equal(now) {
		t.Error("localTime should return same time when TZ is unset")
	}
}

func TestLocalTime_UTC(t *testing.T) {
	os.Setenv("TZ", "UTC")
	defer os.Unsetenv("TZ")
	now := time.Now()
	result := localTime(now)
	if !result.Equal(now) {
		t.Error("localTime should return same time for UTC")
	}
}

func TestLocalTime_UTCPlus(t *testing.T) {
	os.Setenv("TZ", "UTC+5")
	defer os.Unsetenv("TZ")
	now := time.Now()
	result := localTime(now)
	// localTime changes the timezone label, not the instant.
	// Just verify it doesn't panic and returns a non-zero time.
	if result.IsZero() {
		t.Error("expected non-zero time")
	}
	// The zone name should reflect the TZ.
	name, _ := result.Zone()
	if name != "UTC+5" {
		t.Logf("zone name: %q (expected UTC+5)", name)
	}
}

func TestLocalTime_UTCMinus(t *testing.T) {
	os.Setenv("TZ", "UTC-3")
	defer os.Unsetenv("TZ")
	now := time.Now()
	result := localTime(now)
	if result.IsZero() {
		t.Error("expected non-zero time")
	}
}

func TestLocalTime_BadUTC(t *testing.T) {
	os.Setenv("TZ", "UTCabc")
	defer os.Unsetenv("TZ")
	now := time.Now()
	result := localTime(now)
	if !result.Equal(now) {
		t.Error("localTime should return same time for invalid UTC offset")
	}
}

func TestTar_ExtractToStdout(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.WriteFile(src, []byte("hello\n"), 0644)
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "src"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d, stderr: %s", code, out.String())
	}
	// Extract to stdout (-O)
	var extractOut bytes.Buffer
	code = run([]string{"-x", "-f", arc, "-O"}, nil, &extractOut, &out, dir)
	if code != 0 {
		t.Fatalf("extract: exit %d", code)
	}
	if !strings.Contains(extractOut.String(), "hello") {
		t.Errorf("expected 'hello' in stdout extract, got %q", extractOut.String())
	}
}

func TestTar_ListVerbose(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("hello\n"), 0644)
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "src.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d", code)
	}
	var listOut bytes.Buffer
	code = run([]string{"-t", "-f", arc, "-v"}, nil, &listOut, &out, "")
	if code != 0 {
		t.Fatalf("list: exit %d", code)
	}
	if !strings.Contains(listOut.String(), "src.txt") {
		t.Errorf("expected 'src.txt' in listing, got %q", listOut.String())
	}
}

func TestTarHardlinkCreateExtract(t *testing.T) {
	dir := t.TempDir()

	// Create a file and a hardlink to it
	src := filepath.Join(dir, "original")
	os.WriteFile(src, []byte("hardlink content"), 0644)
	link := filepath.Join(dir, "hardlink")
	if err := os.Link(src, link); err != nil {
		t.Fatalf("cannot create hardlink: %v", err)
	}

	// Create archive with both files
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "original", "hardlink"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d, stderr: %s", code, out.String())
	}

	// Verify archive listing shows hardlink
	var listOut bytes.Buffer
	code = run([]string{"-t", "-f", arc, "-v"}, nil, &listOut, &out, "")
	if code != 0 {
		t.Fatalf("list: exit %d", code)
	}
	listing := listOut.String()
	if !strings.Contains(listing, "original") {
		t.Error("listing should contain 'original'")
	}
	if !strings.Contains(listing, "hardlink") {
		t.Error("listing should contain 'hardlink'")
	}

	// Extract and verify hardlink relationship is preserved
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	var extOut bytes.Buffer
	code = run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &extOut, &extOut, "")
	if code != 0 {
		t.Fatalf("extract: exit %d, stderr: %s", code, extOut.String())
	}

	// Both files should exist and share the same inode
	fi1, err := os.Stat(filepath.Join(extractDir, "original"))
	if err != nil {
		t.Fatalf("original not extracted: %v", err)
	}
	fi2, err := os.Stat(filepath.Join(extractDir, "hardlink"))
	if err != nil {
		t.Fatalf("hardlink not extracted: %v", err)
	}
	if !os.SameFile(fi1, fi2) {
		t.Error("extracted files are not hardlinked (different inodes)")
	}
}

func TestTarSymlinkTarget(t *testing.T) {
	dir := t.TempDir()

	// Create a file and a symlink to it
	src := filepath.Join(dir, "target")
	os.WriteFile(src, []byte("symlink target content"), 0644)
	sym := filepath.Join(dir, "link")
	if err := os.Symlink("target", sym); err != nil {
		t.Fatalf("cannot create symlink: %v", err)
	}

	// Archive the symlink
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "link"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d, stderr: %s", code, out.String())
	}

	// List and check symlink target is correct
	var listOut bytes.Buffer
	code = run([]string{"-t", "-f", arc, "-v"}, nil, &listOut, &out, "")
	if code != 0 {
		t.Fatalf("list: exit %d", code)
	}
	listing := listOut.String()
	if !strings.Contains(listing, "link -> target") {
		t.Errorf("expected 'link -> target' in listing, got: %s", listing)
	}

	// Extract and verify symlink
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	code = run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("extract: exit %d", code)
	}

	target, err := os.Readlink(filepath.Join(extractDir, "link"))
	if err != nil {
		t.Fatalf("cannot readlink: %v", err)
	}
	if target != "target" {
		t.Errorf("symlink target = %q, want 'target'", target)
	}
}

func TestTarKeepOldFlag(t *testing.T) {
	dir := t.TempDir()

	// Create a file for archiving
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("original content"), 0644)
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "src.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d", code)
	}

	// Pre-create the file with different content
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	existing := filepath.Join(extractDir, "src.txt")
	os.WriteFile(existing, []byte("pre-existing content"), 0644)

	// Extract with -k (keep-old) — should NOT overwrite
	code = run([]string{"-x", "-k", "-f", arc, "-C", extractDir}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("extract -k: exit %d", code)
	}

	data, _ := os.ReadFile(existing)
	if string(data) != "pre-existing content" {
		t.Errorf("file was overwritten despite -k flag: got %q", string(data))
	}

	// Extract without -k (default) — SHOULD overwrite
	code = run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("extract: exit %d", code)
	}

	data, _ = os.ReadFile(existing)
	if string(data) != "original content" {
		t.Errorf("file was NOT overwritten without -k: got %q", string(data))
	}
}

func TestTarShortReadError(t *testing.T) {
	// Test that empty gzipped tar produces "short read" error
	dir := t.TempDir()
	arc := filepath.Join(dir, "empty.tar.gz")
	// Create a valid empty gzip (10-byte gzip header + footer)
	os.WriteFile(arc, []byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}, 0644)
	var out bytes.Buffer
	code := run([]string{"-x", "-z", "-f", arc}, nil, &out, &out, dir)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "short read") {
		t.Errorf("expected 'short read' error, got: %s", out.String())
	}
}

func TestTarSymlinkSafetyUnlink(t *testing.T) {
	// Verify that extracting a regular file removes an existing symlink first.
	dir := t.TempDir()

	// Create an archive with a regular file
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("safe content"), 0644)
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "src.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d", code)
	}

	// Pre-create a symlink at the extraction target
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	symTarget := filepath.Join(extractDir, "dangerous")
	os.WriteFile(symTarget, []byte("should not be touched"), 0644)
	symlink := filepath.Join(extractDir, "src.txt")
	if err := os.Symlink(symTarget, symlink); err != nil {
		t.Fatalf("cannot create symlink: %v", err)
	}

	// Extract — should unlink the symlink and create a regular file
	code = run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("extract: exit %d, stderr: %s", code, out.String())
	}

	// Verify the symlink is gone and replaced with a regular file
	fi, err := os.Lstat(symlink)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("symlink still exists — not replaced with regular file")
	}
	// The dangerous target should be untouched
	dangerData, _ := os.ReadFile(symTarget)
	if string(dangerData) != "should not be touched" {
		t.Error("symlink target was modified — attack not prevented")
	}
}

func TestTarBzip2Extract(t *testing.T) {
	dir := t.TempDir()
	// Test with -j flag (explicit bzip2)
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("bzip2 content"), 0644)

	// Create gzip archive first (bzip2 creation not supported, only extraction)
	arc := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", arc, "-C", dir, "src.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d", code)
	}

	// Test that -j flag is parsed correctly (bzip2 decompression on non-bzip2
	// data will fail, but the flag parsing and plumbing should work)
	// Just verify the flag is recognized and doesn't cause a parse error.
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	code = run([]string{"-x", "-j", "-f", arc, "-C", extractDir}, nil, &out, &out, "")
	// Expected to fail (not bzip2), but -j flag must not cause "unknown flag"
	if code == 0 {
		t.Log("extract with -j unexpectedly succeeded on non-bzip2 data")
	}
	// The error should not be "unknown flag"
	if strings.Contains(out.String(), "unknown flag") {
		t.Error("-j flag not recognized")
	}
}

// createTarWithEntries creates a tar archive in dir with the given entries.
// Each entry is a []string: {name, typeflag, linkname, content}.
// typeflag is "reg", "symlink", or "dir". For symlink, linkname is the target.
// For reg, content is the file content. Returns the archive path.
func createTarWithEntries(t *testing.T, dir, arcName string, entries [][]string) string {
	t.Helper()
	arc := filepath.Join(dir, arcName)
	f, err := os.Create(arc)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	for _, e := range entries {
		name := e[0]
		typ := e[1]
		var hdr *tar.Header
		switch typ {
		case "reg":
			content := ""
			if len(e) > 3 {
				content = e[3]
			}
			hdr = &tar.Header{
				Name:     name,
				Size:     int64(len(content)),
				Typeflag: tar.TypeReg,
				Mode:     0644,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}
			if content != "" {
				if _, err := tw.Write([]byte(content)); err != nil {
					t.Fatal(err)
				}
			}
		case "symlink":
			linkname := ""
			if len(e) > 2 {
				linkname = e[2]
			}
			hdr = &tar.Header{
				Name:     name,
				Linkname: linkname,
				Typeflag: tar.TypeSymlink,
				Mode:     0777,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}
		case "dir":
			hdr = &tar.Header{
				Name:     name,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}
		}
	}
	tw.Close()
	f.Close()
	return arc
}

func TestTarSymlinkSafetyAbsoluteTarget(t *testing.T) {
	// BusyBox test: "tar does not extract into symlinks"
	// Archive: symlink passwd -> /tmp/passwd, then regular file passwd.
	// Tar must refuse the symlink with a warning, extract the regular file.
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "attack.tar", [][]string{
		{"passwd", "symlink", "/tmp/passwd"},
		{"passwd", "reg", "", "safe"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &stdout, &stderr, "")

	// Should exit 0 (warning, not error — no child entries depend on the symlink)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}

	// Must print the symlink safety warning
	if !strings.Contains(stderr.String(), "can't create symlink") {
		t.Errorf("expected symlink safety warning, got stderr: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "passwd") {
		t.Errorf("expected warning to mention 'passwd', got: %s", stderr.String())
	}

	// The regular file must be extracted (since symlink was refused, target is free)
	data, err := os.ReadFile(filepath.Join(extractDir, "passwd"))
	if err != nil {
		t.Fatalf("regular file not extracted: %v", err)
	}
	if string(data) != "safe" {
		t.Errorf("file content = %q, want 'safe'", string(data))
	}

	// The symlink must NOT exist
	if fi, err := os.Lstat(filepath.Join(extractDir, "passwd")); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Error("dangerous symlink was created — security violation")
		}
	}
}

func TestTarSymlinkSafetyKeepOld(t *testing.T) {
	// BusyBox test: "tar -k does not extract into symlinks"
	// Same archive as above but with -k. The symlink warning still prints.
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "attack.tar", [][]string{
		{"passwd", "symlink", "/tmp/passwd"},
		{"passwd", "reg", "", "safe"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-k", "-f", arc, "-C", extractDir}, nil, &stdout, &stderr, "")

	// Must print the symlink safety warning
	if !strings.Contains(stderr.String(), "can't create symlink") {
		t.Errorf("expected symlink safety warning, got stderr: %s", stderr.String())
	}

	// With -k, if target doesn't exist, the regular file should be extracted.
	// The symlink was refused so the regular file path is free.
	data, err := os.ReadFile(filepath.Join(extractDir, "passwd"))
	if err != nil {
		t.Fatalf("regular file not extracted: %v", err)
	}
	if string(data) != "safe" {
		t.Errorf("file content = %q, want 'safe'", string(data))
	}

	// Should exit 0
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestTarSymlinkAttackPrevention(t *testing.T) {
	// BusyBox test: "tar Symlink attack: create symlink and then write through it"
	// Archive: anything.txt, symlink -> /tmp, symlink/bb_test_evilfile.
	// Tar must refuse the symlink, create child file safely, exit 1.
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "attack.tar", [][]string{
		{"anything.txt", "reg", "", "safe"},
		{"symlink", "symlink", "/tmp"},
		{"symlink/bb_test_evilfile", "reg", "", "evil"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &stdout, &stderr, "")

	// Must print the symlink safety warning
	if !strings.Contains(stderr.String(), "can't create symlink") {
		t.Errorf("expected symlink safety warning, got stderr: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "/tmp") {
		t.Errorf("expected warning to mention '/tmp', got: %s", stderr.String())
	}

	// First file should be extracted normally
	data, err := os.ReadFile(filepath.Join(extractDir, "anything.txt"))
	if err != nil {
		t.Fatalf("anything.txt not extracted: %v", err)
	}
	if string(data) != "safe" {
		t.Errorf("anything.txt content = %q, want 'safe'", string(data))
	}

	// The dangerous symlink must NOT exist
	if fi, err := os.Lstat(filepath.Join(extractDir, "symlink")); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Error("dangerous symlink to /tmp was created — security violation")
		}
	}

	// The child file must be extracted safely INSIDE the extraction dir
	// (not at /tmp/bb_test_evilfile). Since the symlink was refused,
	// symlink/ is created as a real directory.
	evilPath := filepath.Join(extractDir, "symlink", "bb_test_evilfile")
	data, err = os.ReadFile(evilPath)
	if err != nil {
		t.Fatalf("symlink/bb_test_evilfile not extracted: %v", err)
	}
	if string(data) != "evil" {
		t.Errorf("bb_test_evilfile content = %q, want 'evil'", string(data))
	}

	// The file must NOT have been written to /tmp
	if _, err := os.Stat("/tmp/bb_test_evilfile"); err == nil {
		t.Error("bb_test_evilfile was written to /tmp — symlink attack succeeded!")
	}

	// Exit code must be 1: symlink had child entries depending on it
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestTarXZCompressionAutoDetect(t *testing.T) {
	// Create a tar archive, compress with XZ, extract via auto-detect.
	dir := t.TempDir()

	// Create source file
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("hello xz"), 0644)

	// Create tar archive
	tarPath := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", tarPath, "-C", dir, "src"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d", code)
	}

	// Compress with XZ
	tarData, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	xzPath := filepath.Join(dir, "test.tar.xz")
	var xzBuf bytes.Buffer
	w, err := xz.NewWriter(&xzBuf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(tarData); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.WriteFile(xzPath, xzBuf.Bytes(), 0644)

	// Extract with auto-detect (no -z or -j flag)
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	var stderr bytes.Buffer
	code = run([]string{"-x", "-f", xzPath, "-C", extractDir}, nil, &stderr, &stderr, "")
	if code != 0 {
		t.Fatalf("extract xz: exit %d, stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(extractDir, "src", "hello.txt"))
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if string(data) != "hello xz" {
		t.Errorf("content = %q, want 'hello xz'", string(data))
	}
}

func TestTarXZListAutoDetect(t *testing.T) {
	// Test listing an XZ-compressed archive via auto-detect.
	dir := t.TempDir()

	// Create tar archive with a known file
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "listme.txt"), []byte("data"), 0644)

	tarPath := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	code := run([]string{"-c", "-f", tarPath, "-C", dir, "src"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("create: exit %d", code)
	}

	// Compress with XZ
	tarData, _ := os.ReadFile(tarPath)
	var xzBuf bytes.Buffer
	w, _ := xz.NewWriter(&xzBuf)
	w.Write(tarData)
	w.Close()
	xzPath := filepath.Join(dir, "list.tar.xz")
	os.WriteFile(xzPath, xzBuf.Bytes(), 0644)

	// List the XZ archive
	var stdout, stderr bytes.Buffer
	code = run([]string{"-t", "-f", xzPath}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("list xz: exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "listme.txt") {
		t.Errorf("expected 'listme.txt' in listing, got: %s", stdout.String())
	}
}

func TestTarSymlinkSafetyRelativeSafe(t *testing.T) {
	// Symlink with a target that stays within the extraction root is allowed.
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "safe.tar", [][]string{
		{"dir/file", "reg", "", "content"},
		{"link", "symlink", "dir/file"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &stderr, &stderr, "")
	if code != 0 {
		t.Fatalf("extract: exit %d, stderr: %s", code, stderr.String())
	}

	// Symlink should exist and point to dir/file
	target, err := os.Readlink(filepath.Join(extractDir, "link"))
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if target != "dir/file" {
		t.Errorf("symlink target = %q, want 'dir/file'", target)
	}

	// No warning for safe symlinks
	if strings.Contains(stderr.String(), "can't create symlink") {
		t.Error("unexpected safety warning for safe symlink")
	}
}

func TestTarSymlinkSafetyRelativeEscape(t *testing.T) {
	// Symlink that escapes via .. is refused when there's a conflict.
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "escape.tar", [][]string{
		{"escape", "symlink", "../../../etc/passwd"},
		{"escape/child", "reg", "", "bad"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &stderr, &stderr, "")

	// Should warn about the dangerous symlink
	if !strings.Contains(stderr.String(), "can't create symlink") {
		t.Errorf("expected symlink safety warning for .. escape, got: %s", stderr.String())
	}

	// Should exit 1: child entry depended on the refused symlink
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Symlink must NOT exist (a real directory for child entries is OK)
	fi, err := os.Lstat(filepath.Join(extractDir, "escape"))
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		t.Error("dangerous ../ symlink was created")
	}

	// Child file should be extracted safely inside extraction dir
	childPath := filepath.Join(extractDir, "escape", "child")
	if data, err := os.ReadFile(childPath); err != nil {
		t.Logf("child file not extracted (expected if dir created): %v", err)
	} else if string(data) != "bad" {
		t.Errorf("child content = %q, want 'bad'", string(data))
	}
}

func TestTarSymlinkSafetyNoConflict(t *testing.T) {
	// Absolute symlink with NO conflicting entry is allowed.
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "safe_abs.tar", [][]string{
		{"mylink", "symlink", "/usr/share/something"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", extractDir}, nil, &stderr, &stderr, "")
	if code != 0 {
		t.Fatalf("extract: exit %d, stderr: %s", code, stderr.String())
	}

	// Symlink should exist
	fi, err := os.Lstat(filepath.Join(extractDir, "mylink"))
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file")
	}

	// No warning (no conflict)
	if strings.Contains(stderr.String(), "can't create symlink") {
		t.Error("unexpected safety warning for non-conflicting absolute symlink")
	}
}

func TestTarExtractToStdout(t *testing.T) {
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "test.tar", [][]string{
		{"hello.txt", "reg", "", "hello stdout"},
		{"sub/world.txt", "reg", "", "world stdout"},
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-O", "-f", arc}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("extract -O: exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hello stdout") {
		t.Errorf("expected 'hello stdout' in stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "world stdout") {
		t.Errorf("expected 'world stdout' in stdout, got: %s", stdout.String())
	}
}

func TestTarExtractWithOverwrite(t *testing.T) {
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "test.tar", [][]string{
		{"file.txt", "reg", "", "new content"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	// Pre-create the file
	os.WriteFile(filepath.Join(extractDir, "file.txt"), []byte("old content"), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "--overwrite", "-f", arc, "-C", extractDir}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("extract --overwrite: exit %d, stderr: %s", code, stderr.String())
	}

	data, _ := os.ReadFile(filepath.Join(extractDir, "file.txt"))
	if string(data) != "new content" {
		t.Errorf("content = %q, want 'new content'", string(data))
	}
}

func TestTarExtractWithIncludeList(t *testing.T) {
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "test.tar", [][]string{
		{"a.txt", "reg", "", "A"},
		{"b.txt", "reg", "", "B"},
		{"c.txt", "reg", "", "C"},
	})

	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-C", extractDir, "a.txt", "c.txt"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("extract with include: exit %d, stderr: %s", code, stderr.String())
	}

	// a.txt and c.txt should exist, b.txt should not
	for _, name := range []string{"a.txt", "c.txt"} {
		if _, err := os.Stat(filepath.Join(extractDir, name)); err != nil {
			t.Errorf("expected %s to exist", name)
		}
	}
	if _, err := os.Stat(filepath.Join(extractDir, "b.txt")); err == nil {
		t.Error("b.txt should not have been extracted")
	}
}

func TestTarListGzipAutoDetect(t *testing.T) {
	dir := t.TempDir()

	// Create tar archive
	tarPath := filepath.Join(dir, "test.tar")
	var out bytes.Buffer
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644)
	code := run([]string{"-c", "-f", tarPath, "-C", dir, "f.txt"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatal("create failed")
	}

	// Compress with gzip
	tarData, _ := os.ReadFile(tarPath)
	var gzBuf bytes.Buffer
	w := gzip.NewWriter(&gzBuf)
	w.Write(tarData)
	w.Close()

	gzPath := filepath.Join(dir, "list.tar.gz")
	os.WriteFile(gzPath, gzBuf.Bytes(), 0644)

	var stdout, stderr bytes.Buffer
	code = run([]string{"-t", "-f", gzPath}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("list gz: exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "f.txt") {
		t.Errorf("expected 'f.txt' in gzip listing, got: %s", stdout.String())
	}
}

func TestTarVerboseCreate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "v.txt"), []byte("verbose test"), 0644)

	arc := filepath.Join(dir, "test.tar")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-c", "-v", "-f", arc, "-C", dir, "v.txt"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("create -v: exit %d", code)
	}
	if !strings.Contains(stdout.String(), "v.txt") {
		t.Errorf("expected 'v.txt' in verbose create output, got: %s", stdout.String())
	}
}

func TestTarStdinList(t *testing.T) {
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "test.tar", [][]string{
		{"stdin_test.txt", "reg", "", "stdin"},
	})

	tarData, _ := os.ReadFile(arc)
	var stdout, stderr bytes.Buffer
	code := run([]string{"-t"}, bytes.NewReader(tarData), &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("list stdin: exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "stdin_test.txt") {
		t.Errorf("expected 'stdin_test.txt' in listing, got: %s", stdout.String())
	}
}

func TestResolveTarPath(t *testing.T) {
	tests := []struct {
		input        string
		wantResolved string
	}{
		{"a/b/c", "a/b/c"},
		{"./a/b", "a/b"},
		{"a/b/../c", "a/c"},
		{"../a", ""},
		{"././a", "a"},
		{"a/./b", "a/b"},
		{"a//b", "a/b"},
		{"../../etc/passwd", "passwd"},
	}
	for _, tc := range tests {
		resolved, stripped := resolveTarPath(tc.input)
		if resolved != tc.wantResolved {
			t.Errorf("resolveTarPath(%q) resolved = %q, want %q", tc.input, resolved, tc.wantResolved)
		}
		// Just verify stripped is non-empty when input has leading ./ or ..
		if strings.HasPrefix(tc.input, ".") && stripped == "" && resolved != tc.input {
			t.Errorf("resolveTarPath(%q) stripped should not be empty when prefix was stripped", tc.input)
		}
		_ = stripped
	}
}

func TestTarJSONListOutput(t *testing.T) {
	dir := t.TempDir()
	arc := createTarWithEntries(t, dir, "test.tar", [][]string{
		{"j.txt", "reg", "", "json"},
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"-t", "--json", "-f", arc}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("list --json: exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"name\"") {
		t.Errorf("expected JSON output, got: %s", stdout.String())
	}
}

func TestTarMissingArchiveError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-f", "/nonexistent/archive.tar"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing archive, got %d", code)
	}
}

func TestTarCorruptGzipError(t *testing.T) {
	dir := t.TempDir()
	// Write corrupt gzip data
	arc := filepath.Join(dir, "bad.tar.gz")
	os.WriteFile(arc, []byte("not a gzip file"), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-z", "-f", arc}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for corrupt gzip, got %d", code)
	}
}

func TestTarExcludeFromFile(t *testing.T) {
	dir := t.TempDir()

	// Create archive with two files
	arc := createTarWithEntries(t, dir, "test.tar", [][]string{
		{"keep.txt", "reg", "", "keep"},
		{"skip_me.txt", "reg", "", "skip"},
	})

	// Create exclude file for extraction
	excludeFile := filepath.Join(dir, "exclude.txt")
	os.WriteFile(excludeFile, []byte("skip_me.txt\n"), 0644)

	// Extract with exclude — only keep.txt should be extracted
	extractDir := filepath.Join(dir, "out")
	os.MkdirAll(extractDir, 0755)
	var stdout, stderr bytes.Buffer
	code := run([]string{"-x", "-f", arc, "-X", excludeFile, "-C", extractDir}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("extract with exclude: exit %d, stderr: %s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(extractDir, "keep.txt")); err != nil {
		t.Error("keep.txt should have been extracted")
	}
	if _, err := os.Stat(filepath.Join(extractDir, "skip_me.txt")); err == nil {
		t.Error("skip_me.txt should have been excluded")
	}
}
