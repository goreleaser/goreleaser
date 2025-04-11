// Package experimental guards experimental features.
package experimental

import (
	"os"
	"slices"
	"strings"
)

const (
	envKey         = "GORELEASER_EXPERIMENTAL"
	defaultGOARMv7 = "defaultgoarm"
)

// DefaultGOARM considers the `defaultgoarm` experiment and returns the correct
// value.
func DefaultGOARM() string {
	if has(defaultGOARMv7) {
		return "7"
	}
	return "6"
}

func has(e string) bool {
	experiments := strings.Split(os.Getenv(envKey), ",")
	return slices.Contains(experiments, e)
}
