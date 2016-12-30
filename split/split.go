package split

import "strings"

// OnSlash split a string on / and return the first 2 parts
func OnSlash(pair string) (string, string) {
	parts := strings.Split(pair, "/")
	return parts[0], parts[1]
}
