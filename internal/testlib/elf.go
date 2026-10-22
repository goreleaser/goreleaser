package testlib

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// WriteDynamicELF writes a minimal Linux ELF executable whose program
// interpreter is interpreter, such as "/lib/ld-linux-x86-64.so.2", to path.
// Only the headers are valid; the file is not meant to run.
func WriteDynamicELF(tb testing.TB, path, interpreter string) {
	tb.Helper()
	const (
		ehsize    = 64
		phentsize = 56
		phnum     = 1
		ptInterp  = 3
	)
	interp := append([]byte(interpreter), 0)
	var buf bytes.Buffer
	w := func(v any) {
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			tb.Fatal(err)
		}
	}
	buf.Write([]byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0}) // ELFCLASS64, little endian
	w(uint16(2))                                                               // ET_EXEC
	w(uint16(62))                                                              // EM_X86_64
	w(uint32(1))                                                               // EV_CURRENT
	w(uint64(0))                                                               // entry
	w(uint64(ehsize))                                                          // phoff
	w(uint64(0))                                                               // shoff
	w(uint32(0))                                                               // flags
	w(uint16(ehsize))                                                          // ehsize
	w(uint16(phentsize))                                                       // phentsize
	w(uint16(phnum))                                                           // phnum
	w(uint16(0))                                                               // shentsize
	w(uint16(0))                                                               // shnum
	w(uint16(0))                                                               // shstrndx
	w(uint32(ptInterp))                                                        // p_type
	w(uint32(4))                                                               // p_flags: PF_R
	w(uint64(ehsize + phentsize))                                              // p_offset
	w(uint64(0))                                                               // p_vaddr
	w(uint64(0))                                                               // p_paddr
	w(uint64(len(interp)))                                                     // p_filesz
	w(uint64(len(interp)))                                                     // p_memsz
	w(uint64(1))                                                               // p_align
	buf.Write(interp)
	if err := os.WriteFile(path, buf.Bytes(), 0o755); err != nil {
		tb.Fatal(err)
	}
}
