package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/caarlos0/ctrlc"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipe/git"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/internal/skips"
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
	failFast          bool
	clean             bool
	deprecated        bool
	parallelism       int
	timeout           time.Duration
	skips             []string

	// Deprecated: use clean instead.
	rmDist bool
	// Deprecated: use skips instead.
	skipPublish bool
	// Deprecated: use skips instead.
	skipSign bool
	// Deprecated: use skips instead.
	skipValidate bool
	// Deprecated: use skips instead.
	skipAnnounce bool
	// Deprecated: use skips instead.
	skipSBOMCataloging bool
	// Deprecated: use skips instead.
	skipDocker bool
	// Deprecated: use skips instead.
	skipKo bool
	// Deprecated: use skips instead.
	skipBefore bool
}

func newReleaseCmd() *releaseCmd {
	root := &releaseCmd{}
	// nolint: dupl
	cmd := &cobra.Command{
		Use:               "release",
		Aliases:           []string{"r"},
		Short:             "Releases the current project",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: timedRunE("release", func(_ *cobra.Command, _ []string) error {
			ctx, err := releaseProject(root.opts)
			if err != nil {
				return err
			}
			deprecateWarn(ctx)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&root.opts.config, "config", "f", "", "Load configuration from file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.Flags().StringVar(&root.opts.releaseNotesFile, "release-notes", "", "Load custom release notes from a markdown file (will skip GoReleaser changelog generation)")
	_ = cmd.MarkFlagFilename("release-notes", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseHeaderFile, "release-header", "", "Load custom release notes header from a markdown file")
	_ = cmd.MarkFlagFilename("release-header", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseFooterFile, "release-footer", "", "Load custom release notes footer from a markdown file")
	_ = cmd.MarkFlagFilename("release-footer", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseNotesTmpl, "release-notes-tmpl", "", "Load custom release notes from a templated markdown file (overrides --release-notes)")
	_ = cmd.MarkFlagFilename("release-notes-tmpl", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseHeaderTmpl, "release-header-tmpl", "", "Load custom release notes header from a templated markdown file (overrides --release-header)")
	_ = cmd.MarkFlagFilename("release-header-tmpl", "md", "mkd", "markdown")
	cmd.Flags().StringVar(&root.opts.releaseFooterTmpl, "release-footer-tmpl", "", "Load custom release notes footer from a templated markdown file (overrides --release-footer)")
	_ = cmd.MarkFlagFilename("release-footer-tmpl", "md", "mkd", "markdown")
	cmd.Flags().BoolVar(&root.opts.autoSnapshot, "auto-snapshot", false, "Automatically sets --snapshot if the repository is dirty")
	cmd.Flags().BoolVar(&root.opts.snapshot, "snapshot", false, "Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts (implies --skip=announce,publish,validate)")
	cmd.Flags().BoolVar(&root.opts.failFast, "fail-fast", false, "Whether to abort the release publishing on the first error")
	cmd.Flags().BoolVar(&root.opts.skipPublish, "skip-publish", false, "Skips publishing artifacts (implies --skip=announce)")
	cmd.Flags().BoolVar(&root.opts.skipAnnounce, "skip-announce", false, "Skips announcing releases (implies --skip=validate)")
	cmd.Flags().BoolVar(&root.opts.skipSign, "skip-sign", false, "Skips signing artifacts")
	cmd.Flags().BoolVar(&root.opts.skipSBOMCataloging, "skip-sbom", false, "Skips cataloging artifacts")
	cmd.Flags().BoolVar(&root.opts.skipDocker, "skip-docker", false, "Skips Docker Images/Manifests builds")
	cmd.Flags().BoolVar(&root.opts.skipKo, "skip-ko", false, "Skips Ko builds")
	cmd.Flags().BoolVar(&root.opts.skipBefore, "skip-before", false, "Skips global before hooks")
	cmd.Flags().BoolVar(&root.opts.skipValidate, "skip-validate", false, "Skips git checks")
	cmd.Flags().BoolVar(&root.opts.clean, "clean", false, "Removes the 'dist' directory")
	cmd.Flags().BoolVar(&root.opts.rmDist, "rm-dist", false, "Removes the 'dist' directory")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	_ = cmd.RegisterFlagCompletionFunc("parallelism", cobra.NoFileCompletions)
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", 30*time.Minute, "Timeout to the entire release process")
	_ = cmd.RegisterFlagCompletionFunc("timeout", cobra.NoFileCompletions)
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")
	_ = cmd.Flags().MarkHidden("rm-dist")
	_ = cmd.Flags().MarkDeprecated("rm-dist", "please use --clean instead")
	for _, f := range []string{
		"publish",
		"announce",
		"sign",
		"sbom",
		"docker",
		"ko",
		"before",
		"validate",
	} {
		_ = cmd.Flags().MarkHidden("skip-" + f)
		_ = cmd.Flags().MarkDeprecated("skip"+f, fmt.Sprintf("please use --skip=%s instead", f))
	}
	cmd.Flags().StringSliceVar(
		&root.opts.skips,
		"skip",
		nil,
		fmt.Sprintf("Skip the given options (valid options are %s)", skips.Release.String()),
	)
	_ = cmd.RegisterFlagCompletionFunc("skip", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return skips.Release.Complete(toComplete), cobra.ShellCompDirectiveDefault
	})

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
	if err := setupReleaseContext(ctx, options); err != nil {
		return nil, err
	}
	return ctx, ctrlc.Default.Run(ctx, func() error {
		for _, pipe := range pipeline.Pipeline {
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

func setupReleaseContext(ctx *context.Context, options releaseOpts) error {
	ctx.Action = context.ActionRelease
	ctx.Deprecated = options.deprecated // test only
	ctx.Parallelism = runtime.GOMAXPROCS(0)
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
	ctx.FailFast = options.failFast
	ctx.Clean = options.clean || options.rmDist
	if options.autoSnapshot && git.CheckDirty(ctx) != nil {
		log.Info("git repository is dirty and --auto-snapshot is set, implying --snapshot")
		ctx.Snapshot = true
	}

	if err := skips.SetRelease(ctx, options.skips...); err != nil {
		return err
	}

	// wire deprecated options
	// XXX: remove soon
	if options.skipPublish {
		skips.Set(ctx, skips.Publish)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-publish was deprecated in favor of --skip=publish, check {{ .URL }} for more details")
	}
	if options.skipSign {
		skips.Set(ctx, skips.Sign)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-sign was deprecated in favor of --skip=sign, check {{ .URL }} for more details")
	}
	if options.skipValidate {
		skips.Set(ctx, skips.Validate)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-validate was deprecated in favor of --skip=validate, check {{ .URL }} for more details")
	}
	if options.skipAnnounce {
		skips.Set(ctx, skips.Announce)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-announce was deprecated in favor of --skip=announce, check {{ .URL }} for more details")
	}
	if options.skipSBOMCataloging {
		skips.Set(ctx, skips.SBOM)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-sbom was deprecated in favor of --skip=sbom, check {{ .URL }} for more details")
	}
	if options.skipDocker {
		skips.Set(ctx, skips.Docker)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-docker was deprecated in favor of --skip=docker, check {{ .URL }} for more details")
	}
	if options.skipKo {
		skips.Set(ctx, skips.Ko)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-ko was deprecated in favor of --skip=ko, check {{ .URL }} for more details")
	}
	if options.skipBefore {
		skips.Set(ctx, skips.Before)
		deprecate.NoticeCustom(ctx, "-skip", "--skip-before was deprecated in favor of --skip=before, check {{ .URL }} for more details")
	}
	if options.rmDist {
		deprecate.NoticeCustom(ctx, "-rm-dist", "--rm-dist was deprecated in favor of --clean, check {{ .URL }} for more details")
	}

	if ctx.Snapshot {
		skips.Set(ctx, skips.Publish, skips.Announce, skips.Validate)
	}
	if skips.Any(ctx, skips.Publish) {
		skips.Set(ctx, skips.Announce)
	}

	if skips.Any(ctx, skips.Release...) {
		log.Warnf(
			logext.Warning("skipping %s..."),
			skips.String(ctx),
		)
	}
	return nil
}
