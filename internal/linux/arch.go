// Package linux contains functions that are useful to generate linux packages.
package linux

import "strings"

// Arch converts a goarch to a linux-compatible arch.
func Arch(key string) string {
	// XXX: list of all linux arches: `go tool dist list | grep linux`
	var arch = strings.TrimPrefix(key, "linux")
	switch arch {
	case "386":
		return "i386"
	case "amd64":
		return "amd64"
	case "arm5": // GOARCH + GOARM
		return "armel"
	case "arm6": // GOARCH + GOARM
		return "armhf"
	case "arm7": // GOARCH + GOARM
		return "armhf"
	default:
		return arch
	}
}
