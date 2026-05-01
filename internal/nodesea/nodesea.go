// Package nodesea implements the Node.js Single Executable
// Application (SEA) toolchain used by the experimental `node`
// builder.
//
// The toolchain shells out to a host-platform Node.js (≥ v25.5.0,
// LIEF-backed) once per build to invoke `node --build-sea
// sea-config.json`. That command injects the SEA blob into a copy of
// the per-target Node binary GoReleaser fetches from
// https://nodejs.org/dist (verifying SHA-256). On macOS targets the
// produced binary is ad-hoc signed via quill (pure-Go, host-OS
// independent) so the kernel loader will accept it. The package owns
// the cache layout, the download + verify path, and the `--build-sea`
// orchestration.
package nodesea

// Format identifies the container format of a Node.js host binary.
type Format int

const (
	// FormatELF is the Linux/BSD ELF container.
	FormatELF Format = iota + 1
	// FormatMachO is the Apple Mach-O container.
	FormatMachO
	// FormatPE is the Windows Portable Executable container.
	FormatPE
)

// String implements fmt.Stringer.
func (f Format) String() string {
	switch f {
	case FormatELF:
		return "elf"
	case FormatMachO:
		return "macho"
	case FormatPE:
		return "pe"
	default:
		return "unknown"
	}
}

// FormatFor returns the container format for a given GOOS string. It
// returns 0 (zero Format) for unsupported operating systems.
func FormatFor(goos string) Format {
	switch goos {
	case "linux":
		return FormatELF
	case "darwin":
		return FormatMachO
	case "windows":
		return FormatPE
	default:
		return 0
	}
}

