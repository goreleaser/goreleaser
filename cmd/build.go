package cmd

import (
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

type sharedBuildOpts struct {
	config       string
	snapshot     bool
	skipValidate bool
	rmDist       bool
	deprecated   bool
	parallelism  int
	timeout      time.Duration
	buildGoos    []string
	buildGoarch  []string
}

type buildOpts struct {
	sharedBuildOpts
	skipPostHooks bool
}

func addSharedBuildFlags(cmd *cobra.Command, opts *sharedBuildOpts) {
	cmd.Flags().StringVarP(&opts.config, "config", "f", "", "Load configuration from file")
	cmd.Flags().BoolVar(&opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations and without publishing any artifacts")
	cmd.Flags().BoolVar(&opts.skipValidate, "skip-validate", false, "Skips several sanity checks")
	cmd.Flags().BoolVar(&opts.rmDist, "rm-dist", false, "Remove the dist folder before building")
	cmd.Flags().IntVarP(&opts.parallelism, "parallelism", "p", 4, "Amount tasks to run concurrently")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire build process")
	cmd.Flags().BoolVar(&opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")

	// In pipe/build, we special case --goos=^A --goarch=^B. The likely
	// intent of such a flag combination is to exclude exactly A_B, but
	// (not A) && (not B) == not (A || B) != not (A && B) by De Morgan's laws.
	// So if we detect exactly one param in both --goos and --goarch, we
	// interpret it as the latter [not (A && B)].
	//
	// If you truly did intend not (A || B), a simple workaround is to repeat
	// one of the flags:  --goos=^A,^A --goarch=^B.
	cmd.Flags().StringSliceVar(
		&opts.buildGoos,
		"goos",
		nil,
		"Build targets that match any of the specified OSes. Supports negation with ^. "+
			"If --goarch is passed, then build targets that match both lists.",
	)
	cmd.Flags().StringSliceVar(
		&opts.buildGoarch,
		"goarch",
		nil,
		"Build targets that match any of the specified architectures. Supports negation with ^. "+
			"If --goos is passed, then build targets that match both lists.",
	)
	_ = cmd.Flags().MarkHidden("deprecated")
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

	cmd.Flags().BoolVar(&root.opts.skipPostHooks, "skip-post-hooks", false, "Skips all post-build hooks")

	addSharedBuildFlags(cmd, &root.opts.sharedBuildOpts)

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

func setupSharedBuildContext(ctx *context.Context, options *sharedBuildOpts) {
	ctx.Parallelism = options.parallelism
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.Snapshot = options.snapshot
	ctx.SkipValidate = ctx.Snapshot || options.skipValidate
	ctx.RmDist = options.rmDist
	ctx.BuildGoos = options.buildGoos
	ctx.BuildGoarch = options.buildGoarch

	// test only
	ctx.Deprecated = options.deprecated
}

func setupBuildContext(ctx *context.Context, options buildOpts) *context.Context {
	setupSharedBuildContext(ctx, &options.sharedBuildOpts)

	ctx.SkipPostBuildHooks = options.skipPostHooks
	ctx.SkipTokenCheck = true

	return ctx
}
