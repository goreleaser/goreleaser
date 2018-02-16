package goreleaserlib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	yaml "gopkg.in/yaml.v2"

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
	"github.com/goreleaser/goreleaser/pipeline/fpm"
	"github.com/goreleaser/goreleaser/pipeline/git"
	"github.com/goreleaser/goreleaser/pipeline/nfpm"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/goreleaser/goreleaser/pipeline/scoop"
	"github.com/goreleaser/goreleaser/pipeline/sign"
	"github.com/goreleaser/goreleaser/pipeline/snapcraft"
)

var (
	normalPadding    = cli.Default.Padding
	increasedPadding = normalPadding * 2
)

func init() {
	log.SetHandler(cli.Default)
}

var pipes = []pipeline.Piper{
	defaults.Pipe{},        // load default configs
	dist.Pipe{},            // ensure ./dist is clean
	git.Pipe{},             // get and validate git repo state
	effectiveconfig.Pipe{}, // writes the actual config (with defaults et al set) to dist
	changelog.Pipe{},       // builds the release changelog
	env.Pipe{},             // load and validate environment variables
	build.Pipe{},           // build
	archive.Pipe{},         // archive in tar.gz, zip or binary (which does no archiving at all)
	fpm.Pipe{},             // archive via fpm (deb, rpm) using fpm
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

// Flags interface represents an extractor of cli flags
type Flags interface {
	IsSet(s string) bool
	String(s string) string
	Int(s string) int
	Bool(s string) bool
	Duration(s string) time.Duration
}

// Release runs the release process with the given flags
func Release(flags Flags) error {
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
	ctx.Validate = !flags.Bool("skip-validate")
	ctx.Publish = !flags.Bool("skip-publish")
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
	if ctx.Snapshot {
		log.Info("publishing disabled in snapshot mode")
		ctx.Publish = false
	}
	ctx.RmDist = flags.Bool("rm-dist")
	return doRelease(ctx)
}

func doRelease(ctx *context.Context) error {
	defer restoreOutputPadding()
	return ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipes {
			restoreOutputPadding()
			log.Infof(color.New(color.Bold).Sprint(strings.ToUpper(pipe.String())))
			cli.Default.Padding = increasedPadding
			if err := handle(pipe.Run(ctx)); err != nil {
				return err
			}
		}
		return nil
	})
}

func restoreOutputPadding() {
	cli.Default.Padding = normalPadding
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
func InitProject(filename string) error {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s already exists", filename)
	}

	var ctx = context.New(config.Project{})
	var pipe = defaults.Pipe{}
	defer restoreOutputPadding()
	log.Infof(color.New(color.Bold).Sprint(strings.ToUpper(pipe.String())))
	cli.Default.Padding = increasedPadding
	if err := pipe.Run(ctx); err != nil {
		return err
	}
	out, err := yaml.Marshal(ctx.Config)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, out, 0644)
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
