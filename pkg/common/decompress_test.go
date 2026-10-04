package common

import (
	"bytes"
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
	if !strings.Contains(out.String(), `"source":"-"`) {
		t.Errorf("expected stdin JSON entry, got %q", out.String())
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
