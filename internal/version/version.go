// Package version validates versions before they are embedded in generated files.
package version

import (
	"fmt"
	"regexp"
)

var safe = regexp.MustCompile(`^[0-9A-Za-z._+-]*$`)

// Check returns an error if the version has characters that are not safe to
// embed in generated files.
func Check(version string) error {
	if !safe.MatchString(version) {
		return fmt.Errorf("version %q has invalid characters: only letters, digits, '.', '_', '+', and '-' are allowed", version)
	}
	return nil
}
