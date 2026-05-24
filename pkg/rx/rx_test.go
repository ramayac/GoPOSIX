package rx

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

func buildXMODEMPacket(data []byte, blockNum byte) []byte {
	crc := crc16(data)
	packet := []byte{SOH, blockNum, byte(255 - blockNum)}
	packet = append(packet, data...)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	packet = append(packet, crcBytes...)
	return packet
}

func TestRxBasic(t *testing.T) {
	data := make([]byte, blockSize)
	copy(data, "Hello, XMODEM!")
	for i := 14; i < blockSize; i++ {
		data[i] = 0x1A
	}
	crc := crc16(data)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)

	packet := []byte{SOH, 1, 0xFE}
	packet = append(packet, data...)
	packet = append(packet, crcBytes...)
	packet = append(packet, EOT)

	outFile := t.TempDir() + "/rx.out"
	var stdout strings.Builder
	n, err := receiveFile(strings.NewReader(string(packet)), &stdout, outFile)
	if err != nil {
		t.Fatalf("receiveFile: %v", err)
	}
	if n != 14 {
		t.Errorf("bytes: got %d, want 14", n)
	}

	out := stdout.String()
	if len(out) != 3 || out[0] != 'C' || out[1] != ACK || out[2] != ACK {
		t.Errorf("stdout: got %x, want 43 06 06", []byte(out))
	}

	content, _ := os.ReadFile(outFile)
	if string(content) != "Hello, XMODEM!" {
		t.Errorf("file content: got %q, want %q", string(content), "Hello, XMODEM!")
	}
}

func TestRxCancel(t *testing.T) {
	// Just a CAN byte
	var stdout strings.Builder
	outFile := t.TempDir() + "/rx.out"
	_, err := receiveFile(strings.NewReader(string([]byte{CAN})), &stdout, outFile)
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected cancel error, got: %v", err)
	}
}

func TestRxNoResponse(t *testing.T) {
	// Empty input
	var stdout strings.Builder
	outFile := t.TempDir() + "/rx.out"
	_, err := receiveFile(strings.NewReader(""), &stdout, outFile)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestRxEOTOnly(t *testing.T) {
	// Immediate EOT = empty file transfer
	var stdout strings.Builder
	outFile := t.TempDir() + "/rx.out"
	n, err := receiveFile(strings.NewReader(string([]byte{EOT})), &stdout, outFile)
	if err != nil {
		t.Fatalf("receiveFile EOT: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}
}

func TestRxCRCError(t *testing.T) {
	// Corrupt CRC
	data := make([]byte, blockSize)
	for i := range data {
		data[i] = byte('A')
	}
	packet := []byte{SOH, 1, 0xFE}
	packet = append(packet, data...)
	packet = append(packet, 0x00, 0x00) // wrong CRC

	var stdout strings.Builder
	outFile := t.TempDir() + "/rx.out"
	_, err := receiveFile(strings.NewReader(string(packet)), &stdout, outFile)
	if err == nil {
		t.Fatal("expected CRC error")
	}
}

func TestRxJSONMode(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run([]string{"--json", "/dev/null"}, strings.NewReader(string([]byte{EOT})), &stdout, &stderr, "")
	if rc != 0 {
		t.Fatalf("JSON mode returned %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"fileName\"") {
		t.Errorf("JSON missing fileName: %s", stdout.String())
	}
}

func TestRxMissingFile(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run(nil, strings.NewReader(""), &stdout, &stderr, "")
	if rc == 0 {
		t.Fatal("expected error for missing filename")
	}
}

func TestRxFlagError(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run([]string{"--bad-flag"}, strings.NewReader(""), &stdout, &stderr, "")
	if rc == 0 {
		t.Fatal("expected error for bad flag")
	}
}

func TestRxCRCTable(t *testing.T) {
	// Verify CRC table initialization
	crc := crc16([]byte("123456789"))
	if crc != 0x31C3 {
		t.Errorf("CRC of '123456789': got %04X, want 29B1", crc)
	}
}
