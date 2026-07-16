package cmd

import (
	stdctx "context"
	"fmt"
	"runtime"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipeline"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
	draft             bool
	failFast          bool
	clean             bool
	deprecated        bool
	parallelism       int
	timeout           time.Duration
	skips             []string
}

func newReleaseCmd() *releaseCmd {
	root := &releaseCmd{}
	//nolint:dupl
	cmd := &cobra.Command{
		Use:               "release",
		Aliases:           []string{"r"},
		Short:             "Releases the current project",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return releaseProject(cmd.Context(), root.opts)
		},
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
	cmd.Flags().BoolVar(&root.opts.draft, "draft", false, "Whether to set the release to draft. Overrides release.draft in the configuration file")
	cmd.Flags().BoolVar(&root.opts.failFast, "fail-fast", false, "Whether to abort the release publishing on the first error")
	cmd.Flags().BoolVar(&root.opts.clean, "clean", false, "Removes the 'dist' directory")
	cmd.Flags().IntVarP(&root.opts.parallelism, "parallelism", "p", 0, "Amount tasks to run concurrently (default: number of CPUs)")
	_ = cmd.RegisterFlagCompletionFunc("parallelism", cobra.NoFileCompletions)
	cmd.Flags().DurationVar(&root.opts.timeout, "timeout", time.Hour, "Timeout to the entire release process")
	_ = cmd.RegisterFlagCompletionFunc("timeout", cobra.NoFileCompletions)
	cmd.Flags().BoolVar(&root.opts.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")
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

func printReleaseSummary(ctx *context.Context) {
	log.Infof(boldStyle.Render("release summary"))

	log.IncreasePadding()

	assets := ctx.Artifacts.Filter(artifact.ByTypes(artifact.ReleaseUploadableTypes()...)).List()
	if len(assets) > 0 {
		log.Infof("Published %s to %s with %d assets.", ctx.Git.CurrentTag, ctx.Git.URL, len(assets))
	}

	if !skips.Any(ctx, skips.Homebrew) && len(ctx.Config.Casks) > 0 {
		for _, brew := range ctx.Config.Casks {
			log.Infof("Published %s to %s/%s", brew.Name, brew.Repository.Owner, brew.Repository.Name)
		}
	}

	// Docker v1 is deprecated, but we check it just in case
	if !skips.Any(ctx, skips.Docker) && len(ctx.Config.Dockers) > 0 {
		for _, docker := range ctx.Config.Dockers {
			// Show the image templates if available, otherwise the ID
			if len(docker.ImageTemplates) > 0 {
				log.Infof("Pushed docker images: %s", docker.ImageTemplates[0])
			} else {
				log.Infof("Pushed Docker images (%s)", docker.ID)
			}
		}
	}

	if !skips.Any(ctx, skips.Docker) && len(ctx.Config.DockersV2) > 0 {
		for _, docker := range ctx.Config.DockersV2 {
			if len(docker.Images) > 0 {
				log.Infof("Pushed docker images: %s", docker.Images[0])
			} else {
				log.Infof("Pushed docker images (%s)", docker.ID)
			}
		}
	}

	if !skips.Any(ctx, skips.Ko) && len(ctx.Config.Kos) > 0 {
		for _, ko := range ctx.Config.Kos {
			if len(ko.Repositories) > 0 {
				log.Infof("Pushed Ko images to %s", ko.Repositories[0])
			} else {
				log.Infof("Pushed Ko images (%s)", ko.ID)
			}
		}
	}

	if !skips.Any(ctx, skips.Winget) && len(ctx.Config.Winget) > 0 {
		for _, winget := range ctx.Config.Winget {
			log.Infof("Opened PR to %s/%s", winget.Repository.Owner, winget.Repository.Name)
		}
	}

	if !skips.Any(ctx, skips.Chocolatey) && len(ctx.Config.Chocolateys) > 0 {
		for _, choc := range ctx.Config.Chocolateys {
			log.Infof("Published %s to Chocolatey", choc.Name)
		}
	}

	if !skips.Any(ctx, skips.AUR) && len(ctx.Config.AURs) > 0 {
		for _, aur := range ctx.Config.AURs {
			log.Infof("Pushed %s to AUR", aur.Name)
		}
	}

	if !skips.Any(ctx, skips.Nix) && len(ctx.Config.Nix) > 0 {
		for _, nix := range ctx.Config.Nix {
			log.Infof("Published nix package in %s/%s", nix.Repository.Owner, nix.Repository.Name)
		}
	}

	if !skips.Any(ctx, skips.Scoop) && len(ctx.Config.Scoops) > 0 {
		for _, scoop := range ctx.Config.Scoops {
			log.Infof("Updated scoop manifest in %s/%s", scoop.Repository.Owner, scoop.Repository.Name)
		}
	}

	// There doesn't appear to be a skip for Krew, so we just check if the config is not empty.
	if len(ctx.Config.Krews) > 0 {
		for _, krew := range ctx.Config.Krews {
			log.Infof("Updated krew manifest in %s/%s", krew.Repository.Owner, krew.Repository.Name)
		}
	}

	if !skips.Any(ctx, skips.MCP) && ctx.Config.MCP.Name != "" {
		log.Infof("Published to MCP registry: %s", ctx.Config.MCP.Name)
	}

	if len(ctx.Config.Blobs) > 0 {
		for _, blob := range ctx.Config.Blobs {
			log.Infof("Uploaded artifacts to blob storage: %s", blob.Bucket)
		}
	}

	if len(ctx.Config.Artifactories) > 0 {
		for _, art := range ctx.Config.Artifactories {
			target, err := tmpl.New(ctx).Apply(art.Target)
			if err != nil {
				target = art.Target // fallback to raw template if rendering fails
			}
			log.Infof("Uploaded artifacts to Artifactory: %s", target)
		}
	}

	if len(ctx.Config.Uploads) > 0 {
		for _, upload := range ctx.Config.Uploads {
			target, err := tmpl.New(ctx).Apply(upload.Target)
			if err != nil {
				target = upload.Target // fallback to raw template if rendering fails
			}
			log.Infof("Uploaded %s to %s", upload.Name, target)
		}
	}

	if len(ctx.Config.Publishers) > 0 {
		for _, pub := range ctx.Config.Publishers {
			log.Infof("Executed custom publisher: %s", pub.Name)
		}
	}

	if len(ctx.Config.Milestones) > 0 {
		for _, milestone := range ctx.Config.Milestones {
			log.Infof("Closed milestone in %s/%s", milestone.Repo.Owner, milestone.Repo.Name)
		}
	}

	log.DecreasePadding()
}

func releaseProject(parent stdctx.Context, options releaseOpts) error {
	start := time.Now()
	log.Infof(boldStyle.Render("starting release"))
	cfg, err := loadConfig(!options.snapshot, options.config)
	if err != nil {
		return decorateWithCtxErr(parent, err, "release", after(start))
	}

	ctx, cancel := context.WrapWithTimeout(parent, cfg, options.timeout)
	defer cancel()

	if err := setupReleaseContext(ctx, options); err != nil {
		return decorateWithCtxErr(ctx, err, "release", after(start))
	}
	for _, pipe := range pipeline.Pipeline {
		if err := skip.Maybe(
			pipe,
			logging.Log(
				pipe.String(),
				errhandler.Handle(pipe.Run),
			),
		)(ctx); err != nil {
			return decorateWithCtxErr(ctx, err, "release", after(start))
		}
	}

	deprecateWarn(ctx)
	log.Infof(boldStyle.Render(fmt.Sprintf("release succeeded after %s", after(start))))
	printReleaseSummary(ctx)
	return nil
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
	ctx.Clean = options.clean
	if options.autoSnapshot && git.CheckDirty(ctx) != nil {
		log.Info("git repository is dirty and --auto-snapshot is set, implying --snapshot")
		ctx.Snapshot = true
	}

	if options.draft {
		ctx.Config.Release.Draft = true
	}

	if err := skips.SetRelease(ctx, options.skips...); err != nil {
		return err
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
