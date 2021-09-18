package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/apex/log"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
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
	id            string
	snapshot      bool
	skipValidate  bool
	skipPostHooks bool
	rmDist        bool
	deprecated    bool
	parallelism   int
	timeout       time.Duration
	singleTarget  bool
}

func newBuildCmd() *buildCmd {
	root := &buildCmd{}
	// nolint: dupl
	cmd := &cobra.Command{
		Use:     "build",
		Aliases: []string{"b"},
		Short:   "Builds the current project",
		Long: `The build command allows you to execute only a subset of the pipeline, i.e. only the build step with its dependencies.

It allows you to quickly check if your GoReleaser build configurations are doing what you expect.

Finally, it allows you to generate a local build for your current machine only using the ` + "`--single-target`" + ` option, and specific build IDs using the ` + "`--id`" + ` option.
`,
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
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips several sanity checks")
	cmd.Flags().BoolVar(&root.opts.skipPostHooks, "skip-post-hooks", false, "Skips all post-build hooks")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Remove the dist folder before building")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire build process")
	cmd.Flags().BoolVar(&root.opts.singleTarget, "single-target", false, "Builds only for current GOOS and GOARCH")
	cmd.Flags().StringVar(&root.opts.id, "id", "", "Builds only the specified build id")
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
	if err := setupBuildContext(ctx, options); err != nil {
		return nil, err
	}
	return ctx, ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipeline.BuildPipeline {
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
