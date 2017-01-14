package main

import (
	"log"
	"os"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/initialize"
	"github.com/goreleaser/releaser/pipeline"
	"github.com/goreleaser/releaser/pipeline/brew"
	"github.com/goreleaser/releaser/pipeline/build"
	"github.com/goreleaser/releaser/pipeline/compress"
	"github.com/goreleaser/releaser/pipeline/release"
	"github.com/urfave/cli"
)

var version = "master"

var pipes = []pipeline.Pipe{
	build.Pipe{},
	compress.Pipe{},
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
	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "Generate a skeleton releaser.yml file based on the origin remote of the repo",
			Action: func(c *cli.Context) error {
				if err := initialize.Init(); err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				log.Println("gorelaser.yml created!")
				return nil
			},
		},
	}

	app.Action = func(c *cli.Context) (err error) {
		var file = c.String("config")
		config, err := config.Load(file)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		log.Println("Releasing", config.Git.CurrentTag, "...")
		for _, pipe := range pipes {
			if err := pipe.Run(config); err != nil {
				return cli.NewExitError(pipe.Name()+" failed: "+err.Error(), 1)
			}
		}
		log.Println("Done!")
		return
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
