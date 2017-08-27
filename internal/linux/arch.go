// Package linux contains functions that are useful to generate linux packages.
package linux

import "strings"

// Arch converts a goarch to a linux-compatible arch
func Arch(key string) string {
	switch {
	case strings.Contains(key, "amd64"):
		return "amd64"
	case strings.Contains(key, "386"):
		return "i386"
	case strings.Contains(key, "arm64"):
		return "arm64"
	case strings.Contains(key, "arm6"):
		return "armhf"
	}
	return key
}
