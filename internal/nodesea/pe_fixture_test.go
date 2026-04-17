package nodesea

import (
	"bytes"
	"encoding/binary"
)

// peBuilder assembles a minimal valid PE32+ file just realistic enough
// for our checksum + cert-table logic to exercise. Not a general PE
// writer.
type peBuilder struct {
	cert     []byte // appended Authenticode cert; nil = unsigned
	rsrcSize int    // if >0, append a .rsrc section of this size
	textData []byte // if non-empty, .text section payload
}

func (b *peBuilder) build() []byte {
	const (
		dosStubLen     = 0x40
		peSigLen       = 4
		fileHeaderLen  = 20
		optHeaderLen   = 240 // PE32+: 112 (fixed) + 16*8 (data dirs)
		dataDirCount   = 16
		sectionHdrLen  = 40
		sectionDataLen = 0x200
	)

	sectionHdrCnt := uint16(1)
	if b.rsrcSize > 0 {
		sectionHdrCnt = 2
	}

	totalLen := dosStubLen + peSigLen + fileHeaderLen + optHeaderLen +
		sectionHdrLen*int(sectionHdrCnt) + sectionDataLen
	if b.rsrcSize > 0 {
		totalLen += b.rsrcSize
	}

	buf := make([]byte, totalLen)
	// DOS header: MZ + e_lfanew at 0x3c.
	buf[0] = 'M'
	buf[1] = 'Z'
	binary.LittleEndian.PutUint32(buf[0x3c:], dosStubLen)

	off := dosStubLen
	// PE signature.
	copy(buf[off:], "PE\x00\x00")
	off += peSigLen

	// IMAGE_FILE_HEADER.
	binary.LittleEndian.PutUint16(buf[off+0:], 0x8664)        // machine = AMD64
	binary.LittleEndian.PutUint16(buf[off+2:], sectionHdrCnt) // numberOfSections
	binary.LittleEndian.PutUint16(buf[off+16:], optHeaderLen) // sizeOfOptionalHeader
	binary.LittleEndian.PutUint16(buf[off+18:], 0x22)         // characteristics
	off += fileHeaderLen

	// IMAGE_OPTIONAL_HEADER64.
	optStart := off
	binary.LittleEndian.PutUint16(buf[off:], 0x20b)            // PE32+ magic
	binary.LittleEndian.PutUint32(buf[off+32:], 0x1000)        // SectionAlignment
	binary.LittleEndian.PutUint32(buf[off+36:], 0x200)         // FileAlignment
	binary.LittleEndian.PutUint32(buf[off+108:], dataDirCount) // NumberOfRvaAndSizes
	off += optHeaderLen

	// Section header (.text).
	textOff := off
	copy(buf[textOff:], ".text\x00\x00\x00")
	binary.LittleEndian.PutUint32(buf[textOff+8:], sectionDataLen)  // VirtualSize
	binary.LittleEndian.PutUint32(buf[textOff+12:], 0x1000)         // VirtualAddress
	binary.LittleEndian.PutUint32(buf[textOff+16:], sectionDataLen) // SizeOfRawData
	textRawOff := uint32(dosStubLen + peSigLen + fileHeaderLen + optHeaderLen + sectionHdrLen*int(sectionHdrCnt))
	binary.LittleEndian.PutUint32(buf[textOff+20:], textRawOff)
	binary.LittleEndian.PutUint32(buf[textOff+36:], 0x60000020) // characteristics
	off += sectionHdrLen
	if len(b.textData) > 0 {
		copy(buf[textRawOff:], b.textData)
	}

	// Section header (.rsrc).
	if b.rsrcSize > 0 {
		rsrcSh := off
		copy(buf[rsrcSh:], ".rsrc\x00\x00\x00")
		binary.LittleEndian.PutUint32(buf[rsrcSh+8:], uint32(b.rsrcSize))  // VirtualSize
		binary.LittleEndian.PutUint32(buf[rsrcSh+12:], 0x2000)             // VirtualAddress
		binary.LittleEndian.PutUint32(buf[rsrcSh+16:], uint32(b.rsrcSize)) // SizeOfRawData
		rsrcRawOff := textRawOff + sectionDataLen
		binary.LittleEndian.PutUint32(buf[rsrcSh+20:], rsrcRawOff)
		binary.LittleEndian.PutUint32(buf[rsrcSh+36:], 0x40000040) // INITIALIZED_DATA | READ
		// DataDirectory[2] (RESOURCE).
		dirOff := optStart + 112
		binary.LittleEndian.PutUint32(buf[dirOff+2*8:], 0x2000)
		binary.LittleEndian.PutUint32(buf[dirOff+2*8+4:], uint32(b.rsrcSize))
		// Write a minimal valid empty resource directory at rsrcRawOff.
		// IMAGE_RESOURCE_DIRECTORY: 16 bytes all zeros = 0 named, 0 ID
		// entries — this parses fine.
		// (already zero from make([]byte, totalLen))
	}

	// Append cert if present and write its dirent.
	if b.cert != nil {
		certOff := uint32(len(buf))
		buf = append(buf, b.cert...)
		dirOff := optStart + 112
		secEntryOff := dirOff + 4*8
		binary.LittleEndian.PutUint32(buf[secEntryOff:], certOff)
		binary.LittleEndian.PutUint32(buf[secEntryOff+4:], uint32(len(b.cert)))
	}
	return buf
}

// peSecurityDir returns (va, size) of the IMAGE_DIRECTORY_ENTRY_SECURITY
// entry from raw PE bytes.
func peSecurityDir(data []byte) (uint32, uint32) {
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	optStart := peOff + 24
	dirOff := optStart + 112
	secEntryOff := dirOff + 4*8
	va := binary.LittleEndian.Uint32(data[secEntryOff : secEntryOff+4])
	size := binary.LittleEndian.Uint32(data[secEntryOff+4 : secEntryOff+8])
	return va, size
}

// peComputedChecksum recomputes the checksum on raw PE bytes (for
// verifying our writer wrote a self-consistent value).
func peComputedChecksum(data []byte) (stored, computed uint32) {
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	optStart := peOff + 24
	checksumOff := optStart + 64
	stored = binary.LittleEndian.Uint32(data[checksumOff : checksumOff+4])
	zeroed := bytes.Clone(data)
	for i := checksumOff; i < checksumOff+4; i++ {
		zeroed[i] = 0
	}
	computed = peChecksum(zeroed, checksumOff)
	return
}
