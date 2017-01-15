package main

import (
	"log"
	"os"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/goreleaser/goreleaser/pipeline/archive"
	"github.com/goreleaser/goreleaser/pipeline/brew"
	"github.com/goreleaser/goreleaser/pipeline/build"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/goreleaser/goreleaser/pipeline/env"
	"github.com/goreleaser/goreleaser/pipeline/git"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/goreleaser/goreleaser/pipeline/repos"
	"github.com/urfave/cli"
)

var version = "master"

var pipes = []pipeline.Pipe{
	// load data, set defaults, etc...
	defaults.Pipe{},
	env.Pipe{},
	git.Pipe{},
	repos.Pipe{},

	// real work
	build.Pipe{},
	archive.Pipe{},
	release.Pipe{},
	brew.Pipe{},
}

func main() {
	var app = cli.NewApp()
	app.Name = "release"
	app.Version = version
	app.Usage = "Deliver Go binaries as fast and easily as possible"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load configuration from `FILE`",
			Value: "goreleaser.yml",
		},
	}
	app.Action = func(c *cli.Context) (err error) {
		var file = c.String("config")
		cfg, err := config.Load(file)
		// Allow failing to load the config file if file is not explicitly specified
		if err != nil && c.IsSet("config") {
			return cli.NewExitError(err.Error(), 1)
		}
		context := context.New(cfg)
		log.SetFlags(0)
		for _, pipe := range pipes {
			log.Println(pipe.Description())
			log.SetPrefix(" -> ")
			if err := pipe.Run(context); err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			log.SetPrefix("")
		}
		log.Println("Done!")
		return
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
