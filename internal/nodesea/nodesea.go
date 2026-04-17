// Package nodesea implements a pure-Go toolchain for building Node.js
// Single Executable Applications (SEAs).
//
// It performs every step that traditionally requires the upstream
// `postject` npm package and other helper tooling:
//
//  1. Downloading and SHA-256-verifying the official Node.js host binary
//     for a given target from https://nodejs.org/dist.
//  2. Stripping the existing code signature from Mach-O and PE binaries so
//     a SEA blob can be injected without invalidating the binary.
//  3. Injecting a SEA blob into ELF, Mach-O and PE binaries by adding a
//     dedicated section / segment / resource and flipping the
//     `NODE_SEA_FUSE_…` sentinel byte.
//
// The only external runtime requirement is `node` itself (>= 22), which
// is still needed to generate the SEA blob from the user's
// `sea-config.json` via `node --experimental-sea-config`.
package nodesea

// Sentinel is the literal "fuse" string Node.js (and postject-compatible
// hosts) embed in the host binary to advertise SEA support. In a stock
// host the byte immediately following the sentinel is the ASCII colon
// `:` followed by ASCII `0`; injecting a blob flips that trailing `0`
// to `1` so Node.js knows a blob is attached.
//
// The full token in the binary therefore looks like:
//
//	NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2:0   (stock)
//	NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2:1   (after injection)
//
// See https://nodejs.org/api/single-executable-applications.html and
// https://github.com/nodejs/postject (search for "POSTJECT_SENTINEL").
const Sentinel = "NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2"

// SentinelStock is the full sentinel token as it appears in an unmodified
// host binary.
const SentinelStock = Sentinel + ":0"

// SentinelFused is the full sentinel token after a successful blob
// injection.
const SentinelFused = Sentinel + ":1"

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
