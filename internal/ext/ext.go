package ext

import "github.com/goreleaser/goreleaser/internal/buildtarget"

// For returns the binary extension for the given platform
func For(target buildtarget.Target) (ext string) {
	return ForOS(target.OS)
}

func ForOS(os string) string {
	if os == "windows" {
		return ".exe"
	}
	return ""
}
