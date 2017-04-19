package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/goreleaser/goreleaser/pipeline/archive"
	"github.com/goreleaser/goreleaser/pipeline/brew"
	"github.com/goreleaser/goreleaser/pipeline/build"
	"github.com/goreleaser/goreleaser/pipeline/checksums"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/goreleaser/goreleaser/pipeline/env"
	"github.com/goreleaser/goreleaser/pipeline/fpm"
	"github.com/goreleaser/goreleaser/pipeline/git"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/urfave/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var pipes = []pipeline.Pipe{
	defaults.Pipe{},  // load default configs
	git.Pipe{},       // get and validate git repo state
	env.Pipe{},       // load and validate environment variables
	build.Pipe{},     // build
	archive.Pipe{},   // archive (tar.gz, zip, etc)
	fpm.Pipe{},       // archive via fpm (deb, rpm, etc)
	checksums.Pipe{}, // checksums of the files
	release.Pipe{},   // release to github
	brew.Pipe{},      // push to brew tap
}

func init() {
	log.SetFlags(0)
}

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
			Usage: "Path to a markdown file with custom release notes",
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
	app.Action = func(c *cli.Context) (err error) {
		var file = c.String("config")
		var notes = c.String("release-notes")
		cfg, err := config.Load(file)
		if err != nil {
			// Allow file not found errors if config file was not
			// explicitly specified
			_, statErr := os.Stat(file)
			if !os.IsNotExist(statErr) || c.IsSet("config") {
				return cli.NewExitError(err.Error(), 1)
			}
		}
		var ctx = context.New(cfg)
		ctx.Validate = !c.Bool("skip-validate")
		ctx.Publish = !c.Bool("skip-publish")
		if notes != "" {
			bts, err := ioutil.ReadFile(notes)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			ctx.Changelog = string(bts)
		}
		for _, pipe := range pipes {
			log.Println(pipe.Description())
			log.SetPrefix(" -> ")
			if err := pipe.Run(ctx); err != nil {
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
