package winget

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/internal/yaml"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var (
	errNoRepoName     = pipe.Skip("repository name is not set")
	errSkipUpload     = pipe.Skip("winget.skip_upload is set")
	errSkipUploadAuto = pipe.Skip("winget.skip_upload is set to 'auto', and current version is a pre-release")
)

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

	publisher, err := tmpl.New(ctx).Apply(winget.Publisher)
	if err != nil {
		return err
	}
	winget.Publisher = publisher

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

	version := Version{
		PackageIdentifier: name,
		PackageVersion:    ctx.Version,
		DefaultLocale:     defaultLocale,
		ManifestType:      "version",
		ManifestVersion:   manifestVersion,
	}
	versionContent, err := yaml.Marshal(version)
	if err != nil {
		return err
	}

	filename := winget.Name + ".yaml"
	path := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("winget version", path).Info("writing")
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		generatedHeader,
		versionLangServer,
		string(versionContent),
	}, "\n")), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write winget version: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.WingetVersion,
		Extra: map[string]interface{}{
			wingetConfigExtra: winget,
		},
	})

	installer := Installer{
		PackageIdentifier: name,
		PackageVersion:    ctx.Version,
		InstallerLocale:   defaultLocale,
		InstallerType:     "zip",
		Commands:          []string{},
		ReleaseDate:       ctx.Date.Format(time.DateOnly),
		Installers:        []InstallerItem{},
		ManifestType:      "installer",
		ManifestVersion:   manifestVersion,
	}

	for _, archive := range ctx.Artifacts.Filter(
		artifact.And(
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
		),
	).List() {
		sha256, err := archive.Checksum("sha256")
		if err != nil {
			return err
		}
		var files []InstallerItemFile
		for _, bin := range artifact.ExtraOr(*archive, artifact.ExtraBinaries, []string{}) {
			files = append(files, InstallerItemFile{
				RelativeFilePath: bin,
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
			InstallerUrl:         url,
			InstallerSha256:      sha256,
			UpgradeBehavior:      "uninstallPrevious",
		})
	}

	installerContent, err := yaml.Marshal(installer)
	if err != nil {
		return err
	}
	filename = winget.Name + ".installer.yaml"
	path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("winget installer", path).Info("writing")
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		generatedHeader,
		installerLangServer,
		string(installerContent),
	}, "\n")), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write winget installer: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.WingetInstaller,
		Extra: map[string]interface{}{
			wingetConfigExtra: winget,
		},
	})

	locale := Locale{
		PackageIdentifier: name,
		PackageVersion:    ctx.Version,
		PackageLocale:     defaultLocale,
		Publisher:         publisher,
		PublisherUrl:      winget.PublisherURL,
		Author:            author,
		PackageName:       name,
		PackageUrl:        homepage,
		License:           winget.License,
		LicenseUrl:        winget.LicenseURL,
		Copyright:         winget.Copyright,
		ShortDescription:  shortDescription,
		Description:       description,
		Moniker:           name,
		Tags:              []string{},
		ReleaseNotes:      ctx.ReleaseNotes,
		ReleaseNotesUrl:   "TODO",
		ManifestType:      "defaultLocale",
		ManifestVersion:   manifestVersion,
	}

	localeContent, err := yaml.Marshal(locale)
	if err != nil {
		return err
	}
	filename = winget.Name + "." + defaultLocale + ".yaml"
	path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("winget locale", path).Info("writing")
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		generatedHeader,
		defaultLocaleLangServer,
		string(localeContent),
	}, "\n")), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write winget locale: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.WingetDefaultLocale,
		Extra: map[string]interface{}{
			wingetConfigExtra: winget,
		},
	})

	return nil
}

func (p Pipe) publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, winget := range ctx.Artifacts.Filter(artifact.Or(
		artifact.ByType(artifact.WingetInstaller),
		artifact.ByType(artifact.WingetVersion),
		artifact.ByType(artifact.WingetDefaultLocale),
	)).GroupByID() {
		err := doPublish(ctx, cli, winget)
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

func doPublish(ctx *context.Context, cl client.Client, pkgs []*artifact.Artifact) error {
	winget, err := artifact.Extra[config.Winget](*pkgs[0], wingetConfigExtra)
	if err != nil {
		return err
	}

	if strings.TrimSpace(winget.SkipUpload) == "true" {
		return errSkipUpload
	}

	if strings.TrimSpace(winget.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return errSkipUploadAuto
	}

	repo := client.RepoFromRef(winget.Repository)

	gpath := winget.Path

	msg, err := tmpl.New(ctx).Apply(winget.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, winget.CommitAuthor)
	if err != nil {
		return err
	}

	var files []client.RepoFile
	for _, pkg := range pkgs {
		content, err := os.ReadFile(pkg.Path)
		if err != nil {
			return err
		}
		files = append(files, client.RepoFile{
			Content: content,
			Path:    filepath.Join(gpath, pkg.Name),
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

	// XXX: how bad is to create one commit for each file? probably bad, right?
	// github api does not seem to allow to create multiple files in a single commit though...
	// maybe support only plain git repositories instead?? after all, it should also work 🤔
	if !winget.Repository.PullRequest.Enabled {
		return cl.CreateFiles(ctx, author, repo, msg, files)
	}

	log.Info("winget.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	if err := cl.CreateFiles(ctx, author, repo, msg, files); err != nil {
		return err
	}

	title := fmt.Sprintf("Updated %s to %s", ctx.Config.ProjectName, ctx.Version)
	return pcl.OpenPullRequest(ctx, client.Repo{
		Name:   winget.Repository.PullRequest.Base.Name,
		Owner:  winget.Repository.PullRequest.Base.Owner,
		Branch: winget.Repository.PullRequest.Base.Branch,
	}, repo, title, winget.Repository.PullRequest.Draft)
}

const (
	manifestVersion         = "1.5.0"
	versionLangServer       = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.5.0.schema.json"
	installerLangServer     = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.5.0.schema.json"
	defaultLocaleLangServer = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.5.0.schema.json"
	defaultLocale           = "en-US"
	generatedHeader         = `# This file was generated by GoReleaser. DO NOT EDIT.`
)

type Version struct {
	PackageIdentifier string
	PackageVersion    string
	DefaultLocale     string
	ManifestType      string
	ManifestVersion   string
}

type InstallerItemFile struct {
	RelativeFilePath     string
	PortableCommandAlias string
}

type InstallerItem struct {
	Architecture         string
	NestedInstallerType  string
	NestedInstallerFiles []InstallerItemFile
	InstallerUrl         string
	InstallerSha256      string
	UpgradeBehavior      string
}

type Installer struct {
	PackageIdentifier string
	PackageVersion    string
	InstallerLocale   string // en-us
	InstallerType     string // zip
	Commands          []string
	ReleaseDate       string
	Installers        []InstallerItem
	ManifestType      string
	ManifestVersion   string
}

type Locale struct {
	PackageIdentifier string
	PackageVersion    string
	PackageLocale     string
	Publisher         string
	PublisherUrl      string
	Author            string
	PackageName       string
	PackageUrl        string
	License           string
	LicenseUrl        string
	Copyright         string
	ShortDescription  string
	Description       string
	Moniker           string
	Tags              []string
	ReleaseNotes      string
	ReleaseNotesUrl   string
	ManifestType      string
	ManifestVersion   string
}

var fromGoArch = map[string]string{
	"amd64": "x64",
	"386":   "x86",
}
