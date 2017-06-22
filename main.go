package main

import (
	"fmt"
	"os"

	"github.com/apex/log"
	lcli "github.com/apex/log/handlers/text"
	"github.com/goreleaser/goreleaser/goreleaserlib"
	"github.com/urfave/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	log.SetHandler(lcli.New(os.Stdout))
}

func main() {
	var app = cli.NewApp()
	app.Name = "goreleaser"
	app.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)
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
		cli.BoolFlag{
			Name:  "snapshot",
			Usage: "Generate an unversioned snapshot release",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode",
		},
	}
	app.Action = func(c *cli.Context) error {
		log.Infof("Running goreleaser %v", version)
		if err := goreleaserlib.Release(c); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:    "init",
			Aliases: []string{"i"},
			Usage:   "generate goreleaser.yml",
			Action: func(c *cli.Context) error {
				var filename = "goreleaser.yml"
				if err := goreleaserlib.InitProject(filename); err != nil {
					return cli.NewExitError(err.Error(), 1)
				}

				log.Infof("%s created. Please edit accordingly to your needs.", filename)
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.WithError(err).Fatal("Failed")
	}
}
