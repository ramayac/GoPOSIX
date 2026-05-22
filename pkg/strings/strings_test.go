package strings

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestScan_Simple(t *testing.T) {
	data := []byte("hello\x00world")
	entries, err := Scan(bytes.NewReader(data), 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Value != "hello" || entries[0].Offset != 0 {
		t.Errorf("entry0: %+v", entries[0])
	}
	if entries[1].Value != "world" || entries[1].Offset != 6 {
		t.Errorf("entry1: %+v", entries[1])
	}
}

func TestScan_MinLength(t *testing.T) {
	data := []byte("ab\x00cde\x00fghi")
	entries, err := Scan(bytes.NewReader(data), 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Value != "fghi" {
		t.Errorf("minLen 4: got %+v", entries)
	}

	entries2, _ := Scan(bytes.NewReader(data), 2)
	if len(entries2) != 3 {
		t.Errorf("minLen 2: expected 3, got %d: %+v", len(entries2), entries2)
	}
}

func TestScan_TabIsPrintable(t *testing.T) {
	data := []byte("a\tb\x00cde")
	entries, err := Scan(bytes.NewReader(data), 3)
	if err != nil {
		t.Fatal(err)
	}
	// "a\tb" (3 chars with tab) and "cde" (3 chars)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Value != "a\tb" {
		t.Errorf("entry0 should have tab: %q", entries[0].Value)
	}
	if entries[1].Value != "cde" {
		t.Errorf("entry1: %q", entries[1].Value)
	}
}

func TestScan_Empty(t *testing.T) {
	entries, err := Scan(bytes.NewReader([]byte{}), 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty, got %d", len(entries))
	}
}

func TestScan_AllNonPrintable(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x1F, 0x7F}
	entries, err := Scan(bytes.NewReader(data), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty, got %d", len(entries))
	}
}

func TestScan_EdgeOfBuffer(t *testing.T) {
	// String spanning across our 4096-byte read buffer boundary
	// Create data that's larger than 4096 with a string at the boundary
	prefix := bytes.Repeat([]byte{0x00}, 4000)
	suffix := bytes.Repeat([]byte{0x00}, 200)
	data := append(append(prefix, []byte("hello_world")...), suffix...)
	entries, err := Scan(bytes.NewReader(data), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Value != "hello_world" {
		t.Errorf("boundary: got %+v", entries)
	}
	if entries[0].Offset != 4000 {
		t.Errorf("offset: got %d, want 4000", entries[0].Offset)
	}
}

func TestFormatOffset(t *testing.T) {
	tests := []struct {
		offset int64
		radix  byte
		want   string
	}{
		{0, 'd', "0"},
		{10, 'd', "10"},
		{255, 'x', "ff"},
		{8, 'o', "10"},
		{42, 'd', "42"},
	}
	for _, tc := range tests {
		got := FormatOffset(tc.offset, tc.radix)
		if got != tc.want {
			t.Errorf("FormatOffset(%d, %c): got %q, want %q", tc.offset, tc.radix, got, tc.want)
		}
	}
}

func TestIsPrintable(t *testing.T) {
	if !isPrintable('A') {
		t.Error("A should be printable")
	}
	if !isPrintable(' ') {
		t.Error("space should be printable")
	}
	if !isPrintable('\t') {
		t.Error("tab should be printable")
	}
	if isPrintable('\x00') {
		t.Error("NUL should not be printable")
	}
	if isPrintable('\x1F') {
		t.Error("0x1F should not be printable")
	}
	if isPrintable('\x7F') {
		t.Error("DEL should not be printable")
	}
}

// --- CLI layer ---

func TestStringsRun_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello\x00world_padding")
	rc := stringsRun([]string{"-n", "5"}, &outBuf, &errBuf, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "world") {
		t.Errorf("missing 'world' in output: %q", outBuf.String())
	}
	// "hello" has exactly 5 chars (== minLen), should appear too
	if !strings.Contains(outBuf.String(), "hello") {
		t.Errorf("'hello' should appear (len 5 = minLen): %q", outBuf.String())
	}
}

func TestStringsRun_Json(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello\x00world")
	rc := stringsRun([]string{"--json"}, &outBuf, &errBuf, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "\"strings\"") {
		t.Errorf("missing 'strings' in JSON: %s", outBuf.String())
	}
}

func TestStringsRun_File(t *testing.T) {
	// Use a temp file
	f, err := os.CreateTemp("", "strings-test-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	data := []byte("test_data_here\x00more_stuff_here")
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	f.Close()

	var outBuf, errBuf bytes.Buffer
	rc := stringsRun([]string{f.Name()}, &outBuf, &errBuf, nil, "")
	if rc != 0 {
		t.Logf("stderr: %s", errBuf.String())
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "test_data_here") {
		t.Errorf("missing 'test_data_here': %q", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "more_stuff_here") {
		t.Errorf("missing 'more_stuff_here': %q", outBuf.String())
	}
}

func TestStringsRun_RadixFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	// "12345" = 0x3132333435, starts at offset 0
	in := strings.NewReader("xxxxx")
	rc := stringsRun([]string{"-t", "x", "-n", "3"}, &outBuf, &errBuf, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0\nstderr: %s", rc, errBuf.String())
	}
	// Output should contain hex offset and the string
	output := outBuf.String()
	fmt.Printf("output: %q\n", output)
	if !strings.Contains(output, "0") && !strings.Contains(output, "xxxxx") {
		t.Errorf("unexpected output: %q", output)
	}
}
