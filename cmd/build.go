package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/caarlos0/ctrlc"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/spf13/cobra"
)

type buildCmd struct {
	cmd  *cobra.Command
	opts buildOpts
}

type buildOpts struct {
	config       string
	ids          []string
	snapshot     bool
	clean        bool
	deprecated   bool
	parallelism  int
	timeout      time.Duration
	singleTarget bool
	output       string
	skips        []string

	// Deprecated: use clean instead.
	rmDist bool
	// Deprecated: use skip instead.
	skipValidate bool
	// Deprecated: use skip instead.
	skipBefore bool
	// Deprecated: use skip instead.
	skipPostHooks bool
}

func newBuildCmd() *buildCmd {
	root := &buildCmd{}
	//nolint:dupl
	cmd := &cobra.Command{
		Use:     "build",
		Aliases: []string{"b"},
		Short:   "Builds the current project",
		Long: `The ` + "`goreleaser build`" + ` command is analogous to the ` + "`go build`" + ` command, in the sense it only builds binaries.

Its intended usage is, for example, within Makefiles to avoid setting up ldflags and etc in several places. That way, the GoReleaser config becomes the source of truth for how the binaries should be built.

It also allows you to generate a local build for your current machine only using the ` + "`--single-target`" + ` option, and specific build IDs using the ` + "`--id`" + ` option in case you have more than one.

When using ` + "`--single-target`" + `, the ` + "`GOOS`" + ` and ` + "`GOARCH`" + ` environment variables are used to determine the target, defaulting to the current machine target if not set.
`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: timedRunE("build", func(_ *cobra.Command, _ []string) error {
			ctx, err := buildProject(root.opts)
			if err != nil {
				return err
			}
			deprecateWarn(ctx)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips several sanity checks")
	cmd.Flags().BoolVar(&root.opts.skipBefore, "skip-before", false, "Skips global before hooks")
	cmd.Flags().BoolVar(&root.opts.skipPostHooks, "skip-post-hooks", false, "Skips all post-build hooks")
	cmd.Flags().BoolVar(&root.opts.clean, "clean", false, "Removes the 'dist' directory before building")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Removes the 'dist' directory before building")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Number of tasks to run concurrently (default: number of CPUs)")
	_ = cmd.RegisterFlagCompletionFunc("parallelism", cobra.NoFileCompletions)
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire build process")
	_ = cmd.RegisterFlagCompletionFunc("timeout", cobra.NoFileCompletions)
	cmd.Flags().BoolVar(&root.opts.singleTarget, "single-target", false, "Builds only for current GOOS and GOARCH, regardless of what's set in the configuration file")
	cmd.Flags().StringArrayVar(&root.opts.ids, "id", nil, "Builds only the specified build ids")
	_ = cmd.RegisterFlagCompletionFunc("id", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		// TODO: improve this
		cfg, err := loadConfig(root.opts.config)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(cfg.Builds))
		for _, build := range cfg.Builds {
			ids = append(ids, build.ID)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	cmd.Flags().StringVarP(&root.opts.output, "output", "o", "", "Copy the binary to the path after the build. Only taken into account when using --single-target and a single id (either with --id or if configuration only has one build)")
	_ = cmd.MarkFlagFilename("output", "")
	_ = cmd.Flags().MarkHidden("rm-dist")
	_ = cmd.Flags().MarkHidden("deprecated")

	for _, f := range []string{
		"post-hooks",
		"before",
		"validate",
	} {
		_ = cmd.Flags().MarkHidden("skip-" + f)
		_ = cmd.Flags().MarkDeprecated("skip-"+f, fmt.Sprintf("please use --skip=%s instead", f))
	}
	cmd.Flags().StringSliceVar(
		&root.opts.skips,
		"skip",
		nil,
		fmt.Sprintf("Skip the given options (valid options are: %s)", skips.Build.String()),
	)
	_ = cmd.RegisterFlagCompletionFunc("skip", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return skips.Build.Complete(toComplete), cobra.ShellCompDirectiveDefault
	})

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
	ctx.Action = context.ActionBuild
	ctx.Deprecated = options.deprecated // test only
	ctx.Parallelism = runtime.GOMAXPROCS(0)
	if options.parallelism > 0 {
		ctx.Parallelism = options.parallelism
	}
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.Snapshot = options.snapshot

	if err := skips.SetBuild(ctx, options.skips...); err != nil {
		return err
	}

	if options.skipValidate {
		skips.Set(ctx, skips.Validate)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-validate was deprecated in favor of --skip=validate, check {{ .URL }} for more details")
	}
	if options.skipBefore {
		skips.Set(ctx, skips.Before)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-before was deprecated in favor of --skip=before, check {{ .URL }} for more details")
	}
	if options.skipPostHooks {
		skips.Set(ctx, skips.PostBuildHooks)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-post-hooks was deprecated in favor of --skip=post-hooks, check {{ .URL }} for more details")
	}

	if options.rmDist {
		deprecate.NoticeCustom(ctx, "-rm-dist", "--rm-dist was deprecated in favor of --clean, check {{ .URL }} for more details")
	}

	if ctx.Snapshot {
		skips.Set(ctx, skips.Validate)
	}

	ctx.SkipTokenCheck = true
	ctx.Clean = options.clean || options.rmDist

	if options.singleTarget {
		ctx.Partial = true
	}

	if len(options.ids) > 0 {
		if err := setupBuildID(ctx, options.ids); err != nil {
			return err
		}
	}

	if skips.Any(ctx, skips.Build...) {
		log.Warnf(
			logext.Warning("skipping %s..."),
			skips.String(ctx),
		)
	}

	return nil
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
	bins := ctx.Artifacts.Filter(artifact.ByType(artifact.Binary)).List()
	if len(bins) == 0 {
		return fmt.Errorf("no binary found")
	}
	path := bins[0].Path
	out := w.output
	if out == "." {
		out = filepath.Base(path)
	}
	return gio.Copy(path, out)
}
