package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("internal/pipe/gentoo/gentoo.go")
	s := string(b)

	searchMatch := `
			for _, dv := range deletedVersions {
				if strings.Contains(filename, dv) {
					removed = true
					break
				}
			}`

	replaceMatch := `
			for _, dv := range deletedVersions {
				if idx := strings.Index(filename, dv); idx != -1 {
					isMatch := true
					if idx > 0 && filename[idx-1] != '_' && filename[idx-1] != '-' {
						isMatch = false
					}
					endIdx := idx + len(dv)
					if endIdx < len(filename) {
						next := filename[endIdx]
						if next == '.' {
							if endIdx+1 < len(filename) && filename[endIdx+1] >= '0' && filename[endIdx+1] <= '9' {
								isMatch = false
							}
						} else if next != '_' && next != '-' {
							isMatch = false
						}
					}
					if isMatch {
						removed = true
						break
					}
				}
			}`

	s = strings.Replace(s, searchMatch, replaceMatch, 1)
	os.WriteFile("internal/pipe/gentoo/gentoo.go", []byte(s), 0644)
	fmt.Println("Applied match fix")
}
