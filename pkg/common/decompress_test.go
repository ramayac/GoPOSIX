package common

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeDecompressSpec returns a spec whose NewReader just passes input through.
// Good enough to exercise the file walking and output logic without real compression.
func fakeDecompressSpec(name string) DecompressSpec {
	return DecompressSpec{
		ProgName: name,
		HelpText: "Usage: " + name + " [FILE]...",
		Suffixes: []DecompSuffix{{Ext: ".xz", Cut: 3}},
		NewReader: func(r io.Reader) (io.Reader, error) {
			return r, nil
		},
		CorruptMsg: "corrupted data",
		Quietable:  true,
		Flags: FlagSpec{Defs: []FlagDef{
			{Short: "c", Long: "stdout", Type: FlagBool},
			{Short: "f", Long: "force", Type: FlagBool},
			{Short: "k", Long: "keep", Type: FlagBool},
			{Short: "q", Long: "quiet", Type: FlagBool},
			{Short: "h", Long: "help", Type: FlagBool},
			{Long: "json", Type: FlagBool},
		}},
	}
}

func TestDecompressMode_Stdin(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, nil, strings.NewReader("hello"), &out, &errBuf, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.String() != "hello" {
		t.Errorf("got %q, want 'hello'", out.String())
	}
}

func TestDecompressMode_StdinJSON(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	var out bytes.Buffer
	code := DecompressMode(spec, []string{"--json"}, strings.NewReader("hello"), &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	// F16: stdout carries only the envelope; the payload travels as
	// base64 content inside it.
	var env struct {
		Command string `json:"command"`
		Data    struct {
			Files []DecompFileInfo `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope, got %q: %v", out.String(), err)
	}
	if env.Command != "fakedec" || len(env.Data.Files) != 1 || env.Data.Files[0].Source != "-" {
		t.Fatalf("unexpected envelope: %q", out.String())
	}
	if env.Data.Files[0].BytesResult != 5 {
		t.Errorf("expected bytesResult 5, got %d", env.Data.Files[0].BytesResult)
	}
	if env.Data.Files[0].Content != base64.StdEncoding.EncodeToString([]byte("hello")) {
		t.Errorf("expected base64 content, got %q", env.Data.Files[0].Content)
	}
}

func TestDecompressMode_StdoutFileJSON(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	if err := os.WriteFile(src, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"--json", "-c", src}, nil, &out, &errBuf, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %q)", code, errBuf.String())
	}
	var env struct {
		Data struct {
			Files []DecompFileInfo `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope, got %q: %v", out.String(), err)
	}
	if len(env.Data.Files) != 1 {
		t.Fatalf("expected one entry, got %q", out.String())
	}
	entry := env.Data.Files[0]
	if entry.Destination != "-" || entry.BytesResult != 7 {
		t.Errorf("unexpected entry: %+v", entry)
	}
	if entry.Content != base64.StdEncoding.EncodeToString([]byte("payload")) {
		t.Errorf("expected base64 content, got %q", entry.Content)
	}
}

func TestDecompressMode_StdinCorrupt(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return nil, errors.New("bad magic")
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, nil, strings.NewReader("x"), &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "fakedec: corrupted data") {
		t.Errorf("expected corrupt message, got %q", errBuf.String())
	}
}

func TestDecompressMode_StdinQuiet(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return nil, errors.New("bad magic")
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"-q"}, strings.NewReader("x"), &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if errBuf.Len() != 0 {
		t.Errorf("quiet mode must suppress stderr, got %q", errBuf.String())
	}
}

func TestDecompressMode_Help(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	var out bytes.Buffer
	code := DecompressMode(spec, []string{"--help"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage: fakedec") {
		t.Errorf("expected help text, got %q", out.String())
	}
}

func TestDecompressMode_BadFlag(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"--nope"}, nil, &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "fakedec:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestDecompressMode_FileExtract(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{src}, nil, &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	dest := filepath.Join(dir, "data")
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "content" {
		t.Errorf("expected extracted content, got %q err %v", b, err)
	}
	// source removed (no -k)
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("expected source removed")
	}
}

func TestDecompressMode_FileKeep(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"-k", src}, nil, &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if _, err := os.Stat(src); err != nil {
		t.Error("expected source kept with -k")
	}
}

func TestDecompressMode_FileExistsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "data"), []byte("existing"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{src}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "File exists") {
		t.Errorf("expected File exists, got %q", errBuf.String())
	}
}

func TestDecompressMode_FileForce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "data"), []byte("existing"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"-f", src}, nil, &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	b, _ := os.ReadFile(filepath.Join(dir, "data"))
	if string(b) != "content" {
		t.Errorf("expected overwritten content, got %q", b)
	}
}

func TestDecompressMode_StdoutMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"-c", src}, nil, &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.String() != "content" {
		t.Errorf("got %q, want 'content'", out.String())
	}
	if _, err := os.Stat(src); err != nil {
		t.Error("stdout mode must keep source")
	}
}

func TestDecompressMode_UnknownSuffix(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.txt")
	os.WriteFile(src, []byte("x"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{src}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "unknown suffix - ignored") {
		t.Errorf("expected suffix message, got %q", errBuf.String())
	}
}

func TestDecompressMode_MissingAndDir(t *testing.T) {
	dir := t.TempDir()
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{filepath.Join(dir, "gone.xz")}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("missing: exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "No such file or directory") {
		t.Errorf("missing: got %q", errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	code = DecompressMode(spec, []string{filepath.Join(dir, "sub.xz")}, nil, &out, &errBuf, dir)
	os.Mkdir(filepath.Join(dir, "sub.xz"), 0755)
	code = DecompressMode(spec, []string{filepath.Join(dir, "sub.xz")}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("dir: exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "Is a directory") {
		t.Errorf("dir: got %q", errBuf.String())
	}
}

func TestDecompressMode_CatMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.bz2")
	os.WriteFile(src, []byte("content"), 0644)
	spec := fakeDecompressSpec("bzcat")
	spec.Suffixes = nil
	spec.CatMode = true
	spec.Quietable = false
	spec.Flags = FlagSpec{Defs: []FlagDef{{Long: "json", Type: FlagBool}}}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{src}, nil, &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.String() != "content" {
		t.Errorf("got %q, want 'content'", out.String())
	}
}

func TestDecompressMode_RecoverPanics(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	spec.RecoverPanics = true
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		panic("bad magic")
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, nil, strings.NewReader("x"), &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "corrupted data") {
		t.Errorf("expected corrupted data, got %q", errBuf.String())
	}
}

func TestDecompressMode_JSONResults(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out bytes.Buffer
	code := DecompressMode(spec, []string{"--json", "-k", src}, nil, &out, &out, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), `"destination":"`) {
		t.Errorf("expected destination in JSON, got %q", out.String())
	}
}

// panicReader panics on every Read.
type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("boom") }

func TestDecompressMode_SilenceLog(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	spec.SilenceLog = true
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, nil, strings.NewReader("hello"), &out, &errBuf, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out.String() != "hello" {
		t.Errorf("got %q, want 'hello'", out.String())
	}
}

func TestDecompressMode_BadFlagJSON(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"--json", "--nope"}, nil, &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "FLAG_ERROR") {
		t.Errorf("expected FLAG_ERROR envelope on stderr, got %q", errBuf.String())
	}
}

func TestDecompressMode_StdinCorruptJSON(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return nil, errors.New("bad magic")
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"--json"}, strings.NewReader("x"), &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "DECOMPRESS_ERROR") {
		t.Errorf("expected DECOMPRESS_ERROR envelope on stderr, got %q", errBuf.String())
	}
}

func TestDecompressMode_RelativeSrcWithCwd(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.xz"), []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"data.xz"}, nil, &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	b, err := os.ReadFile(filepath.Join(dir, "data"))
	if err != nil || string(b) != "content" {
		t.Errorf("expected extracted content, got %q err %v", b, err)
	}
}

func TestDecompressMode_FileModeRecover(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.xz"), []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	spec.RecoverPanics = true
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		panic("bad magic")
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"data.xz"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "corrupted data") {
		t.Errorf("expected corrupted data, got %q", errBuf.String())
	}
}

func TestDecompressMode_NewReaderErrorFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.xz"), []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return nil, errors.New("bad magic")
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"data.xz"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "corrupted data") {
		t.Errorf("expected corrupted data, got %q", errBuf.String())
	}
}

func TestDecompressMode_CatModeReadError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.bz2"), []byte("content"), 0644)
	spec := fakeDecompressSpec("bzcat")
	spec.Suffixes = nil
	spec.CatMode = true
	spec.Quietable = false
	spec.Flags = FlagSpec{Defs: []FlagDef{{Long: "json", Type: FlagBool}}}
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return errorReader{}, nil
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"data.bz2"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "read failed") {
		t.Errorf("expected raw read error, got %q", errBuf.String())
	}
}

func TestDecompressMode_DestOpenError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.xz"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(dir, "data"), 0755) // dest is a directory
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"-f", "data.xz"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "fakedec:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestDecompressMode_CopyErrorCleanup(t *testing.T) {
	dir := t.TempDir()
	// Symlink with .xz suffix to a file that errors on read (/proc/self/mem).
	os.Symlink("/proc/self/mem", filepath.Join(dir, "data.xz"))
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"data.xz"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "corrupted data") {
		t.Errorf("expected corrupted data, got %q", errBuf.String())
	}
	// incomplete output must be removed
	if _, err := os.Stat(filepath.Join(dir, "data")); !os.IsNotExist(err) {
		t.Error("expected incomplete output removed")
	}
}

func TestDecompressMode_CopyRecoverPanic(t *testing.T) {
	spec := fakeDecompressSpec("fakedec")
	spec.RecoverPanics = true
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return panicReader{}, nil
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, nil, strings.NewReader("x"), &out, &errBuf, "")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "corrupted data") {
		t.Errorf("expected corrupted data, got %q", errBuf.String())
	}
}

func TestDecompressMode_StdoutCopyError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.xz"), []byte("content"), 0644)
	spec := fakeDecompressSpec("fakedec")
	spec.NewReader = func(r io.Reader) (io.Reader, error) {
		return errorReader{}, nil
	}
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"-c", "data.xz"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "corrupted data") {
		t.Errorf("expected corrupted data, got %q", errBuf.String())
	}
}

func TestDecompressMode_SrcOpenError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission test needs a non-root user")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "data.xz")
	os.WriteFile(src, []byte("content"), 0000)
	spec := fakeDecompressSpec("fakedec")
	var out, errBuf bytes.Buffer
	code := DecompressMode(spec, []string{"data.xz"}, nil, &out, &errBuf, dir)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "permission denied") {
		t.Errorf("expected permission error, got %q", errBuf.String())
	}
}
