// Package deprecate provides simple functions to standardize the output
// of deprecation notices on goreleaser
package deprecate

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
)

// Notice warns the user about the deprecation of the given property
func Notice(property string) {
	cli.Default.Padding += 3
	defer func() {
		cli.Default.Padding -= 3
	}()
	url := "https://goreleaser.com/#deprecation_notices." + property
	log.Warn(color.New(color.Bold, color.FgHiYellow).Sprintf(
		"DEPRECATED: `%s` should not be used anymore, check %s for more info.",
		property,
		url,
	))
}
