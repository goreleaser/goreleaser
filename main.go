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

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type releaseOptions struct {
	Config       string
	ReleaseNotes string
	Snapshot     bool
	SkipPublish  bool
	SkipSign     bool
	SkipValidate bool
	RmDist       bool
	Debug        bool
	Parallelism  int
	Timeout      time.Duration
}

func init() {
	log.SetHandler(cli.Default)
}

func main() {
	fmt.Println()
	defer fmt.Println()

	var app = kingpin.New("goreleaser", "Deliver Go binaries as fast and easily as possible")
	var initCmd = app.Command("init", "Generates a .goreleaser.yml file").Alias("i")
	var releaseCmd = app.Command("release", "Releases the current project").Alias("r").Default()
	var config = releaseCmd.Flag("config", "Load configuration from file").Short('c').Short('f').PlaceHolder(".goreleaser.yml").String()
	var releaseNotes = releaseCmd.Flag("release-notes", "Load custom release notes from a markdown file").PlaceHolder("notes.md").String()
	var snapshot = releaseCmd.Flag("snapshot", "Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts").Bool()
	var skipPublish = releaseCmd.Flag("skip-publish", "Generates all artifacts but does not publish them anywhere").Bool()
	var skipSign = releaseCmd.Flag("skip-sign", "Skips signing the artifacts").Bool()
	var skipValidate = releaseCmd.Flag("skip-validate", "Skips all git sanity checks").Bool()
	var rmDist = releaseCmd.Flag("rm-dist", "Remove the dist folder before building").Bool()
	var parallelism = releaseCmd.Flag("parallelism", "Amount of slow tasks to do in concurrently").Short('p').Default("4").Int() // TODO: use runtime.NumCPU here?
	var debug = releaseCmd.Flag("debug", "Enable debug mode").Bool()
	var timeout = releaseCmd.Flag("timeout", "Timeout to the entire release process").Default("30m").Duration()

	app.Version(fmt.Sprintf("%v, commit %v, built at %v", version, commit, date))
	app.VersionFlag.Short('v')
	app.HelpFlag.Short('h')

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case initCmd.FullCommand():
		var filename = ".goreleaser.yml"
		if err := initProject(filename); err != nil {
			log.WithError(err).Error("failed to init project")
			terminate(1)
			return
		}
		log.WithField("file", filename).Info("config created; please edit accordingly to your needs")
	case releaseCmd.FullCommand():
		start := time.Now()
		log.Infof(color.New(color.Bold).Sprintf("releasing using goreleaser %s...", version))
		var options = releaseOptions{
			Config:       *config,
			ReleaseNotes: *releaseNotes,
			Snapshot:     *snapshot,
			SkipPublish:  *skipPublish,
			SkipValidate: *skipValidate,
			SkipSign:     *skipSign,
			RmDist:       *rmDist,
			Parallelism:  *parallelism,
			Debug:        *debug,
			Timeout:      *timeout,
		}
		if err := releaseProject(options); err != nil {
			log.WithError(err).Errorf(color.New(color.Bold).Sprintf("release failed after %0.2fs", time.Since(start).Seconds()))
			terminate(1)
			return
		}
		log.Infof(color.New(color.Bold).Sprintf("release succeeded after %0.2fs", time.Since(start).Seconds()))
	}
}

func terminate(status int) {
	os.Exit(status)
}

func releaseProject(options releaseOptions) error {
	if options.Debug {
		log.SetLevel(log.DebugLevel)
	}
	cfg, err := loadConfig(options.Config)
	if err != nil {
		return err
	}
	ctx, cancel := context.NewWithTimeout(cfg, options.Timeout)
	defer cancel()
	ctx.Parallelism = options.Parallelism
	ctx.Debug = options.Debug
	log.Debugf("parallelism: %v", ctx.Parallelism)
	if options.ReleaseNotes != "" {
		bts, err := ioutil.ReadFile(options.ReleaseNotes)
		if err != nil {
			return err
		}
		log.WithField("file", options.ReleaseNotes).Info("loaded custom release notes")
		log.WithField("file", options.ReleaseNotes).Debugf("custom release notes: \n%s", string(bts))
		ctx.ReleaseNotes = string(bts)
	}
	ctx.Snapshot = options.Snapshot
	ctx.SkipPublish = ctx.Snapshot || options.SkipPublish
	ctx.SkipValidate = ctx.Snapshot || options.SkipValidate
	ctx.SkipSign = options.SkipSign
	ctx.RmDist = options.RmDist
	return doRelease(ctx)
}

func doRelease(ctx *context.Context) error {
	defer func() { cli.Default.Padding = 3 }()
	var release = func() error {
		for _, pipe := range pipeline.Pipeline {
			cli.Default.Padding = 3
			log.Infof(color.New(color.Bold).Sprint(strings.ToUpper(pipe.String())))
			cli.Default.Padding = 6
			if err := handle(pipe.Run(ctx)); err != nil {
				return err
			}
		}
		return nil
	}
	return ctrlc.Default.Run(ctx, release)
}

func handle(err error) error {
	if err == nil {
		return nil
	}
	if pipe.IsSkip(err) {
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
	log.Infof(color.New(color.Bold).Sprintf("Generating %s file", filename))
	return ioutil.WriteFile(filename, []byte(exampleConfig), 0644)
}

func loadConfig(path string) (config.Project, error) {
	if path != "" {
		return config.Load(path)
	}
	for _, f := range [4]string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		proj, err := config.Load(f)
		if err != nil && os.IsNotExist(err) {
			continue
		}
		return proj, err
	}
	// the user didn't specified a config file and the known files
	// doest not exist, so, return an empty config and a nil err.
	log.Warn("could not load config, using defaults")
	return config.Project{}, nil
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
