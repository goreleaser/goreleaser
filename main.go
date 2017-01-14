package main

import (
	"log"
	"os"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/context"
	"github.com/goreleaser/releaser/pipeline"
	"github.com/goreleaser/releaser/pipeline/brew"
	"github.com/goreleaser/releaser/pipeline/build"
	"github.com/goreleaser/releaser/pipeline/cleanup"
	"github.com/goreleaser/releaser/pipeline/compress"
	"github.com/goreleaser/releaser/pipeline/defaults"
	"github.com/goreleaser/releaser/pipeline/env"
	"github.com/goreleaser/releaser/pipeline/git"
	"github.com/goreleaser/releaser/pipeline/release"
	"github.com/goreleaser/releaser/pipeline/repos"
	"github.com/goreleaser/releaser/pipeline/valid"
	"github.com/urfave/cli"
)

var version = "master"

var pipes = []pipeline.Pipe{
	// load data, set defaults, etc...
	defaults.Pipe{},
	env.Pipe{},
	git.Pipe{},
	repos.Pipe{},

	// validate
	valid.Pipe{},

	// real work
	build.Pipe{},
	compress.Pipe{},
	release.Pipe{},
	brew.Pipe{},
	cleanup.Pipe{},
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
		config, err := config.Load(file)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		context := context.New(config)
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
