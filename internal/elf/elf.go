// Package elf helps handling ELF files.
package elf

import "debug/elf"

// IsDynamicallyLinked checks if the given binary is dynamically linked.
// It returns true if the binary is an ELF file with a PT_INTERP segment,
// which indicates it needs a dynamic linker.
// For non-ELF files (e.g., macOS, Windows), it returns false.
func IsDynamicallyLinked(path string) bool {
	f, err := elf.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	for _, prog := range f.Progs {
		if prog.Type == elf.PT_INTERP {
			return true
		}
	}
	return false
}
