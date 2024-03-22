package winget

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var (
	errNoRepoName               = pipe.Skip("winget.repository.name is required")
	errNoPublisher              = pipe.Skip("winget.publisher is required")
	errNoLicense                = pipe.Skip("winget.license is required")
	errNoShortDescription       = pipe.Skip("winget.short_description is required")
	errInvalidPackageIdentifier = pipe.Skip("winget.package_identifier is invalid")
	errSkipUpload               = pipe.Skip("winget.skip_upload is set")
	errSkipUploadAuto           = pipe.Skip("winget.skip_upload is set to 'auto', and current version is a pre-release")
	errMultipleArchives         = pipe.Skip("found multiple archives for the same platform, please consider filtering by id")
	errMixedFormats             = pipe.Skip("found archives with multiple formats (.exe and .zip)")

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

const wingetConfigExtra = "WingetConfig"

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
	for _, winget := range ctx.Config.Winget {
		err := p.doRun(ctx, winget, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Pipe) doRun(ctx *context.Context, winget config.Winget, cl client.ReleaseURLTemplater) error {
	if winget.Repository.Name == "" {
		return errNoRepoName
	}

	tp := tmpl.New(ctx)

	err := tp.ApplyAll(
		&winget.Publisher,
		&winget.Name,
		&winget.Author,
		&winget.PublisherURL,
		&winget.PublisherSupportURL,
		&winget.Homepage,
		&winget.SkipUpload,
		&winget.Description,
		&winget.ShortDescription,
		&winget.ReleaseNotesURL,
		&winget.Path,
		&winget.Copyright,
		&winget.CopyrightURL,
		&winget.License,
		&winget.LicenseURL,
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

	if winget.Path == "" {
		winget.Path = filepath.Join("manifests", strings.ToLower(string(winget.Publisher[0])), winget.Publisher, winget.Name, ctx.Version)
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

	if winget.PackageIdentifier == "" {
		winget.PackageIdentifier = winget.Publisher + "." + winget.Name
	}

	if !packageIdentifierValid.MatchString(winget.PackageIdentifier) {
		return fmt.Errorf("%w: %s", errInvalidPackageIdentifier, winget.PackageIdentifier)
	}

	if err := createYAML(ctx, winget, Version{
		PackageIdentifier: winget.PackageIdentifier,
		PackageVersion:    ctx.Version,
		DefaultLocale:     defaultLocale,
		ManifestType:      "version",
		ManifestVersion:   manifestVersion,
	}, artifact.WingetVersion); err != nil {
		return err
	}

	installer, err := makeInstaller(ctx, winget, archives)
	if err != nil {
		return err
	}

	if err := createYAML(ctx, winget, installer, artifact.WingetInstaller); err != nil {
		return err
	}

	return createYAML(ctx, winget, Locale{
		PackageIdentifier:   winget.PackageIdentifier,
		PackageVersion:      ctx.Version,
		PackageLocale:       defaultLocale,
		Publisher:           winget.Publisher,
		PublisherURL:        winget.PublisherURL,
		PublisherSupportURL: winget.PublisherSupportURL,
		Author:              winget.Author,
		PackageName:         winget.Name,
		PackageURL:          winget.Homepage,
		License:             winget.License,
		LicenseURL:          winget.LicenseURL,
		Copyright:           winget.Copyright,
		CopyrightURL:        winget.CopyrightURL,
		ShortDescription:    winget.ShortDescription,
		Description:         winget.Description,
		Moniker:             winget.Name,
		Tags:                winget.Tags,
		ReleaseNotes:        winget.ReleaseNotes,
		ReleaseNotesURL:     winget.ReleaseNotesURL,
		ManifestType:        "defaultLocale",
		ManifestVersion:     manifestVersion,
	}, artifact.WingetDefaultLocale)
}

func (p Pipe) publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, files := range ctx.Artifacts.Filter(artifact.Or(
		artifact.ByType(artifact.WingetInstaller),
		artifact.ByType(artifact.WingetVersion),
		artifact.ByType(artifact.WingetDefaultLocale),
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
	winget, err := artifact.Extra[config.Winget](*wingets[0], wingetConfigExtra)
	if err != nil {
		return err
	}

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
			Path:       filepath.Join(winget.Path, pkg.Name),
			Identifier: repoFileID(pkg.Type),
		})
	}

	if winget.Repository.Git.URL != "" {
		return client.NewGitUploadClient(repo.Branch).
			CreateFiles(ctx, author, repo, msg, files)
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
		return nil
	}

	log.Info("winget.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	return pcl.OpenPullRequest(ctx, base, repo, msg, winget.Repository.PullRequest.Draft)
}

func langserverLineFor(tp artifact.Type) string {
	switch tp {
	case artifact.WingetInstaller:
		return installerLangServer
	case artifact.WingetDefaultLocale:
		return defaultLocaleLangServer
	default:
		return versionLangServer
	}
}

func extFor(tp artifact.Type) string {
	switch tp {
	case artifact.WingetVersion:
		return ".yaml"
	case artifact.WingetInstaller:
		return ".installer.yaml"
	case artifact.WingetDefaultLocale:
		return ".locale." + defaultLocale + ".yaml"
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
	case artifact.WingetDefaultLocale:
		return "locale"
	default:
		// should never happen
		return ""
	}
}

func installerItemFilesFor(archive artifact.Artifact) []InstallerItemFile {
	var files []InstallerItemFile
	folder := artifact.ExtraOr(archive, artifact.ExtraWrappedIn, ".")
	for _, bin := range artifact.ExtraOr(archive, artifact.ExtraBinaries, []string{}) {
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
		InstallerLocale:   defaultLocale,
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

	var amd64Count, i386count, zipCount, binaryCount int
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
		if archive.Format() == "zip" {
			zipCount++
			installer.InstallerType = "zip"
			item.NestedInstallerType = "portable"
			item.NestedInstallerFiles = installerItemFilesFor(*archive)
		} else {
			binaryCount++
			installer.InstallerType = "portable"
			installer.Commands = []string{winget.Name}
		}
		installer.Installers = append(installer.Installers, item)
		switch archive.Goarch {
		case "386":
			i386count++
		case "amd64":
			amd64Count++
		}
	}

	if binaryCount > 0 && zipCount > 0 {
		return Installer{}, errMixedFormats
	}

	if i386count > 1 || amd64Count > 1 {
		return Installer{}, errMultipleArchives
	}

	return installer, nil
}
