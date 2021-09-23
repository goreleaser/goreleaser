package cmd

import (
	"runtime"
	"time"

	"github.com/apex/log"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipe/git"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/spf13/cobra"
)

type releaseCmd struct {
	cmd  *cobra.Command
	opts releaseOpts
}

type releaseOpts struct {
	config            string
	releaseNotesFile  string
	releaseNotesTmpl  string
	releaseHeaderFile string
	releaseHeaderTmpl string
	releaseFooterFile string
	releaseFooterTmpl string
	autoSnapshot      bool
	snapshot          bool
	skipPublish       bool
	skipSign          bool
	skipValidate      bool
	skipAnnounce      bool
	rmDist            bool
	deprecated        bool
	parallelism       int
	timeout           time.Duration
}

func newReleaseCmd() *releaseCmd {
	root := &releaseCmd{}
	// nolint: dupl
	cmd := &cobra.Command{
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

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	cmd.Flags().StringVar(&root.opts.releaseNotesFile, "release-notes", "", "Load custom release notes from a markdown file")
	cmd.Flags().StringVar(&root.opts.releaseHeaderFile, "release-header", "", "Load custom release notes header from a markdown file")
	cmd.Flags().StringVar(&root.opts.releaseFooterFile, "release-footer", "", "Load custom release notes footer from a markdown file")
	cmd.Flags().StringVar(&root.opts.releaseNotesTmpl, "release-notes-tmpl", "", "Load custom release notes from a templated markdown file (overrides --release-notes)")
	cmd.Flags().StringVar(&root.opts.releaseHeaderTmpl, "release-header-tmpl", "", "Load custom release notes header from a templated markdown file (overrides --release-header)")
	cmd.Flags().StringVar(&root.opts.releaseFooterTmpl, "release-footer-tmpl", "", "Load custom release notes footer from a templated markdown file (overrides --release-footer)")
	cmd.Flags().BoolVar(&root.opts.autoSnapshot, "auto-snapshot", false, "Automatically sets --snapshot if the repo is dirty")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts (implies --skip-publish, --skip-announce and --skip-validate)")
	cmd.Flags().BoolVar(&root.opts.skipPublish, "skip-publish", false, "Skips publishing artifacts")
	cmd.Flags().BoolVar(&root.opts.skipAnnounce, "skip-announce", false, "Skips announcing releases (implies --skip-validate)")
	cmd.Flags().BoolVar(&root.opts.skipSign, "skip-sign", false, "Skips signing artifacts")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips git checks")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Removes the dist folder")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire release process")
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")

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

func setupReleaseContext(ctx *context.Context, options releaseOpts) *context.Context {
	ctx.Parallelism = runtime.NumCPU()
	if options.parallelism > 0 {
		ctx.Parallelism = options.parallelism
	}
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.ReleaseNotesFile = options.releaseNotesFile
	ctx.ReleaseNotesTmpl = options.releaseNotesTmpl
	ctx.ReleaseHeaderFile = options.releaseHeaderFile
	ctx.ReleaseHeaderTmpl = options.releaseHeaderTmpl
	ctx.ReleaseFooterFile = options.releaseFooterFile
	ctx.ReleaseFooterTmpl = options.releaseFooterTmpl
	ctx.Snapshot = options.snapshot
	if options.autoSnapshot && git.CheckDirty() != nil {
		log.Info("git repo is dirty and --auto-snapshot is set, implying --snapshot")
		ctx.Snapshot = true
	}
	ctx.SkipPublish = ctx.Snapshot || options.skipPublish
	ctx.SkipAnnounce = ctx.Snapshot || options.skipPublish || options.skipAnnounce
	ctx.SkipValidate = ctx.Snapshot || options.skipValidate
	ctx.SkipSign = options.skipSign
	ctx.RmDist = options.rmDist

	// test only
	ctx.Deprecated = options.deprecated
	return ctx
}
