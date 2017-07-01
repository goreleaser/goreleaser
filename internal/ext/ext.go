package ext

import "strings"

// For returns the binary extension for the given platform
func For(platform string) (ext string) {
	if strings.HasPrefix(platform, "windows") {
		ext = ".exe"
	}
	return
}
