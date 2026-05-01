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

// supportedGoos reports whether the given GOOS has a SEA injector.
func supportedGoos(goos string) bool {
	switch goos {
	case "linux", "darwin", "windows":
		return true
	}
	return false
}
