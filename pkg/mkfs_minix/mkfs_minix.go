// Package mkfs_minix implements the POSIX/BusyBox mkfs.minix utility.
package mkfs_minix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "i", Long: "inodes", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// MkfsResult is the structured output for --json mode.
type MkfsResult struct {
	Inodes        uint16 `json:"inodes"`
	Zones         uint16 `json:"zones"`
	FirstDataZone uint16 `json:"first_data_zone"`
	IMapBlocks    uint16 `json:"imap_blocks"`
	ZMapBlocks    uint16 `json:"zmap_blocks"`
}

// MinixSuperBlock represents the Minix V1 Superblock structure.
type MinixSuperBlock struct {
	NInodes       uint16
	NZones        uint16
	IMapBlocks    uint16
	ZMapBlocks    uint16
	FirstDataZone uint16
	LogZoneSize   uint16
	MaxSize       uint32
	Magic         uint16
	State         uint16
}

// MinixInode represents a Minix V1 Inode structure (32 bytes).
type MinixInode struct {
	Mode   uint16
	UID    uint16
	Size   uint32
	Time   uint32
	GID    uint8
	NLinks uint8
	Zones  [9]uint16
}

// Run creates a Minix V1 filesystem and writes it to w.
func Run(w io.Writer, blocks, inodes int) (MkfsResult, error) {
	if blocks < 10 {
		return MkfsResult{}, fmt.Errorf("number of blocks must be at least 10")
	}

	// Calculate default inodes if not specified
	if inodes <= 0 {
		inodes = blocks / 3
		if inodes < 16 {
			inodes = 16
		}
		// Inode table size constraints
		if inodes > 65535 {
			inodes = 65535
		}
	}

	// Sizing block structures
	imapBlocks := uint16((inodes + 8191) / 8192)
	zmapBlocks := uint16((blocks + 8191) / 8192)
	inodeBlocks := uint16((inodes*32 + 1023) / 1024)
	firstDataZone := uint16(2 + imapBlocks + zmapBlocks + inodeBlocks)

	if int(firstDataZone) >= blocks {
		return MkfsResult{}, fmt.Errorf("filesystem too small for requested metadata")
	}

	// Adjust inodes to exactly match inodeBlocks capacity
	inodesActual := int(inodeBlocks) * 32

	res := MkfsResult{
		Inodes:        uint16(inodesActual),
		Zones:         uint16(blocks),
		FirstDataZone: firstDataZone,
		IMapBlocks:    imapBlocks,
		ZMapBlocks:    zmapBlocks,
	}

	// 1. Boot Block (Block 0)
	bootBlock := make([]byte, 1024)
	if _, err := w.Write(bootBlock); err != nil {
		return res, err
	}

	// 2. Superblock (Block 1)
	sb := MinixSuperBlock{
		NInodes:       uint16(inodesActual),
		NZones:        uint16(blocks),
		IMapBlocks:    imapBlocks,
		ZMapBlocks:    zmapBlocks,
		FirstDataZone: firstDataZone,
		LogZoneSize:   0,
		MaxSize:       268966912, // V1 max file size
		Magic:         0x137F,    // V1 14-char filename magic
		State:         1,         // Clean
	}

	sbBuf := new(bytes.Buffer)
	if err := binary.Write(sbBuf, binary.LittleEndian, sb); err != nil {
		return res, err
	}
	// Pad superblock block to 1024 bytes
	sbBlock := make([]byte, 1024)
	copy(sbBlock, sbBuf.Bytes())
	if _, err := w.Write(sbBlock); err != nil {
		return res, err
	}

	// 3. Inode Bitmap blocks
	imapSize := int(imapBlocks) * 1024
	imap := make([]byte, imapSize)
	// Inode 0 is reserved. Inode 1 is root directory.
	// So set bit 0 and bit 1 of inode bitmap to 1.
	imap[0] = 0x03 // 00000011
	// Set out-of-bounds bits to 1
	for bit := inodesActual + 1; bit < imapSize*8; bit++ {
		byteIdx := bit / 8
		bitIdx := bit % 8
		imap[byteIdx] |= 1 << bitIdx
	}
	if _, err := w.Write(imap); err != nil {
		return res, err
	}

	// 4. Zone Bitmap blocks
	zmapSize := int(zmapBlocks) * 1024
	zmap := make([]byte, zmapSize)
	// Zone 0 is reserved. Zone 1 is root directory.
	// So set bit 0 and bit 1 of zone bitmap to 1.
	zmap[0] = 0x03
	// Set out-of-bounds bits to 1
	for bit := blocks; bit < zmapSize*8; bit++ {
		byteIdx := bit / 8
		bitIdx := bit % 8
		zmap[byteIdx] |= 1 << bitIdx
	}
	if _, err := w.Write(zmap); err != nil {
		return res, err
	}

	// 5. Inode Table blocks
	inodeTableSize := int(inodeBlocks) * 1024
	inodeTable := make([]byte, inodeTableSize)
	// Write Inode 1 (root directory) at offset 32 bytes
	rootInode := MinixInode{
		Mode:   040755, // Directory with 755 permissions
		UID:    0,
		Size:   32,
		Time:   0,
		GID:    0,
		NLinks: 2,
	}
	rootInode.Zones[0] = firstDataZone

	rootInodeBuf := new(bytes.Buffer)
	if err := binary.Write(rootInodeBuf, binary.LittleEndian, rootInode); err != nil {
		return res, err
	}
	copy(inodeTable[32:64], rootInodeBuf.Bytes())
	if _, err := w.Write(inodeTable); err != nil {
		return res, err
	}

	// 6. First Data Zone (Root Directory Entries)
	rootBlock := make([]byte, 1024)
	// Entry 1: "." pointing to Inode 1
	binary.LittleEndian.PutUint16(rootBlock[0:2], 1)
	copy(rootBlock[2:16], ".")
	// Entry 2: ".." pointing to Inode 1
	binary.LittleEndian.PutUint16(rootBlock[16:18], 1)
	copy(rootBlock[18:32], "..")

	if _, err := w.Write(rootBlock); err != nil {
		return res, err
	}

	// 7. Write remaining blocks (fully zero-filled zones)
	remBlocks := blocks - int(firstDataZone) - 1
	zeroBlock := make([]byte, 1024)
	for i := 0; i < remBlocks; i++ {
		if _, err := w.Write(zeroBlock); err != nil {
			return res, err
		}
	}

	return res, nil
}

func mkfsMinixRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "mkfs.minix: %v\n", err)
		return 2
	}

	if len(flags.Positional) < 1 {
		fmt.Fprintln(errOut, "mkfs.minix: missing device or file operand")
		return 2
	}

	target := flags.Positional[0]
	blocks := 1440 // default floppy size
	if len(flags.Positional) > 1 {
		fmt.Sscanf(flags.Positional[1], "%d", &blocks)
	}

	inodes := 0
	if val := flags.Get("i"); val != "" {
		fmt.Sscanf(val, "%d", &inodes)
	}

	jsonMode := flags.Has("json")

	// Open output file
	file, err := os.Create(target)
	if err != nil {
		fmt.Fprintf(errOut, "mkfs.minix: %v\n", err)
		return 1
	}
	defer file.Close()

	res, err := Run(file, blocks, inodes)
	if err != nil {
		fmt.Fprintf(errOut, "mkfs.minix: %v\n", err)
		return 1
	}

	if jsonMode {
		common.Render("mkfs.minix", res, true, stdout, func() {})
		return 0
	}

	// Output standard stats summary
	fmt.Fprintf(stdout, "%d inodes\n", res.Inodes)
	fmt.Fprintf(stdout, "%d blocks\n", res.Zones)
	fmt.Fprintf(stdout, "Firstdatazone=%d (%d)\n", res.FirstDataZone, res.FirstDataZone)
	fmt.Fprintf(stdout, "Zonesize=1024\n")
	fmt.Fprintf(stdout, "Maxsize=268966912\n")

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "mkfs.minix",
		Usage: "Create a Minix filesystem",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			return mkfsMinixRun(args, stdout, stderr, stdin, cwd)
		},
	})
}
