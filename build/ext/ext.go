package ext

import "github.com/goreleaser/goreleaser/build/buildtarget"

// For returns the binary extension for the given platform
func For(target buildtarget.Target) string {
	if target.OS == "windows" {
		return ".exe"
	}
	return ""
}
