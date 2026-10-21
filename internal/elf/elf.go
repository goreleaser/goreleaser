// Package elf helps handling ELF files.
package elf

import (
	"bytes"
	"debug/elf"
	"io"
	"slices"
	"strings"
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

// Libc reports the libc family selected by an ELF program interpreter. It
// returns an empty string for statically linked, non-ELF, and unknown dynamic
// binaries so callers do not guess a compatible runtime.
func Libc(path string) string {
	return libcFromInterpreter(interpreter(path))
}

func interpreter(path string) string {
	f, err := elf.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		b, err := io.ReadAll(prog.Open())
		if err != nil {
			return ""
		}
		return string(bytes.TrimRight(b, "\x00"))
	}
	return ""
}

func libcFromInterpreter(interpreter string) string {
	name := strings.ToLower(interpreter)
	switch {
	case strings.Contains(name, "ld-musl-"):
		return "musl"
	case strings.Contains(name, "ld-linux"):
		return "gnu"
	default:
		return ""
	}
}
