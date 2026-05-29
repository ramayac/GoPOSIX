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

func TestRxCoverageExt(t *testing.T) {
	t.Run("duplicate block (expected-1)", func(t *testing.T) {
		data := make([]byte, blockSize)
		copy(data, "hello")
		for i := 5; i < blockSize; i++ {
			data[i] = 0x1A
		}
		packet := buildXMODEMPacket(data, 1)

		// Send block 1, then duplicate block 1, then EOT
		var stream []byte
		stream = append(stream, packet...)
		stream = append(stream, packet...)
		stream = append(stream, EOT)

		var stdout strings.Builder
		outFile := t.TempDir() + "/rx_dup.out"
		n, err := receiveFile(strings.NewReader(string(stream)), &stdout, outFile)
		if err != nil {
			t.Fatal(err)
		}
		if n != 5 {
			t.Errorf("bytes written: got %d, want 5", n)
		}
	})

	t.Run("invalid inverse block number", func(t *testing.T) {
		data1 := make([]byte, blockSize)
		copy(data1, "first")
		for i := 5; i < blockSize; i++ {
			data1[i] = 0x1A
		}
		packet1 := buildXMODEMPacket(data1, 1)

		data2 := make([]byte, blockSize)
		// SOH, block 2, bad inverse (not 253)
		packet2 := []byte{SOH, 2, 0xFF}
		packet2 = append(packet2, data2...)
		crcBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(crcBytes, crc16(data2))
		packet2 = append(packet2, crcBytes...)

		var stream []byte
		stream = append(stream, packet1...)
		stream = append(stream, packet2...)

		var stdout strings.Builder
		outFile := t.TempDir() + "/rx_bad_inv.out"
		n, err := receiveFile(strings.NewReader(string(stream)), &stdout, outFile)
		if err != nil {
			t.Fatalf("expected graceful completion, got error: %v", err)
		}
		if n != 5 {
			t.Errorf("expected only first block of 5 bytes to be written, got %d", n)
		}
	})

	t.Run("unexpected block number", func(t *testing.T) {
		data1 := make([]byte, blockSize)
		copy(data1, "first")
		for i := 5; i < blockSize; i++ {
			data1[i] = 0x1A
		}
		packet1 := buildXMODEMPacket(data1, 1)

		data3 := make([]byte, blockSize)
		packet3 := buildXMODEMPacket(data3, 3) // expecting 2, got 3

		var stream []byte
		stream = append(stream, packet1...)
		stream = append(stream, packet3...)

		var stdout strings.Builder
		outFile := t.TempDir() + "/rx_unexpected.out"
		n, err := receiveFile(strings.NewReader(string(stream)), &stdout, outFile)
		if err != nil {
			t.Fatalf("expected graceful completion, got error: %v", err)
		}
		if n != 5 {
			t.Errorf("expected only first block of 5 bytes to be written, got %d", n)
		}
	})

	t.Run("cancel command in loop", func(t *testing.T) {
		data := make([]byte, blockSize)
		packet1 := buildXMODEMPacket(data, 1)

		var stream []byte
		stream = append(stream, packet1...)
		stream = append(stream, CAN)

		var stdout strings.Builder
		outFile := t.TempDir() + "/rx_cancel_loop.out"
		_, err := receiveFile(strings.NewReader(string(stream)), &stdout, outFile)
		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Errorf("expected cancel error, got %v", err)
		}
	})

	t.Run("incomplete read in loop", func(t *testing.T) {
		data := make([]byte, blockSize)
		packet1 := buildXMODEMPacket(data, 1)

		// Send block 1, then incomplete block 2 (just SOH)
		var stream []byte
		stream = append(stream, packet1...)
		stream = append(stream, SOH, 2) // truncated

		var stdout strings.Builder
		outFile := t.TempDir() + "/rx_inc.out"
		_, err := receiveFile(strings.NewReader(string(stream)), &stdout, outFile)
		if err != nil {
			t.Fatal("expected graceful exit to done on incomplete read")
		}
	})

	t.Run("write file error", func(t *testing.T) {
		var stdout strings.Builder
		// Write to non-existent folder
		_, err := receiveFile(strings.NewReader(string([]byte{EOT})), &stdout, "/nonexistent/rx.out")
		if err == nil {
			t.Fatal("expected write file error")
		}
	})

	t.Run("handshake invalid first block", func(t *testing.T) {
		// SOH followed by bad block number (not 1)
		packet := []byte{SOH, 2, 0xFD}
		packet = append(packet, make([]byte, blockSize+2)...)

		var stdout strings.Builder
		outFile := t.TempDir() + "/rx_handshake_err.out"
		_, err := receiveFile(strings.NewReader(string(packet)), &stdout, outFile)
		if err == nil {
			t.Fatal("expected error for invalid first block in handshake")
		}
	})
}
