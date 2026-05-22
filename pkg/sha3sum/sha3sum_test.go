package sha3sum

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errorReader struct{}

func (e errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("mock read failure")
}

func TestHashFile(t *testing.T) {
	input := "The quick brown fox jumps over the lazy dog"
	r := strings.NewReader(input)

	// NIST SHA3-224 expected hex
	expected224 := "d15dadceaa4d5d7bb3b48f446421d542e08ad8887305e28d58335795"
	hash, err := HashFile(r, "224")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != expected224 {
		t.Errorf("HashFile(224) expected %s, got %s", expected224, hash)
	}

	// 384
	_, err = HashFile(strings.NewReader(input), "384")
	if err != nil {
		t.Fatalf("unexpected error for 384: %v", err)
	}

	// 512
	_, err = HashFile(strings.NewReader(input), "512")
	if err != nil {
		t.Fatalf("unexpected error for 512: %v", err)
	}

	// HashFile error path
	_, err = HashFile(errorReader{}, "224")
	if err == nil {
		t.Error("expected error from failing reader")
	}

	// Invalid algorithm
	_, err = HashFile(strings.NewReader(input), "999")
	if err == nil {
		t.Error("expected error for invalid algorithm size")
	}
}

func TestSha3sumRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sha3sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}

	// 1. Basic Hashing default (224)
	var stdout, stderr bytes.Buffer
	gotExit := run([]string{file1}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run basic expected exit 0, got %d. stderr: %s", gotExit, stderr.String())
	}
	expectedHash224 := "dfb7f18c77e928bb56faeb2da27291bd790bc1045cde45f3210bb6c5"
	if !strings.Contains(stdout.String(), expectedHash224) {
		t.Errorf("stdout expected containing hash %s, got: %s", expectedHash224, stdout.String())
	}

	// 2. Select algorithm size (256)
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-a", "256", file1}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run -a 256 expected exit 0, got %d", gotExit)
	}
	expectedHash256 := "644bcc7e564373040999aac89e7622f3ca71fba1d972fd94a31c3bfbf24e3938"
	if !strings.Contains(stdout.String(), expectedHash256) {
		t.Errorf("stdout expected containing hash %s, got: %s", expectedHash256, stdout.String())
	}

	// 3. Stdin Hashing
	stdout.Reset()
	stderr.Reset()
	stdin := strings.NewReader("hello world")
	gotExit = run([]string{"-"}, stdin, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run stdin expected exit 0, got %d", gotExit)
	}
	if !strings.Contains(stdout.String(), expectedHash224) {
		t.Errorf("stdout expected containing hash %s, got: %s", expectedHash224, stdout.String())
	}

	// 4. Invalid algorithm selection
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-a", "invalid", file1}, nil, &stdout, &stderr, "")
	if gotExit == 0 {
		t.Error("run with invalid algorithm expected non-zero exit")
	}

	// 5. JSON Mode hashing
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"--json", file1}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run json expected exit 0, got %d", gotExit)
	}
	if !strings.Contains(stdout.String(), `"hash":`) {
		t.Errorf("expected json payload, got: %s", stdout.String())
	}

	// 6. Verification / Check Mode
	checksumContent := expectedHash224 + "  " + file1 + "\n"
	checkFile := filepath.Join(tmpDir, "check.sha3")
	if err := os.WriteFile(checkFile, []byte(checksumContent), 0644); err != nil {
		t.Fatalf("failed to write check file: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", checkFile}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run check mode expected exit 0, got %d. stderr: %s", gotExit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "file1.txt: OK") {
		t.Errorf("check mode expected success log, got: %s", stdout.String())
	}

	// 7. Invalid flag syntax
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for invalid flag, got %d", gotExit)
	}

	// 8. JSON Mode with invalid algorithm
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"--json", "-a", "999", file1}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for json mode with invalid algorithm, got %d", gotExit)
	}

	// 9. Missing file hashing
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{filepath.Join(tmpDir, "missing.txt")}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for missing file, got %d", gotExit)
	}

	// 10. Malformed checksum line
	malformedCheckFile := filepath.Join(tmpDir, "malformed.sha3")
	if err := os.WriteFile(malformedCheckFile, []byte("badline\n"), 0644); err != nil {
		t.Fatalf("failed to write malformed check file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", malformedCheckFile}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for malformed checksum line, got %d", gotExit)
	}

	// 11. Empty checksum file
	emptyCheckFile := filepath.Join(tmpDir, "empty.sha3")
	if err := os.WriteFile(emptyCheckFile, []byte("\n# comment\n"), 0644); err != nil {
		t.Fatalf("failed to write empty check file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", emptyCheckFile}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for empty checksum file, got %d", gotExit)
	}

	// 12. Check mode with missing target file
	missingCheckFile := filepath.Join(tmpDir, "missing_target.sha3")
	if err := os.WriteFile(missingCheckFile, []byte(expectedHash224+"  missing_target.txt\n"), 0644); err != nil {
		t.Fatalf("failed to write missing target check file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", missingCheckFile}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for check with missing target, got %d", gotExit)
	}

	// 13. Check mode with hash mismatch
	mismatchCheckFile := filepath.Join(tmpDir, "mismatch.sha3")
	badHash := "00000000000000000000000000000000000000000000000000000000"
	if err := os.WriteFile(mismatchCheckFile, []byte(badHash+"  "+file1+"\n"), 0644); err != nil {
		t.Fatalf("failed to write mismatch check file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", mismatchCheckFile}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for check with hash mismatch, got %d", gotExit)
	}

	// 14. Check mode from stdin
	stdout.Reset()
	stderr.Reset()
	stdinCheck := strings.NewReader(expectedHash224 + "  " + file1 + "\n")
	gotExit = run([]string{"-c", "-"}, stdinCheck, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0 for stdin check mode, got %d", gotExit)
	}

	// 15. Algorithm 384 and 512 hashing
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-a", "384", file1}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run -a 384 expected exit 0, got %d", gotExit)
	}

	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-a", "512", file1}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("run -a 512 expected exit 0, got %d", gotExit)
	}

	// 16. Check mode with -a flag restricting algorithm
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", "-a", "224", checkFile}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("check with -a 224 expected exit 0, got %d. stderr: %s", gotExit, stderr.String())
	}

	// 17. Check mode--nonexistent check file
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-c", "/nonexistent_check_sha3file"}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for nonexistent check file, got %d", gotExit)
	}
}
