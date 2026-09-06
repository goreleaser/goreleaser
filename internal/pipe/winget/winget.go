package winget

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/commitauthor"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/summary"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

var (
	errNoRepoName                = pipe.Skip("winget.repository.name is required")
	errNoPublisher               = pipe.Skip("winget.publisher is required")
	errNoLicense                 = pipe.Skip("winget.license is required")
	errNoShortDescription        = pipe.Skip("winget.short_description is required")
	errInvalidPackageIdentifier  = pipe.Skip("winget.package_identifier is invalid")
	errSkipUpload                = pipe.Skip("winget.skip_upload is set")
	errSkipUploadAuto            = pipe.Skip("winget.skip_upload is set to 'auto', and current version is a pre-release")
	errMultipleArchives          = pipe.Skip("found multiple archives for the same platform, please consider filtering by id")
	errMixedFormats              = pipe.Skip("found archives with multiple formats (.exe and .zip)")
	errAdditionalLocaleEmpty     = pipe.Skip("winget.additional_locales.locale is empty")
	errAdditionalLocaleDuplicate = pipe.Skip("winget.additional_locales contains duplicate locales")
	errAdditionalLocaleIsDefault = pipe.Skip("winget.additional_locales.locale must not equal default_locale")

	// copied from winget src
	packageIdentifierValid = regexp.MustCompile("^[^\\.\\s\\\\/:\\*\\?\"<>\\|\\x01-\\x1f]{1,32}(\\.[^\\.\\s\\\\/:\\*\\?\"<>\\|\\x01-\\x1f]{1,32}){1,7}$")
)

type errNoArchivesFound struct {
	goamd64 string
	ids     []string
}

func (e errNoArchivesFound) Error() string {
	return fmt.Sprintf("no zip archives found matching goos=[windows] goarch=[amd64 386] goamd64=%s ids=%v", e.goamd64, e.ids)
}

const (
	wingetConfigExtra = "WingetConfig"
	wingetLocaleExtra = "WingetLocale"
)

type Pipe struct{}

func (Pipe) String() string        { return "winget" }
func (Pipe) ContinueOnError() bool { return true }
func (p Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Winget) || len(ctx.Config.Winget) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Winget {
		winget := &ctx.Config.Winget[i]

		winget.CommitAuthor = commitauthor.Default(winget.CommitAuthor)

		if winget.CommitMessageTemplate == "" {
			winget.CommitMessageTemplate = "New version: {{ .PackageIdentifier }} {{ .Version }}"
		}
		if winget.Name == "" {
			winget.Name = ctx.Config.ProjectName
		}
		if winget.Goamd64 == "" {
			winget.Goamd64 = "v1"
		}
		winget.DefaultLocale = cmp.Or(winget.DefaultLocale, defaultLocale)
		winget.PackageName = cmp.Or(winget.PackageName, winget.Name)
	}

	return nil
}

func (p Pipe) Run(ctx *context.Context) error {
	cli, err := client.NewReleaseClient(ctx)
	if err != nil {
		return err
	}

	return p.runAll(ctx, cli)
}

// Publish .
func (p Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return p.publishAll(ctx, cli)
}

func (p Pipe) runAll(ctx *context.Context, cli client.ReleaseURLTemplater) error {
	// even if one of them is skipped, we still go through all of them, and
	// return the skips all at once in the end.
	skips := pipe.SkipMemento{}
	for i, winget := range ctx.Config.Winget {
		err := p.doRun(ctx, winget, cli, fmt.Sprintf("winget-%d", i))
		if err != nil && pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skips.Evaluate()
}

func (p Pipe) doRun(ctx *context.Context, winget config.Winget, cl client.ReleaseURLTemplater, artifactID string) error {
	if winget.Repository.Name == "" {
		return errNoRepoName
	}

	tp := tmpl.New(ctx)

	err := tp.ApplyAll(
		&winget.Publisher,
		&winget.Name,
		&winget.PackageIdentifier,
		&winget.PackageName,
		&winget.Author,
		&winget.PublisherURL,
		&winget.PublisherSupportURL,
		&winget.PrivacyURL,
		&winget.Homepage,
		&winget.SkipUpload,
		&winget.Description,
		&winget.ShortDescription,
		&winget.ReleaseNotesURL,
		&winget.InstallationNotes,
		&winget.Path,
		&winget.Copyright,
		&winget.CopyrightURL,
		&winget.License,
		&winget.LicenseURL,
		&winget.DefaultLocale,
	)
	if err != nil {
		return err
	}

	if winget.Publisher == "" {
		return errNoPublisher
	}

	if winget.License == "" {
		return errNoLicense
	}

	if winget.PackageIdentifier == "" {
		winget.PackageIdentifier = strings.ReplaceAll(winget.Publisher, " ", "") + "." + winget.Name
	}

	if !packageIdentifierValid.MatchString(winget.PackageIdentifier) {
		return fmt.Errorf("%w: %s", errInvalidPackageIdentifier, winget.PackageIdentifier)
	}

	if winget.Path == "" {
		winget.Path = path.Join(
			"manifests",
			strings.ToLower(string(winget.PackageIdentifier[0])),
			strings.ReplaceAll(winget.PackageIdentifier, ".", "/"),
			ctx.Version,
		)
	}

	winget.Repository, err = client.TemplateRef(tp.Apply, winget.Repository)
	if err != nil {
		return err
	}

	if winget.ShortDescription == "" {
		return errNoShortDescription
	}

	winget.ReleaseNotes, err = tp.WithExtraFields(tmpl.Fields{
		"Changelog": ctx.ReleaseNotes,
	}).Apply(winget.ReleaseNotes)
	if err != nil {
		return err
	}

	if winget.URLTemplate == "" {
		winget.URLTemplate, err = cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return err
		}
	}

	filters := []artifact.Filter{
		artifact.ByGoos("windows"),
		artifact.Or(
			artifact.And(
				artifact.ByFormats("zip"),
				artifact.ByType(artifact.UploadableArchive),
			),
			artifact.ByType(artifact.UploadableBinary),
		),
		artifact.Or(
			artifact.ByGoarch("386"),
			artifact.ByGoarch("arm64"),
			artifact.And(
				artifact.ByGoamd64(winget.Goamd64),
				artifact.ByGoarch("amd64"),
			),
		),
	}
	if len(winget.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(winget.IDs...))
	}
	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return errNoArchivesFound{
			goamd64: winget.Goamd64,
			ids:     winget.IDs,
		}
	}

	// Preflight the additional locales before creating any artifact so an
	// invalid locale cannot leave a partially registered (and published) set
	// of manifests behind.
	winget, err = p.prepareAdditionalLocales(ctx, winget)
	if err != nil {
		return err
	}

	installer, err := makeInstaller(ctx, winget, archives)
	if err != nil {
		return err
	}

	if err := createYAML(ctx, winget, artifactID, Version{
		PackageIdentifier: winget.PackageIdentifier,
		PackageVersion:    ctx.Version,
		DefaultLocale:     winget.DefaultLocale,
		ManifestType:      "version",
		ManifestVersion:   manifestVersion,
	}, artifact.WingetVersion, winget.DefaultLocale); err != nil {
		return err
	}

	if err := createYAML(ctx, winget, artifactID, installer, artifact.WingetInstaller, winget.DefaultLocale); err != nil {
		return err
	}

	if err := createYAML(ctx, winget, artifactID, Locale{
		PackageIdentifier:   winget.PackageIdentifier,
		PackageVersion:      ctx.Version,
		PackageLocale:       winget.DefaultLocale,
		Publisher:           winget.Publisher,
		PublisherURL:        winget.PublisherURL,
		PublisherSupportURL: winget.PublisherSupportURL,
		PrivacyURL:          winget.PrivacyURL,
		Author:              winget.Author,
		PackageName:         winget.PackageName,
		PackageURL:          winget.Homepage,
		License:             winget.License,
		LicenseURL:          winget.LicenseURL,
		Copyright:           winget.Copyright,
		CopyrightURL:        winget.CopyrightURL,
		ShortDescription:    winget.ShortDescription,
		Description:         strings.ReplaceAll(winget.Description, "\t", "  "),
		Moniker:             winget.Name,
		Tags:                fixTags(winget.Tags),
		ReleaseNotes:        winget.ReleaseNotes,
		ReleaseNotesURL:     winget.ReleaseNotesURL,
		InstallationNotes:   winget.InstallationNotes,
		ManifestType:        "defaultLocale",
		ManifestVersion:     manifestVersion,
	}, artifact.WingetDefaultLocale, winget.DefaultLocale); err != nil {
		return err
	}

	return p.doAdditionalLocales(ctx, winget, artifactID)
}

// prepareAdditionalLocales templates and validates every additional locale up
// front (before any artifact is created). It returns the prepared winget so an
// invalid locale cannot leave a partial set of manifests registered.
func (p Pipe) prepareAdditionalLocales(ctx *context.Context, winget config.Winget) (config.Winget, error) {
	tp := tmpl.New(ctx)
	seen := map[string]bool{}
	for i := range winget.AdditionalLocales {
		aloc := &winget.AdditionalLocales[i]

		if err := tp.ApplyAll(&aloc.Locale); err != nil {
			return winget, err
		}

		if aloc.Locale == "" {
			return winget, errAdditionalLocaleEmpty
		}

		if aloc.Locale == winget.DefaultLocale {
			return winget, errAdditionalLocaleIsDefault
		}

		if seen[aloc.Locale] {
			return winget, errAdditionalLocaleDuplicate
		}
		seen[aloc.Locale] = true

		if err := tp.ApplyAll(
			&aloc.Publisher,
			&aloc.PublisherURL,
			&aloc.PublisherSupportURL,
			&aloc.PrivacyURL,
			&aloc.Author,
			&aloc.PackageName,
			&aloc.Homepage,
			&aloc.License,
			&aloc.LicenseURL,
			&aloc.Copyright,
			&aloc.CopyrightURL,
			&aloc.ShortDescription,
			&aloc.Description,
			&aloc.ReleaseNotesURL,
			&aloc.InstallationNotes,
		); err != nil {
			return winget, err
		}

		if aloc.ReleaseNotes != "" {
			releaseNotes, err := tp.WithExtraFields(tmpl.Fields{
				"Changelog": ctx.ReleaseNotes,
			}).Apply(aloc.ReleaseNotes)
			if err != nil {
				return winget, err
			}
			aloc.ReleaseNotes = releaseNotes
		}
	}

	return winget, nil
}

// doAdditionalLocales renders the already validated additional locale
// manifests. It must only be called after prepareAdditionalLocales succeeded.
func (p Pipe) doAdditionalLocales(ctx *context.Context, winget config.Winget, artifactID string) error {
	for _, aloc := range winget.AdditionalLocales {
		tags := aloc.Tags
		if len(tags) == 0 {
			tags = winget.Tags
		}
		if err := createYAML(ctx, winget, artifactID, Locale{
			PackageIdentifier:   winget.PackageIdentifier,
			PackageVersion:      ctx.Version,
			PackageLocale:       aloc.Locale,
			Publisher:           cmp.Or(aloc.Publisher, winget.Publisher),
			PublisherURL:        cmp.Or(aloc.PublisherURL, winget.PublisherURL),
			PublisherSupportURL: cmp.Or(aloc.PublisherSupportURL, winget.PublisherSupportURL),
			PrivacyURL:          cmp.Or(aloc.PrivacyURL, winget.PrivacyURL),
			Author:              cmp.Or(aloc.Author, winget.Author),
			PackageName:         cmp.Or(aloc.PackageName, winget.PackageName),
			PackageURL:          cmp.Or(aloc.Homepage, winget.Homepage),
			License:             cmp.Or(aloc.License, winget.License),
			LicenseURL:          cmp.Or(aloc.LicenseURL, winget.LicenseURL),
			Copyright:           cmp.Or(aloc.Copyright, winget.Copyright),
			CopyrightURL:        cmp.Or(aloc.CopyrightURL, winget.CopyrightURL),
			ShortDescription:    cmp.Or(aloc.ShortDescription, winget.ShortDescription),
			Description:         strings.ReplaceAll(cmp.Or(aloc.Description, winget.Description), "\t", "  "),
			Moniker:             winget.Name,
			Tags:                fixTags(tags),
			ReleaseNotes:        cmp.Or(aloc.ReleaseNotes, winget.ReleaseNotes),
			ReleaseNotesURL:     cmp.Or(aloc.ReleaseNotesURL, winget.ReleaseNotesURL),
			InstallationNotes:   cmp.Or(aloc.InstallationNotes, winget.InstallationNotes),
			ManifestType:        "locale",
			ManifestVersion:     manifestVersion,
		}, artifact.WingetLocale, aloc.Locale); err != nil {
			return err
		}
	}
	return nil
}

func (p Pipe) publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, files := range ctx.Artifacts.Filter(artifact.ByTypes(
		artifact.WingetInstaller,
		artifact.WingetVersion,
		artifact.WingetDefaultLocale,
		artifact.WingetLocale,
	)).GroupByID() {
		err := doPublish(ctx, cli, files)
		if err != nil && pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skips.Evaluate()
}

func doPublish(ctx *context.Context, cl client.Client, wingets []*artifact.Artifact) error {
	winget := artifact.MustExtra[config.Winget](*wingets[0], wingetConfigExtra)
	if strings.TrimSpace(winget.SkipUpload) == "true" {
		return errSkipUpload
	}

	if strings.TrimSpace(winget.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return errSkipUploadAuto
	}

	msg, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"PackageIdentifier": winget.PackageIdentifier,
	}).Apply(winget.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, winget.CommitAuthor)
	if err != nil {
		return err
	}

	repo := client.RepoFromRef(winget.Repository)

	var files []client.RepoFile
	for _, pkg := range wingets {
		content, err := os.ReadFile(pkg.Path)
		if err != nil {
			return err
		}
		files = append(files, client.RepoFile{
			Content:    content,
			Path:       path.Join(winget.Path, pkg.Name),
			Identifier: repoFileID(pkg.Type),
		})
	}

	if winget.Repository.Git.URL != "" {
		if err := client.NewGitUploadClient(repo.Branch).
			CreateFiles(ctx, author, repo, msg, files); err != nil {
			return err
		}
		summary.Appendf("Updated winget package `%s` in `%s`", winget.PackageIdentifier, cmp.Or(repo.String(), winget.Repository.Git.URL))
		return nil
	}

	cl, err = client.NewIfToken(ctx, cl, winget.Repository.Token)
	if err != nil {
		return err
	}

	base := client.Repo{
		Name:   winget.Repository.PullRequest.Base.Name,
		Owner:  winget.Repository.PullRequest.Base.Owner,
		Branch: winget.Repository.PullRequest.Base.Branch,
	}

	// try to sync branch
	fscli, ok := cl.(client.ForkSyncer)
	if ok && winget.Repository.PullRequest.Enabled {
		if err := fscli.SyncFork(ctx, repo, base); err != nil {
			log.WithError(err).Warn("could not sync fork")
		}
	}

	for _, file := range files {
		if err := cl.CreateFile(
			ctx,
			author,
			repo,
			file.Content,
			file.Path,
			msg+": add "+file.Identifier,
		); err != nil {
			return err
		}
	}

	if !winget.Repository.PullRequest.Enabled {
		log.Debug("wingets.pull_request disabled")
		summary.Appendf("Updated winget package `%s` in `%s`", winget.PackageIdentifier, repo.String())
		return nil
	}

	log.Info("winget.pull_request enabled, creating a PR")
	prcl, err := client.NewIfToken(ctx, cl, winget.Repository.PullRequest.Token)
	if err != nil {
		return err
	}
	pcl, ok := prcl.(client.PullRequestOpener)
	if !ok {
		return errors.New("client does not support pull requests")
	}

	url, err := pcl.OpenPullRequest(ctx, base, repo, msg, winget.Repository.PullRequest.Draft)
	if err != nil {
		return err
	}
	if url != "" {
		summary.Appendf("Opened pull request to `%s` (winget package `%s`): %s", cmp.Or(base.String(), repo.String()), winget.PackageIdentifier, url)
	}
	return nil
}

func langserverLineFor(tp artifact.Type) string {
	switch tp {
	case artifact.WingetInstaller:
		return installerLangServer
	case artifact.WingetDefaultLocale:
		return defaultLocaleLangServer
	case artifact.WingetLocale:
		return localeLangServer
	default:
		return versionLangServer
	}
}

func extFor(tp artifact.Type, locale string) string {
	switch tp {
	case artifact.WingetVersion:
		return ".yaml"
	case artifact.WingetInstaller:
		return ".installer.yaml"
	case artifact.WingetDefaultLocale, artifact.WingetLocale:
		return ".locale." + locale + ".yaml"
	default:
		// should never happen
		return ""
	}
}

func repoFileID(tp artifact.Type) string {
	switch tp {
	case artifact.WingetVersion:
		return "version"
	case artifact.WingetInstaller:
		return "installer"
	case artifact.WingetDefaultLocale, artifact.WingetLocale:
		return "locale"
	default:
		// should never happen
		return ""
	}
}

func installerItemFilesFor(archive artifact.Artifact) []InstallerItemFile {
	var files []InstallerItemFile
	folder := artifact.ExtraOr(archive, artifact.ExtraWrappedIn, ".")
	for _, bin := range artifact.MustExtra[[]string](archive, artifact.ExtraBinaries) {
		files = append(files, InstallerItemFile{
			RelativeFilePath:     strings.ReplaceAll(filepath.Join(folder, bin), "/", "\\"),
			PortableCommandAlias: strings.TrimSuffix(filepath.Base(bin), ".exe"),
		})
	}
	return files
}

func makeInstaller(ctx *context.Context, winget config.Winget, archives []*artifact.Artifact) (Installer, error) {
	tp := tmpl.New(ctx)
	var deps []PackageDependency
	for _, dep := range winget.Dependencies {
		if err := tp.ApplyAll(&dep.MinimumVersion, &dep.PackageIdentifier); err != nil {
			return Installer{}, err
		}
		deps = append(deps, PackageDependency{
			PackageIdentifier: dep.PackageIdentifier,
			MinimumVersion:    dep.MinimumVersion,
		})
	}

	installer := Installer{
		PackageIdentifier: winget.PackageIdentifier,
		PackageVersion:    ctx.Version,
		InstallerLocale:   winget.DefaultLocale,
		InstallerType:     "zip",
		Commands:          []string{},
		ReleaseDate:       ctx.Date.Format(time.DateOnly),
		Installers:        []InstallerItem{},
		ManifestType:      "installer",
		ManifestVersion:   manifestVersion,
		Dependencies: Dependencies{
			PackageDependencies: deps,
		},
	}

	var zipCount, binaryCount int
	archCounts := map[string]int{}
	for _, archive := range archives {
		sha256, err := archive.Checksum("sha256")
		if err != nil {
			return Installer{}, err
		}
		url, err := tmpl.New(ctx).WithArtifact(archive).Apply(winget.URLTemplate)
		if err != nil {
			return Installer{}, err
		}
		item := InstallerItem{
			Architecture:    fromGoArch[archive.Goarch],
			InstallerURL:    url,
			InstallerSha256: sha256,
			UpgradeBehavior: "uninstallPrevious",
		}
		switch archive.Type {
		case artifact.UploadableArchive:
			if archive.Format() != "zip" {
				continue
			}
			zipCount++
			installer.InstallerType = "zip"
			item.NestedInstallerType = "portable"
			item.NestedInstallerFiles = installerItemFilesFor(*archive)
		case artifact.UploadableBinary:
			binaryCount++
			installer.InstallerType = "portable"
			cmd := artifact.MustExtra[string](*archive, artifact.ExtraBinary)
			installer.Commands = []string{cmd}
		}
		installer.Installers = append(installer.Installers, item)
		// a manifest may only have one installer per architecture.
		archCounts[item.Architecture]++
	}

	if binaryCount > 0 && zipCount > 0 {
		return Installer{}, errMixedFormats
	}

	for _, count := range archCounts {
		if count > 1 {
			return Installer{}, errMultipleArchives
		}
	}

	return installer, nil
}

func fixTags(in []string) []string {
	for i := range in {
		in[i] = strings.ReplaceAll(strings.ToLower(in[i]), " ", "-")
	}
	return in
}
