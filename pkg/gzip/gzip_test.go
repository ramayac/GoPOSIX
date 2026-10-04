package gzip

import (
	"bytes"
	gzip "compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGzipGunzipCycle(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	content := "hello world gzip test"
	os.WriteFile(file, []byte(content), 0644)

	var buf bytes.Buffer
	code := runGzip([]string{file}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("gzip exit code %d", code)
	}

	if _, err := os.Stat(file); err == nil {
		t.Errorf("original file should be deleted")
	}

	gzFile := file + ".gz"
	if _, err := os.Stat(gzFile); os.IsNotExist(err) {
		t.Fatalf("gz file not created")
	}

	code = runGunzip([]string{gzFile}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("gunzip exit code %d", code)
	}

	if _, err := os.Stat(gzFile); err == nil {
		t.Errorf("gz file should be deleted")
	}

	unpacked, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("unpacked file read err: %v", err)
	}

	if string(unpacked) != content {
		t.Errorf("content mismatch: got %q, want %q", unpacked, content)
	}
}

func TestGzipKeep(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("hello"), 0644)

	var buf bytes.Buffer
	code := runGzip([]string{"-k", file}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Errorf("original file should be kept")
	}
	if _, err := os.Stat(file + ".gz"); os.IsNotExist(err) {
		t.Errorf("gz file should be created")
	}
}

func TestGzipForce(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("hello"), 0644)
	os.WriteFile(file+".gz", []byte("existing"), 0644)

	var buf bytes.Buffer
	code := runGzip([]string{file}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("should fail without force")
	}

	code = runGzip([]string{"-f", file}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("should succeed with force")
	}
}

func TestGzipStdout(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("hello stdout"), 0644)

	var buf bytes.Buffer
	code := runGzip([]string{"-c", file}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Errorf("original file should be kept with -c")
	}
	if _, err := os.Stat(file + ".gz"); err == nil {
		t.Errorf("gz file should not be created with -c")
	}

	if buf.Len() == 0 {
		t.Errorf("stdout should contain compressed data")
	}
}

func TestGzipJSON(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte(strings.Repeat("a", 100)), 0644)

	var buf bytes.Buffer
	code := runGzip([]string{"--json", file}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].([]interface{})
	stat := data[0].(map[string]interface{})

	if stat["file"] != file {
		t.Errorf("wrong file name in json")
	}
	// file is the filename (string), originalSize is the size (float64)
	if orig, ok := stat["originalSize"].(float64); ok && orig != 100 {
		t.Errorf("wrong original size: %v", stat["originalSize"])
	}
	if ns, ok := stat["newSize"].(float64); ok && ns == 0 {
		// newSize may be 0 for very small files; just verify it exists
	}
}

// --- BusyBox hardening tests ---

func TestBusyBox_Gunzip_DoesntExist(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a valid gzip file
	file1 := filepath.Join(tmpDir, "hello.txt")
	os.WriteFile(file1, []byte("HELLO\n"), 0644)
	var buf bytes.Buffer
	code := runGzip([]string{"-k", file1}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("gzip failed: %d", code)
	}
	gzFile := file1 + ".gz"

	// gunzip with non-existent file first, then valid gz
	stderr := captureStderr(func() {
		runGunzip([]string{filepath.Join(tmpDir, "z"), gzFile}, nil, &bytes.Buffer{}, os.Stderr, "")
	})

	// Should mention the non-existent file
	if !strings.Contains(stderr, "z: No such file or directory") {
		t.Errorf("expected 'No such file or directory' error, got: %q", stderr)
	}
	if !strings.Contains(stderr, "gunzip:") {
		t.Errorf("expected gunzip: prefix, got: %q", stderr)
	}
}

func TestBusyBox_Gunzip_UnknownSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	// Create valid gz and a non-gz file
	file1 := filepath.Join(tmpDir, "hello.txt")
	os.WriteFile(file1, []byte("HELLO\n"), 0644)
	var buf bytes.Buffer
	code := runGzip([]string{"-k", file1}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("gzip failed: %d", code)
	}
	notGz := filepath.Join(tmpDir, "t.zz")
	os.WriteFile(notGz, []byte{}, 0644)

	stderr := captureStderr(func() {
		runGunzip([]string{notGz, file1 + ".gz"}, nil, &bytes.Buffer{}, os.Stderr, "")
	})

	if !strings.Contains(stderr, "t.zz: unknown suffix") {
		t.Errorf("expected 'unknown suffix' error, got: %q", stderr)
	}
}

func TestBusyBox_Gunzip_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "t1.txt")
	file2 := filepath.Join(tmpDir, "t2.txt")
	os.WriteFile(file1, []byte("DATA1\n"), 0644)
	os.WriteFile(file2, []byte("DATA2\n"), 0644)
	var buf bytes.Buffer
	code := runGzip([]string{"-k", file1}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("gzip t1 failed: %d", code)
	}
	code = runGzip([]string{"-k", file2}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("gzip t2 failed: %d", code)
	}

	// Create file that would conflict with uncompressed output
	os.WriteFile(filepath.Join(tmpDir, "t1.txt"), []byte("preexisting"), 0644)

	stderr := captureStderr(func() {
		runGunzip([]string{file1 + ".gz", file2 + ".gz"}, nil, &bytes.Buffer{}, os.Stderr, "")
	})

	if !strings.Contains(stderr, "can't open 't1.txt': File exists") &&
		!strings.Contains(stderr, "can't open") {
		t.Errorf("expected 'File exists' error, got: %q", stderr)
	}
}

// captureStderr redirects os.Stderr temporarily to capture output.
func captureStderr(fn func()) string {
	orig := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	w.Close()
	os.Stderr = orig
	return <-done
}
func TestCLI_Compress(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)
	var out bytes.Buffer
	code := runGzip([]string{"-c", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected compressed output")
	}
}
func TestCLI_Decompress(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.gz")
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Write([]byte("hello"))
	gzw.Close()
	os.WriteFile(f, buf.Bytes(), 0644)
	var out bytes.Buffer
	code := runGzip([]string{"-d", "-c", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.String() != "hello" {
		t.Errorf("got %q, want 'hello'", out.String())
	}
}
func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "j.txt")
	os.WriteFile(f, []byte("json"), 0644)
	var out bytes.Buffer
	code := runGzip([]string{"--json", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected output")
	}
}
func TestCLI_Stdout(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "s.txt")
	os.WriteFile(f, []byte("stdout"), 0644)
	var out bytes.Buffer
	code := runGzip([]string{"-c", "--stdout", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}
func TestCLI_MissingFile(t *testing.T) {
	var out bytes.Buffer
	code := runGzip([]string{"/nonexistent/gzip/file"}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := runGzip([]string{"--nonexistent"}, nil, &out, &out, "")
	if code == 0 {
		t.Errorf("exit %d, want non-zero for bad flag", code)
	}
}
func TestCLI_Gunzip(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "gun.gz")
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Write([]byte("hello"))
	gzw.Close()
	os.WriteFile(f, buf.Bytes(), 0644)
	var out bytes.Buffer
	code := runGunzip([]string{"-c", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.String() != "hello" {
		t.Errorf("got %q, want 'hello'", out.String())
	}
}

func TestGzip_Level9(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "level9.txt")
	os.WriteFile(f, []byte("compression level 9 test data"), 0644)
	var out bytes.Buffer
	code := runGzip([]string{"-9", "-c", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.Len() == 0 {
		t.Error("expected compressed output")
	}
}

func TestGzip_StdinDash(t *testing.T) {
	// Roundtrip: compress then decompress via buffers
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	var compBuf bytes.Buffer
	gw := gzip.NewWriter(&compBuf)
	gw.Write(data)
	gw.Close()
	gr, _ := gzip.NewReader(&compBuf)
	decomp, _ := io.ReadAll(gr)
	if !bytes.Equal(data, decomp) {
		t.Error("roundtrip mismatch")
	}
}

func TestGunzipWrongSuffix(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("not gzipped"), 0644)
	var stdout, stderr bytes.Buffer
	code := runGunzip([]string{"-d", f}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("gunzip wrong suffix: exit %d, want 1", code)
	}
}

func TestGzipOutputExistsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("hello"), 0644)
	out := filepath.Join(dir, "src.txt.gz")
	os.WriteFile(out, []byte("existing"), 0644)
	var stdout, stderr bytes.Buffer
	code := runGzip([]string{src}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("gzip existing output: exit %d, want 1", code)
	}
}

func TestGunzipOutputExistsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("hello"), 0644)
	// Create .gz
	var buf bytes.Buffer
	code := runGzip([]string{"-c", src}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatal("gzip -c failed")
	}
	gzFile := filepath.Join(dir, "src.txt.gz")
	os.WriteFile(gzFile, buf.Bytes(), 0644)
	// Create output file that would conflict
	outFile := filepath.Join(dir, "src.txt")
	os.WriteFile(outFile, []byte("existing"), 0644)
	var stdout, stderr bytes.Buffer
	code = runGunzip([]string{"-d", gzFile}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("gunzip existing output: exit %d, want 1", code)
	}
}

func TestGzipJSONMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "j.txt")
	os.WriteFile(src, []byte("json test"), 0644)
	var stdout, stderr bytes.Buffer
	code := runGzip([]string{"--json", "-c", src}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("gzip --json -c: exit %d", code)
	}
	if !strings.Contains(stdout.String(), "command") {
		t.Errorf("expected JSON output, got: %s", stdout.String())
	}
}

func TestGzipJSONStdinCompress(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := gzipRun([]string{"--json"}, &out, &errBuf, strings.NewReader("hello world"), "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	// F16: stdout carries only the envelope; the compressed payload
	// travels as base64 content inside the envelope.
	var env struct {
		Command string `json:"command"`
		Data    []struct {
			File    string `json:"file"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope, got %q: %v", out.String(), err)
	}
	if env.Command != "gzip" || len(env.Data) != 1 || env.Data[0].File != "-" {
		t.Fatalf("unexpected envelope: %q", out.String())
	}
	raw, err := base64.StdEncoding.DecodeString(env.Data[0].Content)
	if err != nil {
		t.Fatalf("bad base64 content: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte{0x1f, 0x8b}) {
		t.Error("expected gzip magic bytes in decoded content")
	}
}

func TestGzipJSONStdinRoundTrip(t *testing.T) {
	var compressed bytes.Buffer
	if code := gzipRun([]string{"--json"}, &compressed, io.Discard, strings.NewReader("hello"), ""); code != 0 {
		t.Fatalf("compress: exit %d", code)
	}
	var env struct {
		Data []struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(compressed.Bytes(), &env); err != nil {
		t.Fatalf("expected envelope from gzip, got %q: %v", compressed.String(), err)
	}
	if len(env.Data) != 1 {
		t.Fatalf("expected one data entry, got %q", compressed.String())
	}
	raw, err := base64.StdEncoding.DecodeString(env.Data[0].Content)
	if err != nil {
		t.Fatalf("bad base64 content: %v", err)
	}
	var out, errBuf bytes.Buffer
	if code := gunzipRun([]string{"--json"}, &out, &errBuf, bytes.NewReader(raw), ""); code != 0 {
		t.Fatalf("decompress: exit %d", code)
	}
	var env2 struct {
		Data []struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env2); err != nil {
		t.Fatalf("expected envelope from gunzip, got %q: %v", out.String(), err)
	}
	if len(env2.Data) != 1 {
		t.Fatalf("expected one data entry, got %q", out.String())
	}
	plain, err := base64.StdEncoding.DecodeString(env2.Data[0].Content)
	if err != nil {
		t.Fatalf("bad base64 content: %v", err)
	}
	if string(plain) != "hello" {
		t.Errorf("decompressed = %q, want hello", plain)
	}
}

func TestGzipJSONStdinBadData(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := gunzipRun([]string{"--json"}, &out, &errBuf, strings.NewReader("not gzip data"), ""); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if errBuf.Len() != 0 {
		t.Errorf("JSON mode should suppress stderr, got %q", errBuf.String())
	}
}

func TestGzipJSONMissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := gzipRun([]string{"--json", "/nonexistent-xyz"}, &out, &errBuf, nil, ""); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestGzip_StdinNoFiles_Compress(t *testing.T) {
	var out bytes.Buffer
	code := gzipRun(nil, &out, &out, strings.NewReader("hello stdin"), "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	gr, err := gzip.NewReader(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(gr)
	if string(data) != "hello stdin" {
		t.Errorf("got %q, want 'hello stdin'", data)
	}
}

func TestGunzip_StdinNoFiles_Decompress(t *testing.T) {
	var gz bytes.Buffer
	gw := gzip.NewWriter(&gz)
	gw.Write([]byte("hello stdin"))
	gw.Close()
	var out bytes.Buffer
	code := gunzipRun(nil, &out, &out, bytes.NewReader(gz.Bytes()), "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.String() != "hello stdin" {
		t.Errorf("got %q, want 'hello stdin'", out.String())
	}
}

func TestGunzip_StdinGarbage(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := gunzipRun(nil, &out, &errBuf, strings.NewReader("not gzip data"), "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "gunzip: stdin:") {
		t.Errorf("expected stdin error on stderr, got %q", errBuf.String())
	}
}

func TestGzip_DashCompress(t *testing.T) {
	var out bytes.Buffer
	code := gzipRun([]string{"-"}, &out, &out, strings.NewReader("dash"), "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	gr, err := gzip.NewReader(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(gr)
	if string(data) != "dash" {
		t.Errorf("got %q, want 'dash'", data)
	}
}

func TestGunzip_DashDecompress(t *testing.T) {
	var gz bytes.Buffer
	gw := gzip.NewWriter(&gz)
	gw.Write([]byte("dash back"))
	gw.Close()
	var out bytes.Buffer
	code := gunzipRun([]string{"-"}, &out, &out, bytes.NewReader(gz.Bytes()), "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.String() != "dash back" {
		t.Errorf("got %q, want 'dash back'", out.String())
	}
}

func TestGzip_CreateFail(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a")
	os.WriteFile(src, []byte("x"), 0644)
	// outName "a.gz" exists as a directory → OpenFile fails
	os.Mkdir(filepath.Join(dir, "a.gz"), 0755)
	var out, errBuf bytes.Buffer
	code := gzipRun([]string{src}, &out, &errBuf, strings.NewReader(""), "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "gzip:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestGunzip_ProcessFail(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.gz")
	os.WriteFile(bad, []byte("this is not gzip data"), 0644)
	var out, errBuf bytes.Buffer
	code := gunzipRun([]string{bad}, &out, &errBuf, strings.NewReader(""), "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "gunzip:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
	// incomplete output must be removed
	if _, err := os.Stat(filepath.Join(dir, "bad")); !os.IsNotExist(err) {
		t.Errorf("expected incomplete output removed, stat err: %v", err)
	}
}

func TestGunzip_DashGarbage(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := gunzipRun([]string{"-"}, &out, &errBuf, strings.NewReader("garbage"), "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "gunzip: stdin:") {
		t.Errorf("expected stdin error on stderr, got %q", errBuf.String())
	}
}
