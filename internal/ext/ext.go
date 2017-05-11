package ext

import "strings"

func For(platform string) (ext string) {
	if strings.HasPrefix(platform, "windows") {
		ext = ".exe"
	}
	return
}
