package cpio

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cavaliergopher/cpio"
)

// createTestArchive builds a cpio archive in memory.
func createTestArchive(t *testing.T, members map[string]string) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	w := cpio.NewWriter(buf)

	for name, content := range members {
		hdr := &cpio.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: cpio.TypeReg | 0644,
		}
		if err := w.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf
}

func TestCpioListMode(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"file1.txt": "hello\n",
		"file2.txt": "world\n",
	})

	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-it"}, archive, &stdout, &stderr, "/tmp")
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "file1.txt") || !strings.Contains(out, "file2.txt") {
		t.Errorf("expected both filenames in list output, got: %s", out)
	}
}

func TestCpioListJSON(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"a.txt": "content\n",
	})

	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-it", "--json"}, archive, &stdout, &stderr, "/tmp")
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"name"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCpioExtract(t *testing.T) {
	dir := t.TempDir()
	archive := createTestArchive(t, map[string]string{
		"extract_me.txt": "extracted!\n",
	})

	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-id"}, archive, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(dir, "extract_me.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "extracted!\n" {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestCpioCreate(t *testing.T) {
	dir := t.TempDir()

	// Create some files
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bbb\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Provide file names on stdin
	names := "a.txt\nb.txt\n"
	stdinR := strings.NewReader(names)

	var archiveOut, stderr bytes.Buffer
	// We call cpioCreate directly to test it
	rc := cpioCreate(&archiveOut, stdinR, false, false, &bytes.Buffer{}, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if archiveOut.Len() == 0 {
		t.Error("expected non-empty archive output")
	}
}

func TestCpioCreateViaRun(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello!\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("hello.txt\n")
	var archiveOut, stderr bytes.Buffer
	rc := cpioRun([]string{"-o"}, stdinR, &archiveOut, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if archiveOut.Len() == 0 {
		t.Error("expected non-empty archive output")
	}
}

func TestCpioNoMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-v"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc when no mode specified")
	}
}

func TestCpioPassThrough(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "pass.txt"), []byte("passed!\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("pass.txt\n")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-p", dstDir}, stdinR, &stdout, &stderr, srcDir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "pass.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "passed!\n" {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestCpioPassThroughJSON(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "json.txt"), []byte("json!\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("json.txt\n")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-p", "--json", dstDir}, stdinR, &stdout, &stderr, srcDir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"name"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestCpioPassThroughNoDest(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-p"}, strings.NewReader(""), &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc when no destination given for -p")
	}
}

func TestCpioFileFlag(t *testing.T) {
	dir := t.TempDir()

	// Create a file to archive
	if err := os.WriteFile(filepath.Join(dir, "ff.txt"), []byte("fileflag\n"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(dir, "out.cpio")

	// Create archive using -F flag
	stdinR := strings.NewReader("ff.txt\n")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-o", "-F", archivePath}, stdinR, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("create failed: %d: %s", rc, stderr.String())
	}

	// List archive using -F flag
	var listOut, listErr bytes.Buffer
	rc = cpioRun([]string{"-it", "-F", archivePath}, nil, &listOut, &listErr, dir)
	if rc != 0 {
		t.Fatalf("list failed: %d: %s", rc, listErr.String())
	}
	if !strings.Contains(listOut.String(), "ff.txt") {
		t.Errorf("expected ff.txt in archive listing, got: %s", listOut.String())
	}
}

func TestCpioCreateJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "jc.txt"), []byte("jcontent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("jc.txt\n")
	var archiveOut, stderr bytes.Buffer
	var jsonOut bytes.Buffer

	// Create archive with JSON mode (JSON goes to stdout, archive to F flag file)
	archivePath := filepath.Join(dir, "jc.cpio")
	rc := cpioRun([]string{"-o", "--json", "-F", archivePath}, stdinR, &jsonOut, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	out := jsonOut.String()
	if !strings.Contains(out, `"name"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
	_ = archiveOut
}

func TestCpioExtractFilter(t *testing.T) {
	dir := t.TempDir()
	archive := createTestArchive(t, map[string]string{
		"want.txt":   "want this\n",
		"unwant.txt": "don't want\n",
	})

	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-id", "want.txt"}, archive, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	if _, err := os.ReadFile(filepath.Join(dir, "want.txt")); err != nil {
		t.Error("expected want.txt to be extracted")
	}
	if _, err := os.Stat(filepath.Join(dir, "unwant.txt")); err == nil {
		t.Error("unwant.txt should NOT be extracted")
	}
}

func TestCpioExtractVerbose(t *testing.T) {
	dir := t.TempDir()
	archive := createTestArchive(t, map[string]string{
		"verbose_extract.txt": "verbose\n",
	})

	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-idv"}, archive, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "verbose_extract.txt") {
		t.Errorf("expected verbose_extract.txt in verbose output, got: %s", out)
	}
}

func TestCpioCreateVerbose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cv.txt"), []byte("cv content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("cv.txt\n")
	var archiveOut, stderr bytes.Buffer
	rc := cpioRun([]string{"-ov"}, stdinR, &archiveOut, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	// verbose output goes to stderr for cpio -o
	if !strings.Contains(stderr.String(), "cv.txt") {
		t.Errorf("expected cv.txt in verbose stderr output, got: %s", stderr.String())
	}
}

func TestCpioPassThroughVerbose(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "pv.txt"), []byte("pass verbose\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("pv.txt\n")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-pv", dstDir}, stdinR, &stdout, &stderr, srcDir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stderr.String(), "pv.txt") {
		t.Errorf("expected verbose stderr output, got: %s", stderr.String())
	}
}

func TestCpioExtractBadArchive(t *testing.T) {
	// Feed garbage as an archive
	garbage := strings.NewReader("this is not a cpio archive!!!")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-i"}, garbage, &stdout, &stderr, "/tmp")
	// Should fail gracefully
	_ = rc
}

func TestCpioCreateSkipsEmptyLine(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("real\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Input has blank lines and a valid file
	stdinR := strings.NewReader("\nreal.txt\n\n")
	var archiveOut, stderr bytes.Buffer
	rc := cpioRun([]string{"-o"}, stdinR, &archiveOut, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
}

func TestCpioPassDirHandling(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("subdir\n")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-p", dstDir}, stdinR, &stdout, &stderr, srcDir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
}

func TestCpioFileOpenError(t *testing.T) {
	// Open a non-existent archive file
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-i", "-F", "/nonexistent/archive.cpio"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for nonexistent archive file")
	}
}

func TestCpioCreateFileError(t *testing.T) {
	// Create to an unwritable location
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-o", "-F", "/nonexistent/output.cpio"}, strings.NewReader(""), &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for unwritable output path")
	}
}

func TestCpioRunDispatch(t *testing.T) {
	// Call run() directly to cover the dispatch wrapper
	var stdout, stderr bytes.Buffer
	rc := run([]string{"-v"}, nil, &stdout, &stderr, "/tmp")
	// -v alone = no mode, should fail
	if rc == 0 {
		t.Error("expected non-zero rc for -v without mode")
	}
}

func TestCpioCreateNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	// Reference a non-existent file in stdin
	stdinR := strings.NewReader("/nonexistent/totally_missing_file.txt\n")
	var archiveOut, stderr bytes.Buffer
	// Should emit error but continue; may or may not return non-zero
	_ = cpioRun([]string{"-o"}, stdinR, &archiveOut, &stderr, dir)
}

func TestCpioExtractJSON(t *testing.T) {
	dir := t.TempDir()
	archive := createTestArchive(t, map[string]string{
		"jex.txt": "json extract\n",
	})

	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-id", "--json"}, archive, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"members"`) {
		t.Errorf("expected JSON with members key, got: %s", out)
	}
}

func TestCpioPassThroughDirJSON(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a subdir to pass
	if err := os.Mkdir(filepath.Join(srcDir, "mysubdir"), 0755); err != nil {
		t.Fatal(err)
	}

	stdinR := strings.NewReader("mysubdir\n")
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-p", "--json", dstDir}, stdinR, &stdout, &stderr, srcDir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
}

func TestCpioListModeOnly(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"list_only.txt": "list only test\n",
	})
	var stdout, stderr bytes.Buffer
	rc := cpioRun([]string{"-t"}, archive, &stdout, &stderr, "/tmp")
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "list_only.txt") {
		t.Errorf("expected list_only.txt in output, got: %s", stdout.String())
	}
}
