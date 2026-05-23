package ar

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bsar "github.com/blakesmith/ar"
)

// createTestArchive builds an ar archive in memory and writes it to a temp file.
func createTestArchive(t *testing.T, members map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.a")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	w := bsar.NewWriter(f)
	if err := w.WriteGlobalHeader(); err != nil {
		f.Close()
		t.Fatal(err)
	}
	for name, content := range members {
		hdr := &bsar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0644,
		}
		if err := w.WriteHeader(hdr); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			f.Close()
			t.Fatal(err)
		}
	}
	f.Close()
	return archivePath
}

func TestArList(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"file1.txt": "hello world\n",
		"file2.txt": "foo bar\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"t", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "file1.txt") || !strings.Contains(out, "file2.txt") {
		t.Errorf("expected both file names in output, got: %s", out)
	}
}

func TestArListVerbose(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"readme.md": "# README\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"tv", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "readme.md") {
		t.Errorf("expected readme.md in verbose output, got: %s", out)
	}
}

func TestArListJSON(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"a.c": "int main(){}\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"t", "--json", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"name"`) || !strings.Contains(out, "a.c") {
		t.Errorf("expected JSON with name field, got: %s", out)
	}
}

func TestArPrint(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"msg.txt": "hello from ar\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"p", archivePath, "msg.txt"}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if stdout.String() != "hello from ar\n" {
		t.Errorf("unexpected output: %q", stdout.String())
	}
}

func TestArExtract(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"extracted.txt": "extracted content\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"x", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "extracted.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "extracted content\n" {
		t.Errorf("unexpected extracted content: %q", string(data))
	}
}

func TestArReplace(t *testing.T) {
	dir := t.TempDir()
	// Create initial archive
	archivePath := createTestArchive(t, map[string]string{
		"old.txt": "old content\n",
	})

	// Write a new file to add
	newFile := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"r", archivePath, newFile}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	// Verify new.txt is in archive
	var listOut, listErr bytes.Buffer
	rc = arRun([]string{"t", archivePath}, nil, &listOut, &listErr, dir)
	if rc != 0 {
		t.Fatalf("list failed: %s", listErr.String())
	}
	if !strings.Contains(listOut.String(), "new.txt") {
		t.Errorf("expected new.txt in archive, got: %s", listOut.String())
	}
	if !strings.Contains(listOut.String(), "old.txt") {
		t.Errorf("expected old.txt still in archive, got: %s", listOut.String())
	}
}

func TestArReplaceExisting(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"file.txt": "original\n",
	})

	updatedFile := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(updatedFile, []byte("updated\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"r", archivePath, updatedFile}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	// Verify content was updated
	var printOut, printErr bytes.Buffer
	rc = arRun([]string{"p", archivePath, "file.txt"}, nil, &printOut, &printErr, dir)
	if rc != 0 {
		t.Fatalf("print failed: %s", printErr.String())
	}
	if printOut.String() != "updated\n" {
		t.Errorf("expected updated content, got: %q", printOut.String())
	}
}

func TestArDelete(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"keep.txt":   "keep me\n",
		"delete.txt": "delete me\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"d", archivePath, "delete.txt"}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	var listOut, listErr bytes.Buffer
	rc = arRun([]string{"t", archivePath}, nil, &listOut, &listErr, dir)
	if rc != 0 {
		t.Fatalf("list failed: %s", listErr.String())
	}
	if strings.Contains(listOut.String(), "delete.txt") {
		t.Errorf("delete.txt should be gone, got: %s", listOut.String())
	}
	if !strings.Contains(listOut.String(), "keep.txt") {
		t.Errorf("keep.txt should still exist, got: %s", listOut.String())
	}
}

func TestArDeleteNoFiles(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{"f.txt": "x\n"})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"d", archivePath}, nil, &stdout, &stderr, dir)
	if rc == 0 {
		t.Error("expected non-zero rc when no files specified for delete")
	}
}

func TestArCreateNew(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "brand_new.a")

	srcFile := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(srcFile, []byte("brand new\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"rc", archivePath, srcFile}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive was not created: %v", err)
	}
}

func TestArNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for no args")
	}
}

func TestArMissingArchive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"t"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for missing archive arg")
	}
}

func TestArNoOperation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"vV", "/tmp/test.a"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for no operation")
	}
}

func TestArExtractVerbose(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"verbose.txt": "verbose test\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"xv", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "x - verbose.txt") {
		t.Errorf("expected verbose output, got: %s", stdout.String())
	}
}

func TestArExtractJSON(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"json_test.txt": "json content\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"x", "--json", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"name"`) {
		t.Errorf("expected JSON output, got: %s", out)
	}
}

func TestArPrintVerbose(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"pv.txt": "pv content\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"pv", archivePath, "pv.txt"}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "<pv.txt>") {
		t.Errorf("expected verbose header in output, got: %s", out)
	}
}

func TestArOpenNonExistentForList(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"t", "/nonexistent/path/test.a"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for nonexistent archive")
	}
}

func TestArModeString(t *testing.T) {
	// Test modeString helper
	result := modeString(0644)
	if result[0] != '-' {
		t.Errorf("expected leading '-', got: %s", result)
	}
	if len(result) != 10 {
		t.Errorf("expected 10-char mode string, got: %q (%d chars)", result, len(result))
	}
}

func TestArReplaceVerbose(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"existing.txt": "original\n",
	})

	// Write a replacement file
	repFile := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(repFile, []byte("replaced!\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Add a brand new file
	newFile := filepath.Join(dir, "brand_new.txt")
	if err := os.WriteFile(newFile, []byte("brand new\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"rv", archivePath, repFile, newFile}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "r - existing.txt") {
		t.Errorf("expected 'r - existing.txt' in verbose output, got: %s", out)
	}
	if !strings.Contains(out, "a - brand_new.txt") {
		t.Errorf("expected 'a - brand_new.txt' in verbose output, got: %s", out)
	}
}

func TestArDeleteVerbose(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"del_verbose.txt": "delete me\n",
		"keep_verbose.txt": "keep me\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"dv", archivePath, "del_verbose.txt"}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "d - del_verbose.txt") {
		t.Errorf("expected 'd - del_verbose.txt' in verbose output, got: %s", stdout.String())
	}
}

func TestArExtractFilter(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"extract_a.txt": "extract a\n",
		"extract_b.txt": "extract b\n",
	})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"x", archivePath, "extract_a.txt"}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}

	// a should be extracted, b should not
	if _, err := os.Stat(filepath.Join(dir, "extract_a.txt")); err != nil {
		t.Error("extract_a.txt should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "extract_b.txt")); err == nil {
		t.Error("extract_b.txt should NOT exist")
	}
}

func TestArPrintAllFiles(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{
		"print_a.txt": "print a\n",
		"print_b.txt": "print b\n",
	})

	var stdout, stderr bytes.Buffer
	// Print all (no filter)
	rc := arRun([]string{"p", archivePath}, nil, &stdout, &stderr, dir)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "print a") || !strings.Contains(out, "print b") {
		t.Errorf("expected both contents, got: %s", out)
	}
}

func TestArPrintNonExistent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"p", "/nonexistent/test.a", "file.txt"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for nonexistent archive")
	}
}

func TestArExtractNonExistent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"x", "/nonexistent/test.a"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for nonexistent archive")
	}
}

func TestArDeleteNonExistent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"d", "/nonexistent/test.a", "file.txt"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for nonexistent archive")
	}
}

func TestArReplaceNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	archivePath := createTestArchive(t, map[string]string{"dummy.txt": "x\n"})

	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"r", archivePath, "/nonexistent/file.txt"}, nil, &stdout, &stderr, dir)
	if rc == 0 {
		t.Error("expected non-zero rc for nonexistent file to add")
	}
}

func TestArJSONOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := arRun([]string{"--json"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc with only --json and no operation")
	}
}
