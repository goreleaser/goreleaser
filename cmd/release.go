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

type releaseCmd struct {
	cmd  *cobra.Command
	opts releaseOpts
}

type releaseOpts struct {
	sharedBuildOpts
	releaseNotes  string
	releaseHeader string
	releaseFooter string
	skipPublish   bool
	skipSign      bool
}

func newReleaseCmd() *releaseCmd {
	var root = &releaseCmd{}
	// nolint: dupl
	var cmd = &cobra.Command{
		Use:           "release",
		Aliases:       []string{"r"},
		Short:         "Releases the current project",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			log.Infof(color.New(color.Bold).Sprint("releasing..."))

			ctx, err := releaseProject(root.opts)
			if err != nil {
				return wrapError(err, color.New(color.Bold).Sprintf("release failed after %0.2fs", time.Since(start).Seconds()))
			}

			if ctx.Deprecated {
				log.Warn(color.New(color.Bold).Sprintf("your config is using deprecated properties, check logs above for details"))
			}

			log.Infof(color.New(color.Bold).Sprintf("release succeeded after %0.2fs", time.Since(start).Seconds()))
			return nil
		},
	}

	cmd.Flags().StringVar(&root.opts.releaseNotes, "release-notes", "", "Load custom release notes from a markdown file")
	cmd.Flags().StringVar(&root.opts.releaseHeader, "release-header", "", "Load custom release notes header from a markdown file")
	cmd.Flags().StringVar(&root.opts.releaseFooter, "release-footer", "", "Load custom release notes footer from a markdown file")
	cmd.Flags().BoolVar(&root.opts.skipPublish, "skip-publish", false, "Skips publishing artifacts")
	cmd.Flags().BoolVar(&root.opts.skipSign, "skip-sign", false, "Skips signing the artifacts")

	addSharedBuildFlags(cmd, &root.opts.sharedBuildOpts)

	root.cmd = cmd
	return root
}

func releaseProject(options releaseOpts) (*context.Context, error) {
	cfg, err := loadConfig(options.config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.NewWithTimeout(cfg, options.timeout)
	defer cancel()
	setupReleaseContext(ctx, options)
	return ctx, ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipeline.Pipeline {
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

func setupReleaseContext(ctx *context.Context, options releaseOpts) *context.Context {
	setupSharedBuildContext(ctx, &options.sharedBuildOpts)

	ctx.ReleaseNotes = options.releaseNotes
	ctx.ReleaseHeader = options.releaseHeader
	ctx.ReleaseFooter = options.releaseFooter
	ctx.SkipPublish = ctx.Snapshot || options.skipPublish
	ctx.SkipSign = options.skipSign

	return ctx
}
