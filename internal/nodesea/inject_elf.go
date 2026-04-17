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
	rawShoff := binary.LittleEndian.Uint64(data[eShoffOff:])
	rawPhentSize := binary.LittleEndian.Uint16(data[ePhentSizeOff:])
	rawPhnum := binary.LittleEndian.Uint16(data[ePhnumOff:])
	rawShentSize := binary.LittleEndian.Uint16(data[eShentSizeOff:])
	rawShnum := binary.LittleEndian.Uint16(data[eShnumOff:])
	rawShstrndx := binary.LittleEndian.Uint16(data[eShstrndxOff:])

	if rawShentSize != shentSize || rawPhentSize != phentSize {
		return fmt.Errorf("%w: unexpected ELF header sizes", ErrNotSupported)
	}

	out := make([]byte, len(data))
	copy(out, data)

	// Step 1: append note bytes.
	noteFileOff := uint64(len(out))
	out = append(out, note...)

	// Step 2: extend .shstrtab in place by appending the new section name
	// and rewriting the table at end. To keep file simple we'll just
	// **append** a fresh strtab containing original + new names and
	// repoint e_shstrndx-style; but rewriting all section sh_name fields
	// would be invasive. Instead: use the original .shstrtab and append
	// our new name to it directly (it lives in the file as a normal
	// section; we extend it by copying its bytes to a new location with
	// the additional name appended, then update its section header to
	// point at the new location & size).
	shstrSec := f.Sections[rawShstrndx]
	if shstrSec.Type != elf.SHT_STRTAB {
		return fmt.Errorf("%w: .shstrtab is not a STRTAB", ErrNotSupported)
	}
	origShStr, err := shstrSec.Data()
	if err != nil {
		return fmt.Errorf("nodesea: read .shstrtab: %w", err)
	}
	newSecName := ".note.node.sea\x00"
	newShStrOff := uint64(len(out))
	newShStr := make([]byte, 0, len(origShStr)+len(newSecName))
	newShStr = append(newShStr, origShStr...)
	newSecNameIdx := uint32(len(newShStr))
	newShStr = append(newShStr, newSecName...)
	out = append(out, newShStr...)

	// Step 3: copy section headers to a new area at EOF, append the new
	// note section header, and update e_shoff/e_shnum.
	oldShoff := int(rawShoff)
	oldShnum := int(rawShnum)
	if oldShoff+oldShnum*shentSize > len(data) {
		return fmt.Errorf("%w: section header table out of range", ErrNotSupported)
	}
	newShoff := uint64(len(out))
	out = append(out, data[oldShoff:oldShoff+oldShnum*shentSize]...)
	// Patch the .shstrtab section header (index = e_shstrndx) to point
	// at our extended copy.
	shstrIdx := int(rawShstrndx)
	shstrEntryOff := int(newShoff) + shstrIdx*shentSize
	binary.LittleEndian.PutUint64(out[shstrEntryOff+24:], newShStrOff)           // sh_offset
	binary.LittleEndian.PutUint64(out[shstrEntryOff+32:], uint64(len(newShStr))) // sh_size
	// Append a new section header for our note.
	noteShdr := make([]byte, shentSize)
	binary.LittleEndian.PutUint32(noteShdr[0:], newSecNameIdx)        // sh_name
	binary.LittleEndian.PutUint32(noteShdr[4:], uint32(elf.SHT_NOTE)) // sh_type
	binary.LittleEndian.PutUint64(noteShdr[8:], 0)                    // sh_flags
	binary.LittleEndian.PutUint64(noteShdr[16:], 0)                   // sh_addr
	binary.LittleEndian.PutUint64(noteShdr[24:], noteFileOff)         // sh_offset
	binary.LittleEndian.PutUint64(noteShdr[32:], uint64(len(note)))   // sh_size
	binary.LittleEndian.PutUint32(noteShdr[40:], 0)                   // sh_link
	binary.LittleEndian.PutUint32(noteShdr[44:], 0)                   // sh_info
	binary.LittleEndian.PutUint64(noteShdr[48:], 4)                   // sh_addralign
	binary.LittleEndian.PutUint64(noteShdr[56:], 0)                   // sh_entsize
	out = append(out, noteShdr...)
	newShnum := uint16(oldShnum + 1)

	// Step 4: copy program headers to a new area at EOF, append a new
	// PT_NOTE phdr, update e_phoff/e_phnum.
	oldPhoff := int(rawPhoff)
	oldPhnum := int(rawPhnum)
	if oldPhoff+oldPhnum*phentSize > len(data) {
		return fmt.Errorf("%w: program header table out of range", ErrNotSupported)
	}
	newPhoff := uint64(len(out))
	out = append(out, data[oldPhoff:oldPhoff+oldPhnum*phentSize]...)
	notePhdr := make([]byte, phentSize)
	binary.LittleEndian.PutUint32(notePhdr[0:], uint32(elf.PT_NOTE)) // p_type
	binary.LittleEndian.PutUint32(notePhdr[4:], uint32(elf.PF_R))    // p_flags
	binary.LittleEndian.PutUint64(notePhdr[8:], noteFileOff)         // p_offset
	binary.LittleEndian.PutUint64(notePhdr[16:], 0)                  // p_vaddr
	binary.LittleEndian.PutUint64(notePhdr[24:], 0)                  // p_paddr
	binary.LittleEndian.PutUint64(notePhdr[32:], uint64(len(note)))  // p_filesz
	binary.LittleEndian.PutUint64(notePhdr[40:], uint64(len(note)))  // p_memsz
	binary.LittleEndian.PutUint64(notePhdr[48:], 4)                  // p_align
	out = append(out, notePhdr...)
	newPhnum := uint16(oldPhnum + 1)

	// Patch ELF header.
	binary.LittleEndian.PutUint64(out[ePhoffOff:], newPhoff)
	binary.LittleEndian.PutUint64(out[eShoffOff:], newShoff)
	binary.LittleEndian.PutUint16(out[ePhnumOff:], newPhnum)
	binary.LittleEndian.PutUint16(out[eShnumOff:], newShnum)
	_ = ePhentSizeOff
	_ = eShentSizeOff
	_ = eShstrndxOff

	tmp := path + ".inject.tmp"
	if err := os.WriteFile(tmp, out, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return FlipSentinel(path)
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

// findElfNote scans every SHT_NOTE section in the file looking for a
// note matching name and type.
func findElfNote(f *elf.File, raw []byte, name string, ntype uint32) bool {
	for _, s := range f.Sections {
		if s.Type != elf.SHT_NOTE {
			continue
		}
		end := int(s.Offset + s.Size)
		if end > len(raw) {
			continue
		}
		data := raw[s.Offset:end]
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
