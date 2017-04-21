package main

import (
	"log"
	"os"

	"github.com/goreleaser/goreleaser"
	"github.com/urfave/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var app = cli.NewApp()
	app.Name = "goreleaser"
	app.Version = version + ", commit " + commit + ", built at " + date
	app.Usage = "Deliver Go binaries as fast and easily as possible"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, file, c, f",
			Usage: "Load configuration from `FILE`",
			Value: "goreleaser.yml",
		},
		cli.StringFlag{
			Name:  "release-notes",
			Usage: "Load custom release notes from a markdown `FILE`",
		},
		cli.BoolFlag{
			Name:  "skip-validate",
			Usage: "Skip all the validations against the release",
		},
		cli.BoolFlag{
			Name:  "skip-publish",
			Usage: "Skip all publishing pipes of the release",
		},
	}
	app.Action = func(c *cli.Context) error {
		log.Printf("Running goreleaser %v\n", version)
		if err := goreleaser.Release(c); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
