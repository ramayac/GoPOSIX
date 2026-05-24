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
		{Short: "n", Long: "namelen", Type: common.FlagValue},
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
func Run(w io.Writer, blocks, inodes, namelen int) (MkfsResult, error) {
	if blocks < 10 {
		return MkfsResult{}, fmt.Errorf("number of blocks must be at least 10")
	}

	if namelen != 14 && namelen != 30 {
		namelen = 30
	}

	// Calculate default inodes if not specified
	if inodes <= 0 {
		inodes = blocks / 3
	}

	// Round up inode count to fill block size (multiple of 32 for V1)
	inodes = (inodes + 31) &^ 31
	if inodes < 16 {
		inodes = 16
	}
	if inodes > 65535 {
		inodes = 65535
	}

	imapBlocks := (inodes + 1 + 8191) / 8192

	// Iteratively compute zmapBlocks and firstDataZone
	zmapBlocks := 0
	firstDataZone := 0
	for i := 0; i < 1000; i++ {
		inodeBlocks := (inodes * 32 + 1023) / 1024
		firstDataZone = 2 + imapBlocks + zmapBlocks + inodeBlocks
		sb_zmaps := (blocks - firstDataZone + 1 + 8191) / 8192
		if zmapBlocks == sb_zmaps {
			break
		}
		zmapBlocks = sb_zmaps
	}

	if firstDataZone >= blocks {
		return MkfsResult{}, fmt.Errorf("filesystem too small for requested metadata")
	}

	res := MkfsResult{
		Inodes:        uint16(inodes),
		Zones:         uint16(blocks),
		FirstDataZone: uint16(firstDataZone),
		IMapBlocks:    uint16(imapBlocks),
		ZMapBlocks:    uint16(zmapBlocks),
	}

	// Magic: 0x137F for 14-char, 0x138F for 30-char
	magic := uint16(0x138F)
	if namelen == 14 {
		magic = 0x137F
	}

	// 1. Boot Block (Block 0) - 1024 bytes (first 512 bytes are boot sector, padded to 1024)
	bootBlock := make([]byte, 1024)
	if _, err := w.Write(bootBlock); err != nil {
		return res, err
	}

	// 2. Superblock (Block 1)
	sb := MinixSuperBlock{
		NInodes:       uint16(inodes),
		NZones:        uint16(blocks),
		IMapBlocks:    uint16(imapBlocks),
		ZMapBlocks:    uint16(zmapBlocks),
		FirstDataZone: uint16(firstDataZone),
		LogZoneSize:   0,
		MaxSize:       (7 + 512 + 512*512) * 1024, // V1 max file size
		Magic:         magic,
		State:         1, // Clean
	}

	sbBuf := new(bytes.Buffer)
	if err := binary.Write(sbBuf, binary.LittleEndian, sb); err != nil {
		return res, err
	}
	sbBlock := make([]byte, 1024)
	copy(sbBlock, sbBuf.Bytes())
	if _, err := w.Write(sbBlock); err != nil {
		return res, err
	}

	// Helpers for bit setting/clearing
	setBit := func(buf []byte, bit int) {
		buf[bit/8] |= 1 << (bit % 8)
	}
	clearBit := func(buf []byte, bit int) {
		buf[bit/8] &^= 1 << (bit % 8)
	}

	// 3. Inode Bitmap blocks
	imapSize := imapBlocks * 1024
	imap := make([]byte, imapSize)
	for i := range imap {
		imap[i] = 0xff
	}
	for i := 1; i <= inodes; i++ {
		clearBit(imap, i)
	}
	setBit(imap, 1) // Mark root inode (1)
	if _, err := w.Write(imap); err != nil {
		return res, err
	}

	// 4. Zone Bitmap blocks
	zmapSize := zmapBlocks * 1024
	zmap := make([]byte, zmapSize)
	for i := range zmap {
		zmap[i] = 0xff
	}
	for i := firstDataZone; i < blocks; i++ {
		clearBit(zmap, i - firstDataZone + 1)
	}
	setBit(zmap, 1) // Mark first data block (root directory)
	if _, err := w.Write(zmap); err != nil {
		return res, err
	}

	// 5. Inode Table blocks
	inodeBlocks := (inodes * 32 + 1023) / 1024
	inodeTableSize := inodeBlocks * 1024
	inodeTable := make([]byte, inodeTableSize)
	
	dirEntrySize := namelen + 2
	rootInode := MinixInode{
		Mode:   040755, // Directory with 755 permissions
		UID:    0,
		Size:   uint32(2 * dirEntrySize), // Third entry has inode 0, size is 2 * dirEntrySize
		Time:   0,
		GID:    0,
		NLinks: 2,
	}
	rootInode.Zones[0] = uint16(firstDataZone)

	rootInodeBuf := new(bytes.Buffer)
	if err := binary.Write(rootInodeBuf, binary.LittleEndian, rootInode); err != nil {
		return res, err
	}
	copy(inodeTable[0:32], rootInodeBuf.Bytes())
	if _, err := w.Write(inodeTable); err != nil {
		return res, err
	}

	// 6. First Data Zone (Root Directory Entries)
	rootBlock := make([]byte, 1024)
	
	// Entry 1: "." (Inode 1)
	binary.LittleEndian.PutUint16(rootBlock[0:2], 1)
	copy(rootBlock[2:2+namelen], ".")

	// Entry 2: ".." (Inode 1)
	binary.LittleEndian.PutUint16(rootBlock[dirEntrySize:dirEntrySize+2], 1)
	copy(rootBlock[dirEntrySize+2:dirEntrySize+2+namelen], "..")

	// Entry 3: ".badblocks" (Inode 0)
	binary.LittleEndian.PutUint16(rootBlock[2*dirEntrySize:2*dirEntrySize+2], 0)
	copy(rootBlock[2*dirEntrySize+2:2*dirEntrySize+2+namelen], ".badblocks")

	if _, err := w.Write(rootBlock); err != nil {
		return res, err
	}

	// 7. Write remaining blocks (fully zero-filled zones)
	remBlocks := blocks - firstDataZone - 1
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
	blocks := 0
	if len(flags.Positional) > 1 {
		fmt.Sscanf(flags.Positional[1], "%d", &blocks)
	} else {
		if fi, serr := os.Stat(target); serr == nil {
			blocks = int(fi.Size() / 1024)
		}
		if blocks <= 0 {
			blocks = 1440 // default floppy size
		}
	}

	inodes := 0
	if val := flags.Get("i"); val != "" {
		fmt.Sscanf(val, "%d", &inodes)
	}

	namelen := 30
	if val := flags.Get("n"); val != "" {
		fmt.Sscanf(val, "%d", &namelen)
		if namelen != 14 && namelen != 30 {
			fmt.Fprintf(errOut, "mkfs.minix: illegal namelen %d\n", namelen)
			return 2
		}
	}

	jsonMode := flags.Has("json")

	// Open output file
	file, err := os.Create(target)
	if err != nil {
		fmt.Fprintf(errOut, "mkfs.minix: %v\n", err)
		return 1
	}
	defer file.Close()

	res, err := Run(file, blocks, inodes, namelen)
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
