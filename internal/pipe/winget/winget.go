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
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var (
	errNoRepoName               = pipe.Skip("winget.repository.name name is required")
	errNoPublisher              = pipe.Skip("winget.publisher is required")
	errNoLicense                = pipe.Skip("winget.license is required")
	errNoShortDescription       = pipe.Skip("winget.short_description is required")
	errInvalidPackageIdentifier = pipe.Skip("winget.package_identifier is invalid")
	errSkipUpload               = pipe.Skip("winget.skip_upload is set")
	errSkipUploadAuto           = pipe.Skip("winget.skip_upload is set to 'auto', and current version is a pre-release")
	errMultipleArchives         = pipe.Skip("found multiple archives for the same platform, please consider filtering by id")
	packageIdentifierValid      = regexp.MustCompile("^[^\\.\\s\\\\/:\\*\\?\"<>\\|\\x01-\\x1f]{1,32}(\\.[^\\.\\s\\\\/:\\*\\?\"<>\\|\\x01-\\x1f]{1,32}){1,7}$")
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

func (Pipe) String() string { return "winget" }
func (p Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.Winget) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Winget {
		winget := &ctx.Config.Winget[i]

		winget.CommitAuthor = commitauthor.Default(winget.CommitAuthor)

		if winget.CommitMessageTemplate == "" {
			winget.CommitMessageTemplate = "{{ .ProjectName }}: {{ .PreviousTag }} -> {{ .Tag }}"
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
	cli, err := client.New(ctx)
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

func (p Pipe) runAll(ctx *context.Context, cli client.Client) error {
	for _, winget := range ctx.Config.Winget {
		err := p.doRun(ctx, winget, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Pipe) doRun(ctx *context.Context, winget config.Winget, cl client.ReleaserURLTemplater) error {
	if winget.Repository.Name == "" {
		return errNoRepoName
	}

	publisher, err := tmpl.New(ctx).Apply(winget.Publisher)
	if err != nil {
		return err
	}
	if publisher == "" {
		return errNoPublisher
	}
	winget.Publisher = publisher

	if winget.License == "" {
		return errNoLicense
	}

	name, err := tmpl.New(ctx).Apply(winget.Name)
	if err != nil {
		return err
	}
	winget.Name = name

	author, err := tmpl.New(ctx).Apply(winget.Author)
	if err != nil {
		return err
	}
	winget.Author = author

	publisherURL, err := tmpl.New(ctx).Apply(winget.PublisherURL)
	if err != nil {
		return err
	}
	winget.PublisherURL = publisherURL

	homepage, err := tmpl.New(ctx).Apply(winget.Homepage)
	if err != nil {
		return err
	}
	winget.Homepage = homepage

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, winget.Repository)
	if err != nil {
		return err
	}
	winget.Repository = ref

	skipUpload, err := tmpl.New(ctx).Apply(winget.SkipUpload)
	if err != nil {
		return err
	}
	winget.SkipUpload = skipUpload

	description, err := tmpl.New(ctx).Apply(winget.Description)
	if err != nil {
		return err
	}
	winget.Description = description

	shortDescription, err := tmpl.New(ctx).Apply(winget.ShortDescription)
	if err != nil {
		return err
	}
	winget.ShortDescription = shortDescription

	if winget.ShortDescription == "" {
		return errNoShortDescription
	}

	releaseNotesURL, err := tmpl.New(ctx).Apply(winget.ReleaseNotesURL)
	if err != nil {
		return err
	}
	winget.ReleaseNotesURL = releaseNotesURL

	if winget.URLTemplate == "" {
		url, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return err
		}
		winget.URLTemplate = url
	}

	path, err := tmpl.New(ctx).Apply(winget.Path)
	if err != nil {
		return err
	}

	if path == "" {
		path = filepath.Join("manifests", strings.ToLower(string(winget.Publisher[0])), winget.Publisher, winget.Name, ctx.Version)
	}
	winget.Path = path

	filters := []artifact.Filter{
		artifact.ByGoos("windows"),
		artifact.ByFormats("zip"),
		artifact.ByType(artifact.UploadableArchive),
		artifact.Or(
			artifact.ByGoarch("386"),
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
		winget.PackageIdentifier = publisher + "." + name
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
	}

	var amd64Count, i386count int
	for _, archive := range archives {
		sha256, err := archive.Checksum("sha256")
		if err != nil {
			return err
		}
		var files []InstallerItemFile
		folder := artifact.ExtraOr(*archive, artifact.ExtraWrappedIn, ".")
		for _, bin := range artifact.ExtraOr(*archive, artifact.ExtraBinaries, []string{}) {
			files = append(files, InstallerItemFile{
				RelativeFilePath: windowsJoin([2]string{folder, bin}),
			})
		}
		url, err := tmpl.New(ctx).WithArtifact(archive).Apply(winget.URLTemplate)
		if err != nil {
			return err
		}
		installer.Installers = append(installer.Installers, InstallerItem{
			Architecture:         fromGoArch[archive.Goarch],
			NestedInstallerType:  "portable",
			NestedInstallerFiles: files,
			InstallerURL:         url,
			InstallerSha256:      sha256,
			UpgradeBehavior:      "uninstallPrevious",
		})
		switch archive.Goarch {
		case "386":
			i386count++
		case "amd64":
			amd64Count++
		}
	}

	if i386count > 1 || amd64Count > 1 {
		return errMultipleArchives
	}

	if err := createYAML(ctx, winget, installer, artifact.WingetInstaller); err != nil {
		return err
	}

	return createYAML(ctx, winget, Locale{
		PackageIdentifier: winget.PackageIdentifier,
		PackageVersion:    ctx.Version,
		PackageLocale:     defaultLocale,
		Publisher:         publisher,
		PublisherURL:      winget.PublisherURL,
		Author:            author,
		PackageName:       name,
		PackageURL:        winget.Homepage,
		License:           winget.License,
		LicenseURL:        winget.LicenseURL,
		Copyright:         winget.Copyright,
		ShortDescription:  shortDescription,
		Description:       description,
		Moniker:           name,
		Tags:              []string{},
		ReleaseNotes:      ctx.ReleaseNotes,
		ReleaseNotesURL:   winget.ReleaseNotesURL,
		ManifestType:      "defaultLocale",
		ManifestVersion:   manifestVersion,
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

	msg, err := tmpl.New(ctx).Apply(winget.CommitMessageTemplate)
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
			Content: content,
			Path:    filepath.Join(winget.Path, pkg.Name),
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

	for _, file := range files {
		if err := cl.CreateFile(ctx, author, repo, file.Content, file.Path, msg); err != nil {
			return err
		}
	}

	if !winget.Repository.PullRequest.Enabled {
		log.Debug("winget.pull_request disabled")
		return nil
	}

	log.Info("winget.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	title := fmt.Sprintf("Updated %s to %s", ctx.Config.ProjectName, ctx.Version)
	return pcl.OpenPullRequest(ctx, client.Repo{
		Name:   winget.Repository.PullRequest.Base.Name,
		Owner:  winget.Repository.PullRequest.Base.Owner,
		Branch: winget.Repository.PullRequest.Base.Branch,
	}, repo, title, winget.Repository.PullRequest.Draft)
}

func extFor(tp artifact.Type) string {
	switch tp {
	case artifact.WingetVersion:
		return ".yaml"
	case artifact.WingetInstaller:
		return ".installer.yaml"
	case artifact.WingetDefaultLocale:
		return "." + defaultLocale + ".yaml"
	default:
		// should never happen
		return ""
	}
}

func windowsJoin(elem [2]string) string {
	if elem[0] == "" {
		return elem[1]
	}
	return elem[0] + "\\" + elem[1]
}
