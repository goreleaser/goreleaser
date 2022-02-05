package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/apex/log"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/muesli/coral"
)

type buildCmd struct {
	cmd  *coral.Command
	opts buildOpts
}

type buildOpts struct {
	config        string
	id            string
	snapshot      bool
	skipValidate  bool
	skipPostHooks bool
	rmDist        bool
	deprecated    bool
	parallelism   int
	timeout       time.Duration
	singleTarget  bool
	output        string
}

func newBuildCmd() *buildCmd {
	root := &buildCmd{}
	// nolint: dupl
	cmd := &coral.Command{
		Use:     "build",
		Aliases: []string{"b"},
		Short:   "Builds the current project",
		Long: `The ` + "`goreleaser build`" + ` command is analogous to the
` + "`go build`" + ` command, in the sense it only builds binaries.

Its itented usage is, for example, within Makefiles to avoid setting up
ldflags and etc in several places. That way, the GoReleaser config becomes the
source of truth for how the binaries should be built.

It also allows you to generate a local build for your current machine only using
the ` + "`--single-target`" + ` option, and specific build IDs using the
` + "`--id`" + ` option in case you have more than one.

When using ` + "`--single-target`" + `, the ` + "`GOOS`" + ` and
` + "`GOARCH`" + ` environment variables are used to determine the target,
defaulting to the current's machine target if not set.
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          coral.NoArgs,
		RunE: func(cmd *coral.Command, args []string) error {
			start := time.Now()

			log.Infof(color.New(color.Bold).Sprint("building..."))

			ctx, err := buildProject(root.opts)
			if err != nil {
				return wrapError(err, color.New(color.Bold).Sprintf("build failed after %0.2fs", time.Since(start).Seconds()))
			}

			if ctx.Deprecated {
				log.Warn(color.New(color.Bold).Sprintf("your config is using deprecated properties, check logs above for details"))
			}

			log.Infof(color.New(color.Bold).Sprintf("build succeeded after %0.2fs", time.Since(start).Seconds()))
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips several sanity checks")
	cmd.Flags().BoolVar(&root.opts.skipPostHooks, "skip-post-hooks", false, "Skips all post-build hooks")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Remove the dist folder before building")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire build process")
	cmd.Flags().BoolVar(&root.opts.singleTarget, "single-target", false, "Builds only for current GOOS and GOARCH")
	cmd.Flags().StringVar(&root.opts.id, "id", "", "Builds only the specified build id")
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	cmd.Flags().StringVarP(&root.opts.output, "output", "o", "", "Path to the binary, defaults to the distribution folder according to configs. Only taked into account when using --single-target and a single id (either with --id or if config only has one build)")
	_ = cmd.Flags().MarkHidden("deprecated")

	root.cmd = cmd
	return root
}

func buildProject(options buildOpts) (*context.Context, error) {
	cfg, err := loadConfig(options.config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.NewWithTimeout(cfg, options.timeout)
	defer cancel()
	if err := setupBuildContext(ctx, options); err != nil {
		return nil, err
	}
	return ctx, ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range setupPipeline(ctx, options) {
			if err := skip.Maybe(
				pipe,
				logging.Log(
					pipe.String(),
					errhandler.Handle(pipe.Run),
					logging.DefaultInitialPadding,
				),
			)(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func setupPipeline(ctx *context.Context, options buildOpts) []pipeline.Piper {
	if options.singleTarget && (options.id != "" || len(ctx.Config.Builds) == 1) {
		return append(pipeline.BuildCmdPipeline, withOutputPipe{options.output})
	}
	return pipeline.BuildCmdPipeline
}

func setupBuildContext(ctx *context.Context, options buildOpts) error {
	ctx.Parallelism = runtime.NumCPU()
	if options.parallelism > 0 {
		ctx.Parallelism = options.parallelism
	}
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.Snapshot = options.snapshot
	ctx.SkipValidate = ctx.Snapshot || options.skipValidate
	ctx.SkipPostBuildHooks = options.skipPostHooks
	ctx.RmDist = options.rmDist
	ctx.SkipTokenCheck = true

	if options.singleTarget {
		setupBuildSingleTarget(ctx)
	}

	if options.id != "" {
		if err := setupBuildID(ctx, options.id); err != nil {
			return err
		}
	}

	// test only
	ctx.Deprecated = options.deprecated
	return nil
}

func setupBuildSingleTarget(ctx *context.Context) {
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	log.Infof("building only for %s/%s", goos, goarch)
	if len(ctx.Config.Builds) == 0 {
		ctx.Config.Builds = append(ctx.Config.Builds, config.Build{})
	}
	for i := range ctx.Config.Builds {
		build := &ctx.Config.Builds[i]
		build.Goos = []string{goos}
		build.Goarch = []string{goarch}
	}
}

func setupBuildID(ctx *context.Context, id string) error {
	if len(ctx.Config.Builds) < 2 {
		log.Warn("single build in config, '--id' ignored")
		return nil
	}

	var keep []config.Build
	for _, build := range ctx.Config.Builds {
		if build.ID == id {
			keep = append(keep, build)
			break
		}
	}

	if len(keep) == 0 {
		return fmt.Errorf("no builds with id '%s'", id)
	}

	ctx.Config.Builds = keep
	return nil
}

// withOutputPipe copies the binary from dist to the specified output path.
type withOutputPipe struct {
	output string
}

func (w withOutputPipe) String() string {
	return fmt.Sprintf("copying binary to %q", w.output)
}

func (w withOutputPipe) Run(ctx *context.Context) error {
	path := ctx.Artifacts.Filter(artifact.ByType(artifact.Binary)).List()[0].Path
	out := w.output
	if out == "" {
		out = filepath.Base(path)
	}
	return gio.Copy(path, out)
}
