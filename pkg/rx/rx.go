// Package rx implements an XMODEM file receiver.
package rx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

const (
	SOH       = 0x01
	EOT       = 0x04
	ACK       = 0x06
	NAK       = 0x15
	CAN       = 0x18
	C         = 0x43
	blockSize = 128
	maxRetries = 10
)

type RxResult struct {
	BytesWritten int64  `json:"bytesWritten"`
	FileName     string `json:"fileName"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

var crcTable [256]uint16

func initCRC() {
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
		crcTable[i] = crc
	}
}

func crc16(data []byte) uint16 {
	crc := uint16(0)
	for _, b := range data {
		crc = (crc << 8) ^ crcTable[(crc>>8)^uint16(b)]
	}
	return crc
}

func receiveFile(r io.Reader, cw io.Writer, path string) (int64, error) {
	var fileBuf bytes.Buffer
	blockExpected := byte(1)
	buf := make([]byte, 2+blockSize+2)

	// Handshake
	for attempt := 0; attempt < maxRetries; attempt++ {
		cw.Write([]byte{C})
		var first [1]byte
		if _, err := io.ReadFull(r, first[:]); err != nil {
			continue
		}
		switch first[0] {
		case SOH:
			if _, err := io.ReadFull(r, buf); err != nil {
				return 0, fmt.Errorf("read first block: %w", err)
			}
			if buf[0] != 1 || buf[1] != 0xFE {
				return 0, fmt.Errorf("invalid first block")
			}
			data := buf[2 : 2+blockSize]
			rcvCRC := binary.BigEndian.Uint16(buf[2+blockSize:])
			if crc16(data) != rcvCRC {
				cw.Write([]byte{NAK})
				continue
			}
			fileBuf.Write(data)
			blockExpected = 2
			cw.Write([]byte{ACK})
			goto recvLoop
		case EOT:
			cw.Write([]byte{ACK})
			goto done
		case CAN:
			return 0, fmt.Errorf("transfer cancelled")
		}
	}
	return 0, fmt.Errorf("no response from sender")

recvLoop:
	for {
		var first [1]byte
		if _, err := io.ReadFull(r, first[:]); err != nil {
			goto done
		}
		switch first[0] {
		case SOH:
			if _, err := io.ReadFull(r, buf); err != nil {
				goto done
			}
			if buf[0] != blockExpected {
				if buf[0] == blockExpected-1 {
					cw.Write([]byte{ACK})
					continue
				}
				goto done
			}
			if buf[1] != byte(255-buf[0]) {
				goto done
			}
			data := buf[2 : 2+blockSize]
			rcvCRC := binary.BigEndian.Uint16(buf[2+blockSize:])
			if crc16(data) != rcvCRC {
				cw.Write([]byte{NAK})
				continue
			}
			fileBuf.Write(data)
			blockExpected++
			cw.Write([]byte{ACK})
		case EOT:
			cw.Write([]byte{ACK})
			goto done
		case CAN:
			return 0, fmt.Errorf("transfer cancelled")
		}
	}

done:
	// Strip trailing 0x1A (CP/M EOF padding)
	raw := fileBuf.Bytes()
	n := len(raw) - 1
	for n >= 0 && raw[n] == 0x1A {
		n--
	}
	if err := os.WriteFile(path, raw[:n+1], 0644); err != nil {
		return 0, err
	}
	return int64(n + 1), nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "rx: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	posArgs := flags.Positional
	if len(posArgs) < 1 {
		fmt.Fprintln(stderr, "rx: missing output filename")
		return 1
	}

	bytesWritten, err := receiveFile(stdin, stdout, posArgs[0])
	if err != nil {
		fmt.Fprintf(stderr, "rx: %v\n", err)
		return 1
	}

	result := RxResult{FileName: posArgs[0], BytesWritten: bytesWritten}
	common.Render("rx", result, jsonMode, stdout, func() {})
	return 0
}

func init() {
	initCRC()
	dispatch.Register(dispatch.Command{
		Name:  "rx",
		Usage: "XMODEM file receiver",
		Run:   run,
	})
}
