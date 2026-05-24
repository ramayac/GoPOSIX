package cryptpw

import (
	"bytes"
	"strings"
	"testing"
)

func TestCryptpwHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: cryptpw")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestCryptpwFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for invalid flags")
	}
}

func TestCryptpwDES(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: des salt 12
	code := run([]string{"-m", "des", "QWErty", "123456789012345678901234567890"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	expected := "12MnB3PqfVbMA\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}

	// Case 2: des salt 55
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "des", "QWErty", "55"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatal(code)
	}
	expected = "55tgFLtkT1Y72\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}

	// Case 3: des salt zz
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "des", "QWErty", "zz"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatal(code)
	}
	expected = "zzIZaaXWOkxVk\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}
}

func TestCryptpwMD5(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-m", "md5", "QWErty", "salt1234"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "$1$salt1234$") {
		t.Errorf("Expected hash to start with prefix, got: %s", stdout.String())
	}

	// With predefined salt prefix
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "md5", "-S", "$1$salt", "QWErty"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatal(code)
	}
	if !strings.HasPrefix(stdout.String(), "$1$salt$") {
		t.Errorf("Expected hash to start with prefix, got: %s", stdout.String())
	}
}

func TestCryptpwSHA256(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Standard SHA256
	code := run([]string{"-m", "sha256", "QWErty", "123456789012345678901234567890"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	expected := "$5$1234567890123456$5DxfOCmU4vRhtzfsbdK.6wSGMwwVbac7ZkWwusb8Si7\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}

	// Case 2: Rounds
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "sha256", "QWErty", "rounds=99999$123456789012345678901234567890"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatal(code)
	}
	expected = "$5$rounds=99999$1234567890123456$aYellycJGZM6AKyVzaQsSrDBdTixubtMnM6J.MN0xM8\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}
}

func TestCryptpwSHA512(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Standard SHA512
	code := run([]string{"-m", "sha512", "QWErty", "123456789012345678901234567890"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	expected := "$6$1234567890123456$KB7QqxFyqmJSWyQYcCuGeFukgz1bPQoipWZf7.9L7z3k8UNTXa6UikbKcUGDc2ANn7DOGmDaroxDgpK16w/RE0\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}

	// Case 2: Rounds
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "sha512", "QWErty", "rounds=99999$123456789012345678901234567890"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatal(code)
	}
	expected = "$6$rounds=99999$1234567890123456$BfF6gD6ZjUmwawH5QaAglYAxtU./yvsz0fcQ464l49aMI2DZW3j5ri28CrxK7riPWNpLuUpfaIdY751SBYKUH.\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}
}

func TestCryptpwStdinAndErrors(t *testing.T) {
	stdin := bytes.NewReader([]byte("stdinPassword\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Read password from stdin
	code := run([]string{"-m", "sha256"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "$5$12345678$") {
		t.Errorf("Expected hash of stdin password, got: %s", stdout.String())
	}

	// Case 2: Unknown method
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "invalid-method", "pw"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for invalid hashing method")
	}

	// Case 3: Unknown method JSON
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-m", "invalid-method", "pw"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected JSON error for invalid hashing method")
	}
}

func TestCryptpwJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--json", "-m", "des", "QWErty", "12"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0 in JSON mode, got %d. Stderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"hash":"12MnB3PqfVbMA"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"method":"des"`)) {
		t.Errorf("Expected valid JSON response, got:\n%s", stdout.String())
	}
}

func TestCryptpwErrorPaths(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: DES password too long (> 8 chars)
	code := run([]string{"-m", "des", "longpassword123", "12"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected DES to fail with password > 8 chars")
	}

	// Case 2: DES password too long JSON
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-m", "des", "longpassword123", "12"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected DES JSON to fail with password > 8 chars")
	}

	// Case 3: DES invalid salt
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "des", "pw", "!!"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected DES to fail with invalid salt")
	}
}

func TestBuildModularSalt(t *testing.T) {
	tests := []struct {
		prefix  string
		rawSalt string
		maxLen  int
		want    string
	}{
		// Basic salt
		{"$5$", "hello", 16, "$5$hello"},
		// Salt longer than maxLen
		{"$5$", "12345678901234567890", 16, "$5$1234567890123456"},
		// Pre-existing prefix → pass through
		{"$5$", "$5$existing", 16, "$5$existing"},
		// rounds= format
		{"$5$", "rounds=99999$mysalt", 16, "$5$rounds=99999$mysalt"},
		// rounds= with long salt
		{"$5$", "rounds=100$12345678901234567890", 8, "$5$rounds=100$12345678"},
		// SHA-512 prefix
		{"$6$", "somesalt", 16, "$6$somesalt"},
		// rounds= with SHA-512
		{"$6$", "rounds=5000$secret", 16, "$6$rounds=5000$secret"},
		// Empty salt
		{"$5$", "", 16, "$5$"},
		// Pre-existing prefix with rounds
		{"$5$", "$5$rounds=20000$salt", 16, "$5$rounds=20000$salt"},
	}
	for _, tc := range tests {
		got := buildModularSalt(tc.prefix, tc.rawSalt, tc.maxLen)
		if got != tc.want {
			t.Errorf("buildModularSalt(%q, %q, %d) = %q, want %q",
				tc.prefix, tc.rawSalt, tc.maxLen, got, tc.want)
		}
	}
}

func TestCryptpwSaltFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Use -S flag for salt
	code := run([]string{"-m", "sha256", "-S", "customsalt", "QWErty"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "$5$customsalt$") {
		t.Errorf("Expected hash with custom salt, got: %s", stdout.String())
	}

	// Use -S with md5
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "md5", "-S", "mysalt", "QWErty"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatal(code)
	}
	if !strings.HasPrefix(stdout.String(), "$1$mysalt$") {
		t.Errorf("Expected md5 hash with custom salt, got: %s", stdout.String())
	}
}

func TestCryptpwHelpJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--json", "--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("\"help\"")) {
		t.Errorf("Expected JSON help output, got: %s", stdout.String())
	}
}

func TestCryptpwFlagErrorNonJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for invalid flags")
	}
	if stderr.Len() == 0 {
		t.Error("Expected stderr output for flag error")
	}
}

func TestCryptpwDESEdgeCases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// DES with empty password
	code := run([]string{"-m", "des", "", "12"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0 for empty password, got %d", code)
	}
	// DES password exactly 8 chars
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", "des", "12345678", "ab"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0 for 8-char password, got %d: %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "ab") {
		t.Errorf("Expected DES hash starting with salt, got: %s", stdout.String())
	}
}

func TestCryptpwDefaultMethod(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Default method (sha256) with password only
	code := run([]string{"testpass"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d", code)
	}
	if !strings.HasPrefix(stdout.String(), "$5$12345678$") {
		t.Errorf("Expected default sha256 hash, got: %s", stdout.String())
	}
}
