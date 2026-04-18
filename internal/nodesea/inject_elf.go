package nodesea

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// ErrAlreadyInjected is returned when InjectELF / InjectMachO / InjectPE
// detects that a SEA blob has already been added to the host binary.
var ErrAlreadyInjected = errors.New("nodesea: blob already injected")

// noteName is the postject-api.h note name used to look up the SEA blob
// at runtime via dl_iterate_phdr on Linux.
const noteName = "NODE_SEA_BLOB"

// noteType is the postject-api.h note type (ASCII "POST" little-endian)
// used to identify SEA notes.
const noteType uint32 = 0x4F575354 // 'P' 'O' 'S' 'T'

// InjectELF injects blob as an ELF note (postject-style, reachable via
// PT_NOTE) into the file at path, then flips the SEA fuse sentinel.
//
// The implementation appends a new SHT_NOTE section at the end of the
// file, a corresponding section header, and a new PT_NOTE program header
// pointing at it. It rewrites the ELF/section/program headers in place
// to reflect the new layout.
//
// Limitations: ELFCLASS64, little-endian only. Returns
// ErrAlreadyInjected if a NODE_SEA_BLOB note is already present.
func InjectELF(path string, blob []byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if len(data) < 0x40 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%w: not an ELF file", ErrNotSupported)
	}
	if data[elf.EI_CLASS] != byte(elf.ELFCLASS64) {
		return fmt.Errorf("%w: only ELFCLASS64 is supported", ErrNotSupported)
	}
	if data[elf.EI_DATA] != byte(elf.ELFDATA2LSB) {
		return fmt.Errorf("%w: only little-endian ELF is supported", ErrNotSupported)
	}

	f, err := elf.NewFile(newReadSeeker(data))
	if err != nil {
		return fmt.Errorf("nodesea: parse ELF: %w", err)
	}
	defer f.Close()

	// Idempotency: refuse to re-inject.
	if findElfNote(f, data, noteName, noteType) {
		return ErrAlreadyInjected
	}

	note := buildNote(noteName, noteType, blob)

	// We append:
	//   1) the note bytes,
	//   2) a new section name appended to .shstrtab (so we copy + extend
	//      the existing strtab),
	//   3) a new section header,
	//   4) a new PT_NOTE program header (we move/extend the program
	//      header table to a fresh area at EOF).
	//
	// ELF64 layout offsets in the header:
	//   e_phoff   @ 0x20  (uint64)
	//   e_shoff   @ 0x28  (uint64)
	//   e_phentsize @ 0x36 (uint16)
	//   e_phnum   @ 0x38  (uint16)
	//   e_shentsize @ 0x3a (uint16)
	//   e_shnum   @ 0x3c  (uint16)
	//   e_shstrndx @ 0x3e (uint16)
	const (
		ePhoffOff     = 0x20
		eShoffOff     = 0x28
		ePhentSizeOff = 0x36
		ePhnumOff     = 0x38
		eShentSizeOff = 0x3a
		eShnumOff     = 0x3c
		eShstrndxOff  = 0x3e

		shentSize = 64
		phentSize = 56
	)

	// Read header values directly from raw bytes (debug/elf hides them).
	rawPhoff := binary.LittleEndian.Uint64(data[ePhoffOff:])
	rawPhentSize := binary.LittleEndian.Uint16(data[ePhentSizeOff:])
	rawPhnum := binary.LittleEndian.Uint16(data[ePhnumOff:])
	rawShentSize := binary.LittleEndian.Uint16(data[eShentSizeOff:])

	if rawShentSize != shentSize || rawPhentSize != phentSize {
		return fmt.Errorf("%w: unexpected ELF header sizes", ErrNotSupported)
	}

	// Compute a new vaddr above the highest existing PT_LOAD's vmaddr
	// end, page-aligned. The note must live inside a PT_LOAD so that
	// dl_iterate_phdr-based lookup (postject runtime) can read it via
	// `base_addr + phdr->p_vaddr`. We'll create a fresh PT_LOAD that
	// covers the new program-header table + the note bytes.
	const pageSize = uint64(0x1000)
	var maxLoadEnd uint64
	for _, p := range f.Progs {
		if p.Type != elf.PT_LOAD {
			continue
		}
		end := p.Vaddr + p.Memsz
		if end > maxLoadEnd {
			maxLoadEnd = end
		}
	}
	newVaddr := alignUp64(maxLoadEnd+pageSize, pageSize)

	out := make([]byte, len(data))
	copy(out, data)

	// Pad to a page boundary so file_off aligns with vaddr (mod
	// pageSize), as required by ELF (p_offset ≡ p_vaddr (mod p_align)).
	for uint64(len(out))%pageSize != 0 {
		out = append(out, 0)
	}
	newFileOff := uint64(len(out))

	// New layout at newFileOff:
	//   [new program-header table (oldPhnum + 2 entries)]
	//   [note bytes]
	newPhnum := rawPhnum + 2
	newPhdrSize := uint64(newPhnum) * phentSize
	noteOffsetInBlock := newPhdrSize

	// Append the original program headers at newFileOff (we'll add the
	// new PT_LOAD and PT_NOTE entries below).
	oldPhoff := int(rawPhoff)
	oldPhnum := int(rawPhnum)
	if oldPhoff+oldPhnum*phentSize > len(data) {
		return fmt.Errorf("%w: program header table out of range", ErrNotSupported)
	}
	out = append(out, data[oldPhoff:oldPhoff+oldPhnum*phentSize]...)

	// Append two empty phdr slots (PT_LOAD then PT_NOTE) — we'll fill
	// them in below.
	out = append(out, make([]byte, 2*phentSize)...)

	// Append the note bytes.
	noteFileOff := newFileOff + noteOffsetInBlock
	out = append(out, note...)

	blockSize := uint64(len(out)) - newFileOff

	// Walk the (now copied) program headers at newFileOff and update the
	// PHDR self-entry to point at the new location. Without this fix-up
	// the dynamic linker reads stale phdr metadata at the old vaddr.
	for i := range oldPhnum {
		entry := int(newFileOff) + i*phentSize
		ptype := elf.ProgType(binary.LittleEndian.Uint32(out[entry:]))
		if ptype != elf.PT_PHDR {
			continue
		}
		binary.LittleEndian.PutUint64(out[entry+8:], newFileOff)   // p_offset
		binary.LittleEndian.PutUint64(out[entry+16:], newVaddr)    // p_vaddr
		binary.LittleEndian.PutUint64(out[entry+24:], newVaddr)    // p_paddr
		binary.LittleEndian.PutUint64(out[entry+32:], newPhdrSize) // p_filesz
		binary.LittleEndian.PutUint64(out[entry+40:], newPhdrSize) // p_memsz
	}

	// Fill in the new PT_LOAD entry (covers the entire new block).
	loadEntry := int(newFileOff) + oldPhnum*phentSize
	binary.LittleEndian.PutUint32(out[loadEntry:], uint32(elf.PT_LOAD))
	binary.LittleEndian.PutUint32(out[loadEntry+4:], uint32(elf.PF_R))
	binary.LittleEndian.PutUint64(out[loadEntry+8:], newFileOff) // p_offset
	binary.LittleEndian.PutUint64(out[loadEntry+16:], newVaddr)  // p_vaddr
	binary.LittleEndian.PutUint64(out[loadEntry+24:], newVaddr)  // p_paddr
	binary.LittleEndian.PutUint64(out[loadEntry+32:], blockSize) // p_filesz
	binary.LittleEndian.PutUint64(out[loadEntry+40:], blockSize) // p_memsz
	binary.LittleEndian.PutUint64(out[loadEntry+48:], pageSize)  // p_align

	// Fill in the new PT_NOTE entry (covers just the note bytes within
	// the new PT_LOAD).
	noteEntry := loadEntry + phentSize
	noteVaddr := newVaddr + noteOffsetInBlock
	binary.LittleEndian.PutUint32(out[noteEntry:], uint32(elf.PT_NOTE))
	binary.LittleEndian.PutUint32(out[noteEntry+4:], uint32(elf.PF_R))
	binary.LittleEndian.PutUint64(out[noteEntry+8:], noteFileOff)        // p_offset
	binary.LittleEndian.PutUint64(out[noteEntry+16:], noteVaddr)         // p_vaddr
	binary.LittleEndian.PutUint64(out[noteEntry+24:], noteVaddr)         // p_paddr
	binary.LittleEndian.PutUint64(out[noteEntry+32:], uint64(len(note))) // p_filesz
	binary.LittleEndian.PutUint64(out[noteEntry+40:], uint64(len(note))) // p_memsz
	binary.LittleEndian.PutUint64(out[noteEntry+48:], 4)                 // p_align

	// Patch ELF header: e_phoff and e_phnum. Leave section header
	// table alone — sections aren't needed at runtime.
	binary.LittleEndian.PutUint64(out[ePhoffOff:], newFileOff)
	binary.LittleEndian.PutUint16(out[ePhnumOff:], newPhnum)
	_ = ePhentSizeOff
	_ = eShentSizeOff
	_ = eShstrndxOff
	_ = eShoffOff
	_ = eShnumOff
	_ = shentSize

	tmp := path + ".inject.tmp"
	if err := os.WriteFile(tmp, out, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return FlipSentinel(path)
}

func alignUp64(v, align uint64) uint64 {
	return (v + align - 1) &^ (align - 1)
}

// buildNote serializes a single ELF note record.
func buildNote(name string, ntype uint32, desc []byte) []byte {
	nameBytes := append([]byte(name), 0) // NUL-terminate
	var buf bytes.Buffer
	header := struct {
		NameSize uint32
		DescSize uint32
		Type     uint32
	}{uint32(len(nameBytes)), uint32(len(desc)), ntype}
	_ = binary.Write(&buf, binary.LittleEndian, header)
	buf.Write(nameBytes)
	for buf.Len()%4 != 0 {
		buf.WriteByte(0)
	}
	buf.Write(desc)
	for buf.Len()%4 != 0 {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

// findElfNote scans every PT_NOTE program header in the file looking
// for a note matching name and type. We use program headers rather than
// section headers because postject's runtime lookup
// (postject_find_resource on Linux) walks PT_NOTE phdrs via
// dl_iterate_phdr and we mirror the same convention.
func findElfNote(f *elf.File, raw []byte, name string, ntype uint32) bool {
	for _, p := range f.Progs {
		if p.Type != elf.PT_NOTE {
			continue
		}
		end := int(p.Off + p.Filesz)
		if end > len(raw) {
			continue
		}
		data := raw[p.Off:end]
		for len(data) >= 12 {
			namesz := binary.LittleEndian.Uint32(data[0:4])
			descsz := binary.LittleEndian.Uint32(data[4:8])
			t := binary.LittleEndian.Uint32(data[8:12])
			data = data[12:]
			padded := func(n uint32) uint32 { return (n + 3) &^ 3 }
			if uint32(len(data)) < padded(namesz)+padded(descsz) {
				break
			}
			gotName := string(bytes.TrimRight(data[:namesz], "\x00"))
			if gotName == name && t == ntype {
				return true
			}
			data = data[padded(namesz)+padded(descsz):]
		}
	}
	return false
}
