package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	stdctx "context"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipeline"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
	autoSnapshot bool
	clean        bool
	deprecated   bool
	parallelism  int
	timeout      time.Duration
	singleTarget bool
	output       string
	skips        []string
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

When using ` + "`--single-target`" + `, you use the ` + "`TARGET`, or GOOS`, `GOARCH`, `GOARM`, `GOAMD64`, `GOARM64`, `GORISCV64`, `GO386`, `GOPPC64`, and `GOMIPS`" + ` environment variables to determine the target, defaulting to the current machine target if not set.
`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return buildProject(cmd.Context(), root.opts)
		},
	}

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot build, skipping all validations")
	cmd.Flags().BoolVar(&root.opts.autoSnapshot, "auto-snapshot", false, "Automatically sets --snapshot if the repository is dirty")
	cmd.Flags().BoolVar(&root.opts.clean, "clean", false, "Removes the 'dist' directory before building")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Number of tasks to run concurrently (default: number of CPUs)")
	_ = cmd.RegisterFlagCompletionFunc("parallelism", cobra.NoFileCompletions)
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", time.Hour, "Timeout to the entire build process")
	_ = cmd.RegisterFlagCompletionFunc("timeout", cobra.NoFileCompletions)
	cmd.Flags().BoolVar(&root.opts.singleTarget, "single-target", false, "Builds only for current GOOS and GOARCH, regardless of what's set in the configuration file")
	cmd.Flags().StringArrayVar(&root.opts.ids, "id", nil, "Builds only the specified build ids")
	_ = cmd.RegisterFlagCompletionFunc("id", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// TODO: improve this
		cfg, err := loadConfig(!root.opts.snapshot, root.opts.config)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(cfg.Builds))
		for _, build := range cfg.Builds {
			if !strings.HasPrefix(build.ID, toComplete) {
				continue
			}
			ids = append(ids, build.ID)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	cmd.Flags().StringVarP(&root.opts.output, "output", "o", "", "Copy the binary to the path after the build. Only taken into account when using --single-target and a single id (either with --id or if configuration only has one build)")
	// _ = cmd.MarkFlagFilename("output") // no extensions to filter
	_ = cmd.Flags().MarkHidden("deprecated")

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

func buildProject(parent stdctx.Context, options buildOpts) error {
	start := time.Now()
	cfg, err := loadConfig(!options.snapshot, options.config)
	if err != nil {
		return decorateWithCtxErr(parent, err, "build", after(start))
	}

	ctx, cancel := context.WrapWithTimeout(parent, cfg, options.timeout)
	defer cancel()

	if err := setupBuildContext(ctx, options); err != nil {
		return decorateWithCtxErr(ctx, err, "build", after(start))
	}
	for _, pipe := range setupPipeline(ctx, options) {
		if err := skip.Maybe(
			pipe,
			logging.Log(
				pipe.String(),
				errhandler.Handle(pipe.Run),
			),
		)(ctx); err != nil {
			return decorateWithCtxErr(ctx, err, "build", after(start))
		}
	}

	deprecateWarn(ctx)
	log.Infof(boldStyle.Render(fmt.Sprintf("build succeeded after %s", after(start).String())))
	return nil
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

	if options.autoSnapshot && git.CheckDirty(ctx) != nil {
		log.Info("git repository is dirty and --auto-snapshot is set, implying --snapshot")
		ctx.Snapshot = true
	}

	if err := skips.SetBuild(ctx, options.skips...); err != nil {
		return err
	}

	if ctx.Snapshot {
		skips.Set(ctx, skips.Validate)
	}

	ctx.SkipTokenCheck = true
	ctx.Clean = options.clean

	if options.singleTarget {
		ctx.SingleTarget = true
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
		if slices.Contains(ids, build.ID) {
			keep = append(keep, build)
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
		return errors.New("no binary found")
	}
	path := bins[0].Path
	out := w.output
	if out == "." {
		out = filepath.Base(path)
	}
	return gio.Copy(path, out)
}
