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
	"github.com/goreleaser/goreleaser/pipeline/source"
	"github.com/urfave/cli"
)

var version = "master"

var pipes = []pipeline.Pipe{
	// load data, set defaults, etc...
	defaults.Pipe{},
	env.Pipe{},
	git.Pipe{},
	repos.Pipe{},

	&source.Pipe{},

	// real work
	build.Pipe{},
	archive.Pipe{},
	release.Pipe{},
	brew.Pipe{},
}

func main() {
	var app = cli.NewApp()
	app.Name = "goreleaser"
	app.Version = version
	app.Usage = "Deliver Go binaries as fast and easily as possible"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load configuration from `FILE`",
			Value: "goreleaser.yml",
		},
	}
	app.Action = func(c *cli.Context) error {
		var file = c.String("config")
		cfg, err := config.Load(file)
		// Allow failing to load the config file if file is not explicitly specified
		if err != nil && c.IsSet("config") {
			return cli.NewExitError(err.Error(), 1)
		}
		ctx := context.New(cfg)
		log.SetFlags(0)
		if err := runPipeline(ctx); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		log.Println("Done!")
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func runPipeline(ctx *context.Context) error {
	for _, pipe := range pipes {
		log.Println(pipe.Description())
		log.SetPrefix(" -> ")
		err := pipe.Run(ctx)
		log.SetPrefix("")
		cleaner, ok := pipe.(pipeline.Cleaner)
		if ok {
			defer cleaner.Clean(ctx)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
