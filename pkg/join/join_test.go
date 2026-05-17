package join

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJoinDefault(t *testing.T) {
	r1 := strings.NewReader("1 one\n2 two\n3 three\n")
	r2 := strings.NewReader("1 alpha\n2 beta\n4 gamma\n")

	result, err := Run(r1, r2, 1, 1, " ", false, false, false, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"1 one alpha", "2 two beta"}
	if len(result.Records) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(result.Records))
	}
	for i, rec := range result.Records {
		if rec["line"] != expected[i] {
			t.Errorf("record %d: expected %q, got %q", i, expected[i], rec["line"])
		}
	}
}

func TestJoinCustomDelimiter(t *testing.T) {
	r1 := strings.NewReader("1:one\n2:two\n3:three\n")
	r2 := strings.NewReader("1:alpha\n2:beta\n4:gamma\n")

	result, err := Run(r1, r2, 1, 1, ":", false, false, false, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"1:one:alpha", "2:two:beta"}
	if len(result.Records) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(result.Records))
	}
	for i, rec := range result.Records {
		if rec["line"] != expected[i] {
			t.Errorf("record %d: expected %q, got %q", i, expected[i], rec["line"])
		}
	}
}

func TestJoinUnpairedA1A2(t *testing.T) {
	r1 := strings.NewReader("1:one\n2:two\n3:three\n")
	r2 := strings.NewReader("1:alpha\n2:beta\n4:gamma\n")

	result, err := Run(r1, r2, 1, 1, ":", true, true, false, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"1:one:alpha", "2:two:beta", "3:three", "4:gamma"}
	if len(result.Records) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(result.Records))
	}
	for i, rec := range result.Records {
		if rec["line"] != expected[i] {
			t.Errorf("record %d: expected %q, got %q", i, expected[i], rec["line"])
		}
	}
}

func TestJoinV1Only(t *testing.T) {
	r1 := strings.NewReader("1:one\n2:two\n3:three\n")
	r2 := strings.NewReader("1:alpha\n2:beta\n4:gamma\n")

	result, err := Run(r1, r2, 1, 1, ":", false, false, true, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"3:three"}
	if len(result.Records) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(result.Records))
	}
	for i, rec := range result.Records {
		if rec["line"] != expected[i] {
			t.Errorf("record %d: expected %q, got %q", i, expected[i], rec["line"])
		}
	}
}

func TestJoinCustomFields(t *testing.T) {
	r1 := strings.NewReader("a:1:x\nb:2:y\n")
	r2 := strings.NewReader("p:1:q\nr:3:s\n")

	// Join on field 2 of file1 (index 1) and field 2 of file2 (index 1)
	result, err := Run(r1, r2, 2, 2, ":", false, false, false, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"a:1:x:p:q"}
	if len(result.Records) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(result.Records))
	}
	for i, rec := range result.Records {
		if rec["line"] != expected[i] {
			t.Errorf("record %d: expected %q, got %q", i, expected[i], rec["line"])
		}
	}
}

func TestJoinOutputFormat(t *testing.T) {
	r1 := strings.NewReader("1:one\n2:two\n")
	r2 := strings.NewReader("1:alpha\n2:beta\n")

	result, err := Run(r1, r2, 1, 1, ":", false, false, false, false, "1.2 2.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"one:alpha", "two:beta"}
	if len(result.Records) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(result.Records))
	}
	for i, rec := range result.Records {
		if rec["line"] != expected[i] {
			t.Errorf("record %d: expected %q, got %q", i, expected[i], rec["line"])
		}
	}
}

func TestJoinCLI_Basic(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")
	os.WriteFile(f1, []byte("a 1\nb 2\n"), 0644)
	os.WriteFile(f2, []byte("a x\nb y\n"), 0644)
	var out bytes.Buffer
	code := run([]string{f1, f2}, &out)
	if code != 0 { t.Errorf("exit %d, want 0", code) }
	if out.String() != "a 1 x\nb 2 y\n" { t.Errorf("got %q", out.String()) }
}

func TestJoinCLI_UnpairedLines(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")
	os.WriteFile(f1, []byte("a 1\nc 3\n"), 0644)
	os.WriteFile(f2, []byte("a x\nb y\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-a1", "-a2", f1, f2}, &out)
	if code != 0 { t.Errorf("exit %d, want 0", code) }
	if out.Len() == 0 { t.Error("expected output") }
}
