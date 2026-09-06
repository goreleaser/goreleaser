// Package buildtarget contains shared build target normalization helpers.
package buildtarget

// Goarch returns the Go architecture name for common non-Go target
// architecture names.
func Goarch(s string) string {
	switch s {
	case "aarch64":
		return "arm64"
	case "x86_64":
		return "amd64"
	case "i386", "i586", "i686", "x86":
		return "386"
	case "loongarch64":
		return "loong64"
	case "powerpc":
		return "ppc"
	case "powerpc64":
		return "ppc64"
	case "powerpc64le":
		return "ppc64le"
	case "riscv64", "riscv64gc":
		return "riscv64"
	case "s390x":
		return "s390x"
	case "arm", "armv7":
		return "arm"
	case "wasm32":
		return "wasm"
	default:
		return s
	}
}
