package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"

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

	pipes = []Piper{
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

// Piper defines a pipe, which can be part of a pipeline (a serie of pipes).
type Piper interface {
	fmt.Stringer

	// Run the pipe
	Run(ctx *context.Context) error
}

func init() {
	log.SetHandler(cli.Default)
}

func main() {
	fmt.Println()
	defer fmt.Println()
	var app = kingpin.New("goreleaser", "Deliver Go binaries as fast and easily as possible")
	app.Version(fmt.Sprintf("%v, commit %v, built at %v", version, commit, date))
	app.VersionFlag.Short('v')
	app.HelpFlag.Short('h')
	var initCmd = app.Command("init", "Generates a .goreleaser.yml file")
	var releaseCmd = app.Command("release", "Release the current project").Default()
	var config = releaseCmd.Flag("config", "Load configuration from `FILE`").
		Short('c').
		Short('f').
		Default(".goreleaser.yml").
		String()
	var releaseNotes = releaseCmd.Flag("release-notes", "Load custom release notes from a markdown `FILE`").
		String()
	var snapshot = releaseCmd.Flag("snapshot", "Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts").
		Bool()
	var skipPublish = releaseCmd.Flag("skip-publish", "Generates all artifacts but does not publish them anywhere").
		Bool()
	var skipValidate = releaseCmd.Flag("skip-validate", "Skips all git state checks").
		Bool()
	var rmDist = releaseCmd.Flag("rm-dist", "Remove the dist folder before building").
		Bool()
	var parallelism = releaseCmd.Flag("parallelism", "Amount of slow task to do in concurrently").
		Short('p').
		Default("4"). // TODO: use runtime.NumCPU here?
		Int()
	var debug = releaseCmd.Flag("debug", "Enable debug mode").
		Bool()
	var timeout = releaseCmd.Flag("timeout", "Timeout to the entire process").
		Default("30m").
		Duration()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case initCmd.FullCommand():
		var filename = ".goreleaser.yml"
		if err := initProject(filename); err != nil {
			// TODO: check how this works out
			log.WithError(err).Error("failed to init project")
			kingpin.Fatalf(err.Error())
		}
		log.WithField("file", filename).Info("config created; please edit accordingly to your needs")
	case releaseCmd.FullCommand():
		start := time.Now()
		log.Infof(bold.Sprint("releasing..."))
		if err := releaseProject(c); err != nil {
			log.WithError(err).Errorf(bold.Sprintf("release failed after %0.2fs", time.Since(start).Seconds()))
			// TODO: check how this works out
			kingpin.Fatalf(err.Error())
		}
		log.Infof(bold.Sprintf("release succeeded after %0.2fs", time.Since(start).Seconds()))
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
