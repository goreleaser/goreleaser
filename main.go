package main

import (
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/cmd"
)

func main() {
	// enable colored output on travis
	if os.Getenv("CI") != "" {
		color.NoColor = false
	}

	log.SetHandler(cli.Default)

	cmd.Execute()
}
