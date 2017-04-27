package main

import (
	"fmt"
	"log"
	"os"

	"github.com/goreleaser/goreleaser/goreleaserlib"
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
	}
	app.Action = func(c *cli.Context) error {
		log.Printf("Running goreleaser %v\n", version)
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
				err := goreleaserlib.InitProject()
				if err != nil {
					return err
				}

				log.Printf("goreleaser.yml created. Please edit accordingly to your needs.")
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
