package split

import "strings"

func OnSlash(pair string) (string, string) {
	parts := strings.Split(pair, "/")
	return parts[0], parts[1]
}
