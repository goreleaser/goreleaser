package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	lcli "github.com/apex/log/handlers/cli"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	"github.com/urfave/cli"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/goreleaser/goreleaser/pipeline/archive"
	"github.com/goreleaser/goreleaser/pipeline/artifactory"
	"github.com/goreleaser/goreleaser/pipeline/brew"
	"github.com/goreleaser/goreleaser/pipeline/build"
	"github.com/goreleaser/goreleaser/pipeline/changelog"
	"github.com/goreleaser/goreleaser/pipeline/checksums"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/goreleaser/goreleaser/pipeline/dist"
	"github.com/goreleaser/goreleaser/pipeline/docker"
	"github.com/goreleaser/goreleaser/pipeline/effectiveconfig"
	"github.com/goreleaser/goreleaser/pipeline/env"
	"github.com/goreleaser/goreleaser/pipeline/git"
	"github.com/goreleaser/goreleaser/pipeline/nfpm"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/goreleaser/goreleaser/pipeline/scoop"
	"github.com/goreleaser/goreleaser/pipeline/sign"
	"github.com/goreleaser/goreleaser/pipeline/snapcraft"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	bold             = color.New(color.Bold)
	normalPadding    = lcli.Default.Padding
	increasedPadding = normalPadding * 2

	pipes = []pipeline.Piper{
		defaults.Pipe{},        // load default configs
		dist.Pipe{},            // ensure ./dist is clean
		git.Pipe{},             // get and validate git repo state
		effectiveconfig.Pipe{}, // writes the actual config (with defaults et al set) to dist
		changelog.Pipe{},       // builds the release changelog
		env.Pipe{},             // load and validate environment variables
		build.Pipe{},           // build
		archive.Pipe{},         // archive in tar.gz, zip or binary (which does no archiving at all)
		nfpm.Pipe{},            // archive via fpm (deb, rpm) using "native" go impl
		snapcraft.Pipe{},       // archive via snapcraft (snap)
		checksums.Pipe{},       // checksums of the files
		sign.Pipe{},            // sign artifacts
		docker.Pipe{},          // create and push docker images
		artifactory.Pipe{},     // push to artifactory
		release.Pipe{},         // release to github
		brew.Pipe{},            // push to brew tap
		scoop.Pipe{},           // push to scoop bucket
	}
)

// Flags interface represents an extractor of cli flags
type Flags interface {
	IsSet(s string) bool
	String(s string) string
	Int(s string) int
	Bool(s string) bool
	Duration(s string) time.Duration
}

func init() {
	log.SetHandler(lcli.Default)
}

func main() {
	fmt.Println()
	defer fmt.Println()
	var app = cli.NewApp()
	app.Name = "goreleaser"
	app.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)
	app.Usage = "Deliver Go binaries as fast and easily as possible"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, file, c, f",
			Usage: "Load configuration from `FILE`",
			Value: ".goreleaser.yml",
		},
		cli.StringFlag{
			Name:  "release-notes",
			Usage: "Load custom release notes from a markdown `FILE`",
		},
		cli.BoolFlag{
			Name:  "snapshot",
			Usage: "Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts",
		},
		cli.BoolFlag{
			Name:  "skip-publish",
			Usage: "Generates all artifacts but does not publish them anywhere",
		},
		cli.BoolFlag{
			Name:  "skip-validate",
			Usage: "Skips all git state checks",
		},
		cli.BoolFlag{
			Name:  "rm-dist",
			Usage: "Remove the dist folder before building",
		},
		cli.IntFlag{
			Name:  "parallelism, p",
			Usage: "Amount of builds launch in parallel",
			Value: 4,
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "How much time the entire release process is allowed to take",
			Value: 30 * time.Minute,
		},
	}
	app.Action = func(c *cli.Context) error {
		start := time.Now()
		log.Infof(bold.Sprint("releasing..."))
		if err := releaseProject(c); err != nil {
			log.WithError(err).Errorf(bold.Sprintf("release failed after %0.2fs", time.Since(start).Seconds()))
			return cli.NewExitError("\n", 1)
		}
		log.Infof(bold.Sprintf("release succeeded after %0.2fs", time.Since(start).Seconds()))
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:    "init",
			Aliases: []string{"i"},
			Usage:   "generate .goreleaser.yml",
			Action: func(c *cli.Context) error {
				var filename = ".goreleaser.yml"
				if err := initProject(filename); err != nil {
					log.WithError(err).Error("failed to init project")
					return cli.NewExitError("\n", 1)
				}

				log.WithField("file", filename).
					Info("config created; please edit accordingly to your needs")
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.WithError(err).Fatal("failed")
	}
}

func releaseProject(flags Flags) error {
	var file = getConfigFile(flags)
	var notes = flags.String("release-notes")
	if flags.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	cfg, err := config.Load(file)
	if err != nil {
		// Allow file not found errors if config file was not
		// explicitly specified
		_, statErr := os.Stat(file)
		if !os.IsNotExist(statErr) || flags.IsSet("config") {
			return err
		}
		log.WithField("file", file).Warn("could not load config, using defaults")
	}
	ctx, cancel := context.NewWithTimeout(cfg, flags.Duration("timeout"))
	defer cancel()
	ctx.Parallelism = flags.Int("parallelism")
	ctx.Debug = flags.Bool("debug")
	log.Debugf("parallelism: %v", ctx.Parallelism)
	if notes != "" {
		bts, err := ioutil.ReadFile(notes)
		if err != nil {
			return err
		}
		log.WithField("file", notes).Info("loaded custom release notes")
		log.WithField("file", notes).Debugf("custom release notes: \n%s", string(bts))
		ctx.ReleaseNotes = string(bts)
	}
	ctx.Snapshot = flags.Bool("snapshot")
	ctx.SkipPublish = ctx.Snapshot || flags.Bool("skip-publish")
	ctx.SkipValidate = ctx.Snapshot || flags.Bool("skip-validate")
	ctx.RmDist = flags.Bool("rm-dist")
	return doRelease(ctx)
}

func doRelease(ctx *context.Context) error {
	defer restoreOutputPadding()
	return ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipes {
			restoreOutputPadding()
			log.Infof(color.New(color.Bold).Sprint(strings.ToUpper(pipe.String())))
			lcli.Default.Padding = increasedPadding
			if err := handle(pipe.Run(ctx)); err != nil {
				return err
			}
		}
		return nil
	})
}

func restoreOutputPadding() {
	lcli.Default.Padding = normalPadding
}

func handle(err error) error {
	if err == nil {
		return nil
	}
	if pipeline.IsSkip(err) {
		log.WithField("reason", err.Error()).Warn("skipped")
		return nil
	}
	return err
}

// InitProject creates an example goreleaser.yml in the current directory
func initProject(filename string) error {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s already exists", filename)
	}
	log.Infof(color.New(color.Bold).Sprint("Generating .goreleaser.yml file"))
	return ioutil.WriteFile(filename, []byte(exampleConfig), 0644)
}

func getConfigFile(flags Flags) string {
	var config = flags.String("config")
	if flags.IsSet("config") {
		return config
	}
	for _, f := range []string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		_, ferr := os.Stat(f)
		if ferr == nil || os.IsExist(ferr) {
			return f
		}
	}
	return config
}

var exampleConfig = `# This is an example goreleaser.yaml file with some sane defaults.
# Make sure to check the documentation at http://goreleaser.com
builds:
- env:
  - CGO_ENABLED=0
archive:
  replacements:
    darwin: Darwin
    linux: Linux
    windows: Windows
    386: i386
    amd64: x86_64
checksum:
  name_template: 'checksums.txt'
snapshot:
  name_template: "{{ .Tag }}-next"
changelog:
  sort: asc
  filters:
    exclude:
    - '^docs:'
    - '^test:'
`
