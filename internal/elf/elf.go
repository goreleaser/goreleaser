// Package elf helps handling ELF files.
package elf

import (
	"debug/elf"
	"slices"
)

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

	return slices.ContainsFunc(f.Progs, func(prog *elf.Prog) bool {
		return prog.Type == elf.PT_INTERP
	})
}
