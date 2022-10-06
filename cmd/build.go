package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/caarlos0/ctrlc"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/spf13/cobra"
)

type buildCmd struct {
	cmd  *cobra.Command
	opts buildOpts
}

type buildOpts struct {
	config        string
	ids           []string
	snapshot      bool
	skipValidate  bool
	skipBefore    bool
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
	cmd := &cobra.Command{
		Use:     "build",
		Aliases: []string{"b"},
		Short:   "Builds the current project",
		Long: `The ` + "`goreleaser build`" + ` command is analogous to the ` + "`go build`" + ` command, in the sense it only builds binaries.

Its intended usage is, for example, within Makefiles to avoid setting up ldflags and etc in several places. That way, the GoReleaser config becomes the source of truth for how the binaries should be built.

It also allows you to generate a local build for your current machine only using the ` + "`--single-target`" + ` option, and specific build IDs using the ` + "`--id`" + ` option in case you have more than one.

When using ` + "`--single-target`" + `, the ` + "`GOOS`" + ` and ` + "`GOARCH`" + ` environment variables are used to determine the target, defaulting to the current machine target if not set.
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: timedRunE("build", func(cmd *cobra.Command, args []string) error {
			ctx, err := buildProject(root.opts)
			if err != nil {
				return err
			}
			deprecateWarn(ctx)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips several sanity checks")
	cmd.Flags().BoolVar(&root.opts.skipBefore, "skip-before", false, "Skips global before hooks")
	cmd.Flags().BoolVar(&root.opts.skipPostHooks, "skip-post-hooks", false, "Skips all post-build hooks")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Remove the dist folder before building")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire build process")
	cmd.Flags().BoolVar(&root.opts.singleTarget, "single-target", false, "Builds only for current GOOS and GOARCH, regardless of what's set in the configuration file")
	cmd.Flags().StringArrayVar(&root.opts.ids, "id", nil, "Builds only the specified build ids")
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	cmd.Flags().StringVarP(&root.opts.output, "output", "o", "", "Copy the binary to the path after the build. Only taken into account when using --single-target and a single id (either with --id or if configuration only has one build)")
	_ = cmd.Flags().SetAnnotation("output", cobra.BashCompFilenameExt, []string{""})
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
				),
			)(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func setupPipeline(ctx *context.Context, options buildOpts) []pipeline.Piper {
	if options.output != "" && options.singleTarget && (len(options.ids) > 0 || len(ctx.Config.Builds) == 1) {
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
	ctx.SkipBefore = options.skipBefore
	ctx.SkipPostBuildHooks = options.skipPostHooks
	ctx.RmDist = options.rmDist
	ctx.SkipTokenCheck = true

	if options.singleTarget {
		setupBuildSingleTarget(ctx)
	}

	if len(options.ids) > 0 {
		if err := setupBuildID(ctx, options.ids); err != nil {
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
	log.WithField("reason", "single target is enabled").Warnf("building only for %s/%s", goos, goarch)
	if len(ctx.Config.Builds) == 0 {
		ctx.Config.Builds = append(ctx.Config.Builds, config.Build{})
	}
	for i := range ctx.Config.Builds {
		build := &ctx.Config.Builds[i]
		build.Goos = []string{goos}
		build.Goarch = []string{goarch}
		build.Goarm = nil
		build.Gomips = nil
		build.Goamd64 = nil
		build.Targets = nil
	}
}

func setupBuildID(ctx *context.Context, ids []string) error {
	if len(ctx.Config.Builds) < 2 {
		log.Warn("single build in config, '--id' ignored")
		return nil
	}

	var keep []config.Build
	for _, build := range ctx.Config.Builds {
		for _, id := range ids {
			if build.ID == id {
				keep = append(keep, build)
				break
			}
		}
	}

	if len(keep) == 0 {
		return fmt.Errorf("no builds with ids %s", strings.Join(ids, ", "))
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
	if out == "." {
		out = filepath.Base(path)
	}
	return gio.Copy(path, out)
}
