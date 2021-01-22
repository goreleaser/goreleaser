package cmd

import (
	"runtime"
	"time"

	"github.com/apex/log"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/spf13/cobra"
)

type buildCmd struct {
	cmd  *cobra.Command
	opts buildOpts
}

type buildOpts struct {
	config        string
	buildIDs      []string
	snapshot      bool
	skipValidate  bool
	skipPostHooks bool
	rmDist        bool
	deprecated    bool
	parallelism   int
	timeout       time.Duration
}

func newBuildCmd() *buildCmd {
	var root = &buildCmd{}
	// nolint: dupl
	var cmd = &cobra.Command{
		Use:           "build",
		Aliases:       []string{"b"},
		Short:         "Builds the current project",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations and without publishing any artifacts")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips several sanity checks")
	cmd.Flags().BoolVar(&root.opts.skipPostHooks, "skip-post-hooks", false, "Skips all post-build hooks")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Remove the dist folder before building")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", runtime.NumCPU(), "Amount tasks to run concurrently")
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire build process")
	cmd.Flags().StringSliceVar(&root.opts.buildIDs, "build-id", nil, "Build only the passed IDs (default empty). This is specified as a comma-separated list of IDs.")
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
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
	setupBuildContext(ctx, options)
	return ctx, ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipeline.BuildPipeline {
			if err := middleware.Logging(
				pipe.String(),
				middleware.ErrHandler(pipe.Run),
				middleware.DefaultInitialPadding,
			)(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func setupBuildContext(ctx *context.Context, options buildOpts) *context.Context {
	ctx.Parallelism = options.parallelism
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.Snapshot = options.snapshot
	ctx.SkipValidate = ctx.Snapshot || options.skipValidate
	ctx.SkipPostBuildHooks = options.skipPostHooks
	ctx.RmDist = options.rmDist
	ctx.SkipTokenCheck = true
	ctx.BuildIDs = options.buildIDs

	// test only
	ctx.Deprecated = options.deprecated
	return ctx
}
