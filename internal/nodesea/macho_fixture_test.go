package nodesea

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
)

// machoBuilder is a tiny helper to assemble synthetic 64-bit Mach-O
// fixtures for tests. It is *not* a general-purpose Mach-O writer; it
// produces just enough structure for our LC walker to operate on.
type machoBuilder struct {
	// One __TEXT segment + one __LINKEDIT segment + optional
	// LC_CODE_SIGNATURE.
	textData       []byte
	linkeditData   []byte
	signatureBytes []byte // if nil, no LC_CODE_SIGNATURE
	// Extra slack between LC area and __TEXT data; useful for testing
	// injection which needs room to grow the LC region in place.
	slack int
}

func (b *machoBuilder) build() []byte {
	const (
		hdrSize    = 32
		seg64Size  = 72
		codeSigLen = 16
	)

	ncmds := uint32(2) // __TEXT, __LINKEDIT
	sizeofcmds := uint32(2 * seg64Size)
	if b.signatureBytes != nil {
		ncmds++
		sizeofcmds += codeSigLen
	}

	// File layout:
	//   header
	//   load commands
	//   __TEXT data (placed right after LCs, fileoff = hdrSize+sizeofcmds)
	//   __LINKEDIT data
	//   signature
	textFileoff := uint64(hdrSize) + uint64(sizeofcmds) + uint64(b.slack)
	textFilesize := uint64(len(b.textData))
	linkFileoff := textFileoff + textFilesize
	linkFilesize := uint64(len(b.linkeditData)) + uint64(len(b.signatureBytes))
	sigFileoff := linkFileoff + uint64(len(b.linkeditData))

	var buf bytes.Buffer
	// Mach-O header (mach_header_64).
	_ = binary.Write(&buf, binary.LittleEndian, macho.Magic64)     // magic
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0x100000c)) // cputype = ARM64
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))         // cpusubtype
	_ = binary.Write(&buf, binary.LittleEndian, uint32(2))         // filetype = MH_EXECUTE
	_ = binary.Write(&buf, binary.LittleEndian, ncmds)
	_ = binary.Write(&buf, binary.LittleEndian, sizeofcmds)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // flags
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // reserved

	writeSeg := func(name string, fileoff, filesize uint64) {
		_ = binary.Write(&buf, binary.LittleEndian, uint32(macho.LoadCmdSegment64))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(seg64Size))
		nameBuf := make([]byte, 16)
		copy(nameBuf, name)
		buf.Write(nameBuf)
		_ = binary.Write(&buf, binary.LittleEndian, uint64(0)) // vmaddr
		_ = binary.Write(&buf, binary.LittleEndian, filesize)  // vmsize (close enough)
		_ = binary.Write(&buf, binary.LittleEndian, fileoff)
		_ = binary.Write(&buf, binary.LittleEndian, filesize)
		_ = binary.Write(&buf, binary.LittleEndian, uint32(7)) // maxprot
		_ = binary.Write(&buf, binary.LittleEndian, uint32(7)) // initprot
		_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // nsects
		_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // flags
	}

	writeSeg("__TEXT", textFileoff, textFilesize)
	writeSeg("__LINKEDIT", linkFileoff, linkFilesize)

	if b.signatureBytes != nil {
		_ = binary.Write(&buf, binary.LittleEndian, uint32(0x1d)) // LC_CODE_SIGNATURE
		_ = binary.Write(&buf, binary.LittleEndian, uint32(codeSigLen))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(sigFileoff))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(b.signatureBytes)))
	}

	if b.slack > 0 {
		buf.Write(make([]byte, b.slack))
	}

	buf.Write(b.textData)
	buf.Write(b.linkeditData)
	if b.signatureBytes != nil {
		buf.Write(b.signatureBytes)
	}
	return buf.Bytes()
}
