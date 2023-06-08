package winget

import (
	"fmt"
	"os"
	"path/filepath"
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
	errNoRepoName     = pipe.Skip("repository name is not set")
	errSkipUpload     = pipe.Skip("winget.skip_upload is set")
	errSkipUploadAuto = pipe.Skip("winget.skip_upload is set to 'auto', and current version is a pre-release")
)

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

	filename := winget.Name + ".yaml"
	path := filepath.Join(ctx.Config.Dist, filename)

	version := Version{
		PackageIdentifier: name,
		PackageVersion:    ctx.Version,
		DefaultLocale:     defaultLocale,
		ManifestType:      "version",
		ManifestVersion:   manifestVersion,
	}

	installer := Installer{
		PackageIdentifier: name,
		PackageVersion:    ctx.Version,
		InstallerLocale:   defaultLocale,
		InstallerType:     "portable",
		Commands:          []string{},
		ReleaseDate:       ctx.Date.Format(time.DateOnly),
		Installers:        []InstallerItem{},
		ManifestType:      "installer",
		ManifestVersion:   manifestVersion,
	}

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

	log.WithField("nixpkg", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write nixpkg: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.Nixpkg,
		Extra: map[string]interface{}{
			nixConfigExtra: nix,
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
	)).List() {
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

const (
	manifestVersion         = "1.5.0"
	versionLangServer       = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.5.0.schema.json"
	installerLangServer     = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.5.0.schema.json"
	defaultLocaleLangServer = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.5.0.schema.json"
	defaultLocale           = "en-US"
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
