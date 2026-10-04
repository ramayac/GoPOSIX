package common

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func digestTestSpec(name, alg string, newFn func() hash.Hash) DigestSpec {
	return DigestSpec{ProgName: name, Algorithm: alg, New: newFn}
}

func TestDigestReader(t *testing.T) {
	got, err := DigestReader(md5.New(), strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	want := hex.EncodeToString(func() []byte { h := md5.New(); h.Write([]byte("hello")); return h.Sum(nil) }())
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestDigestHashMode_File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	code := DigestHashMode(digestTestSpec("md5sum", "md5", md5.New), []string{f}, false, false, strings.NewReader(""), &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	want := "5d41402abc4b2a76b9719d911017c592  " + f + "\n"
	if out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestDigestHashMode_StdinDefault(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := DigestHashMode(digestTestSpec("sha256sum", "sha256", sha256.New), nil, false, false, strings.NewReader("abc"), &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "-") {
		t.Errorf("expected stdin marker, got %q", out.String())
	}
}

func TestDigestHashMode_MissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := DigestHashMode(digestTestSpec("md5sum", "md5", md5.New), []string{"/nonexistent-xyz"}, false, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "md5sum: /nonexistent-xyz:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestDigestHashMode_MissingFileJSON(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := DigestHashMode(digestTestSpec("md5sum", "md5", md5.New), []string{"/nonexistent-xyz"}, false, true, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "IO") {
		t.Errorf("expected JSON error envelope, got %q", out.String())
	}
}

func TestDigestHashMode_JSON(t *testing.T) {
	var out bytes.Buffer
	code := DigestHashMode(digestTestSpec("sha256sum", "sha256", sha256.New), []string{"-"}, false, true, strings.NewReader("abc"), &out, &out)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), `"algorithm":"sha256"`) {
		t.Errorf("expected algorithm in JSON, got %q", out.String())
	}
}

func TestDigestCheckMode_NoFilesError(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), nil, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "md5sum: no checksum file specified") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
	if !strings.Contains(out.String(), "MISSING_FILE") == false {
		t.Log("json envelope only in json mode")
	}
}

func TestDigestCheckMode_NoFilesStdin(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("hello"), 0644)
	digest, _ := DigestReader(md5.New(), strings.NewReader("hello"))
	line := digest + "  " + f + "\n"

	spec := digestTestSpec("sha1sum", "sha1", md5.New)
	spec.CheckNoFilesStdin = true
	var out, errBuf bytes.Buffer
	code := DigestCheckMode(spec, nil, false, strings.NewReader(line), &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), f+": OK") {
		t.Errorf("expected OK line, got %q", out.String())
	}
}

func TestDigestCheckMode_OKAndFail(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	bad := filepath.Join(dir, "bad.txt")
	os.WriteFile(good, []byte("hello"), 0644)
	os.WriteFile(bad, []byte("world"), 0644)
	digest, _ := DigestReader(md5.New(), strings.NewReader("hello"))
	lines := digest + "  " + good + "\n" + digest + "  " + bad + "\n"
	cf := filepath.Join(dir, "checks.md5")
	os.WriteFile(cf, []byte(lines), 0644)

	var out, errBuf bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), []string{cf}, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), good+": OK") {
		t.Errorf("expected OK line, got %q", out.String())
	}
	if !strings.Contains(out.String(), bad+": FAILED") {
		t.Errorf("expected FAILED line, got %q", out.String())
	}
}

func TestDigestCheckMode_ImproperlyFormatted(t *testing.T) {
	dir := t.TempDir()
	cf := filepath.Join(dir, "checks.md5")
	os.WriteFile(cf, []byte("notachecksumline\n"), 0644)

	var out, errBuf bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), []string{cf}, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "improperly formatted checksum line") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestDigestCheckMode_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cf := filepath.Join(dir, "checks.md5")
	os.WriteFile(cf, []byte("# only a comment\n"), 0644)

	var out, errBuf bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), []string{cf}, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "no properly formatted checksum lines found") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestDigestCheckMode_MissingChecksumFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), []string{"/nonexistent.md5"}, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "md5sum: /nonexistent.md5:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestDigestCheckMode_MissingTarget(t *testing.T) {
	dir := t.TempDir()
	cf := filepath.Join(dir, "checks.md5")
	digest, _ := DigestReader(md5.New(), strings.NewReader("hello"))
	os.WriteFile(cf, []byte(digest+"  "+filepath.Join(dir, "gone.txt")+"\n"), 0644)

	var out, errBuf bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), []string{cf}, false, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "FAILED open or read") {
		t.Errorf("expected FAILED message, got %q", errBuf.String())
	}
}

func TestDigestCheckMode_ResolveAlg(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("hello"), 0644)
	digest, _ := DigestReader(sha256.New(), strings.NewReader("hello"))
	cf := filepath.Join(dir, "checks.txt")
	os.WriteFile(cf, []byte(digest+"  "+f+"\n"), 0644)

	spec := digestTestSpec("sha3sum", "sha3-256", sha256.New)
	spec.CheckNoFilesStdin = true
	spec.CheckResolveAlg = func(expectedHash string) (hash.Hash, string, error) {
		return sha256.New(), "sha3-256", nil
	}
	var out, errBuf bytes.Buffer
	code := DigestCheckMode(spec, []string{cf}, false, strings.NewReader(""), &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), f+": OK") {
		t.Errorf("expected OK line, got %q", out.String())
	}
}

func TestDigestCheckMode_JSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("hello"), 0644)
	digest, _ := DigestReader(md5.New(), strings.NewReader("hello"))
	cf := filepath.Join(dir, "checks.md5")
	os.WriteFile(cf, []byte(digest+"  "+f+"\n"), 0644)

	var out bytes.Buffer
	code := DigestCheckMode(digestTestSpec("md5sum", "md5", md5.New), []string{cf}, true, strings.NewReader(""), &out, &out)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), `"status":"OK"`) {
		t.Errorf("expected JSON result, got %q", out.String())
	}
}
