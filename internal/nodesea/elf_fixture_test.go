package nodesea

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
)

// elfBuilder produces a minimal ELF64-LE file with a few sections,
// including a SEA fuse sentinel inside a .text section. Just enough to
// exercise our injector.
type elfBuilder struct{}

func (elfBuilder) build() []byte {
	const (
		ehSize    = 64
		phSize    = 56
		shSize    = 64
		textBytes = 256
	)

	// Sections (in order): NULL, .text, .shstrtab.
	shstrTbl := []byte{0}
	textNameOff := uint32(len(shstrTbl))
	shstrTbl = append(shstrTbl, []byte(".text\x00")...)
	shstrNameOff := uint32(len(shstrTbl))
	shstrTbl = append(shstrTbl, []byte(".shstrtab\x00")...)

	// Layout:
	//   [0..ehSize)        header
	//   [ehSize..)          phdr (1 entry: PT_LOAD)
	//   [textOff..)         .text (256 bytes including sentinel)
	//   [shstrOff..)        .shstrtab
	//   [shoff..)           section header table
	phOff := uint64(ehSize)
	textOff := phOff + phSize
	shstrOff := textOff + textBytes
	shOff := shstrOff + uint64(len(shstrTbl))

	textBuf := make([]byte, textBytes)
	copy(textBuf[16:], []byte(SentinelStock))
	for i, b := range []byte(SentinelStock) {
		textBuf[16+i] = b
	}

	totalSize := shOff + 3*shSize
	buf := make([]byte, totalSize)

	// Header.
	copy(buf[0:], []byte{0x7f, 'E', 'L', 'F'})
	buf[elf.EI_CLASS] = byte(elf.ELFCLASS64)
	buf[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	buf[elf.EI_VERSION] = byte(elf.EV_CURRENT)
	buf[elf.EI_OSABI] = byte(elf.ELFOSABI_LINUX)
	binary.LittleEndian.PutUint16(buf[16:], uint16(elf.ET_EXEC))
	binary.LittleEndian.PutUint16(buf[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(buf[20:], uint32(elf.EV_CURRENT))
	binary.LittleEndian.PutUint64(buf[24:], 0x401000) // entry
	binary.LittleEndian.PutUint64(buf[0x20:], phOff)  // phoff
	binary.LittleEndian.PutUint64(buf[0x28:], shOff)  // shoff
	binary.LittleEndian.PutUint32(buf[0x30:], 0)      // flags
	binary.LittleEndian.PutUint16(buf[0x34:], ehSize) // ehsize
	binary.LittleEndian.PutUint16(buf[0x36:], phSize) // phentsize
	binary.LittleEndian.PutUint16(buf[0x38:], 1)      // phnum
	binary.LittleEndian.PutUint16(buf[0x3a:], shSize) // shentsize
	binary.LittleEndian.PutUint16(buf[0x3c:], 3)      // shnum
	binary.LittleEndian.PutUint16(buf[0x3e:], 2)      // shstrndx

	// Program header (PT_LOAD).
	binary.LittleEndian.PutUint32(buf[phOff+0:], uint32(elf.PT_LOAD))
	binary.LittleEndian.PutUint32(buf[phOff+4:], uint32(elf.PF_R|elf.PF_X))
	binary.LittleEndian.PutUint64(buf[phOff+8:], textOff)
	binary.LittleEndian.PutUint64(buf[phOff+16:], 0x400000) // vaddr
	binary.LittleEndian.PutUint64(buf[phOff+24:], 0x400000) // paddr
	binary.LittleEndian.PutUint64(buf[phOff+32:], textBytes)
	binary.LittleEndian.PutUint64(buf[phOff+40:], textBytes)
	binary.LittleEndian.PutUint64(buf[phOff+48:], 0x1000) // align

	// .text data.
	copy(buf[textOff:], textBuf)
	// .shstrtab data.
	copy(buf[shstrOff:], shstrTbl)

	// Section headers.
	writeSh := func(idx int, name uint32, typ elf.SectionType, off, size uint64, addralign uint64) {
		base := int(shOff) + idx*shSize
		binary.LittleEndian.PutUint32(buf[base+0:], name)
		binary.LittleEndian.PutUint32(buf[base+4:], uint32(typ))
		binary.LittleEndian.PutUint64(buf[base+8:], 0)  // flags
		binary.LittleEndian.PutUint64(buf[base+16:], 0) // addr
		binary.LittleEndian.PutUint64(buf[base+24:], off)
		binary.LittleEndian.PutUint64(buf[base+32:], size)
		binary.LittleEndian.PutUint32(buf[base+40:], 0)
		binary.LittleEndian.PutUint32(buf[base+44:], 0)
		binary.LittleEndian.PutUint64(buf[base+48:], addralign)
		binary.LittleEndian.PutUint64(buf[base+56:], 0)
	}
	// idx 0: NULL section
	writeSh(0, 0, elf.SHT_NULL, 0, 0, 0)
	// idx 1: .text
	writeSh(1, textNameOff, elf.SHT_PROGBITS, textOff, textBytes, 16)
	// idx 2: .shstrtab
	writeSh(2, shstrNameOff, elf.SHT_STRTAB, shstrOff, uint64(len(shstrTbl)), 1)

	// Sanity: parse it.
	if _, err := elf.NewFile(bytes.NewReader(buf)); err != nil {
		panic(err)
	}
	return buf
}
