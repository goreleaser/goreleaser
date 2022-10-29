package chocolatey

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// nuget package extension.
const nupkgFormat = "nupkg"

// custom chocolatey config placed in artifact.
const chocoConfigExtra = "ChocolateyConfig"

// cmd represents a command executor.
var cmd cmder = stdCmd{}

// Pipe for chocolatey packaging.
type Pipe struct{}

func (Pipe) String() string                 { return "chocolatey packages" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Chocolateys) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Chocolateys {
		choco := &ctx.Config.Chocolateys[i]

		if choco.Name == "" {
			choco.Name = ctx.Config.ProjectName
		}

		if choco.Title == "" {
			choco.Title = ctx.Config.ProjectName
		}

		if choco.Goamd64 == "" {
			choco.Goamd64 = "v1"
		}

		if choco.SourceRepo == "" {
			choco.SourceRepo = "https://push.chocolatey.org/"
		}
	}

	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	client, err := client.New(ctx)
	if err != nil {
		return err
	}

	for _, choco := range ctx.Config.Chocolateys {
		if err := doRun(ctx, client, choco); err != nil {
			return err
		}
	}

	return nil
}

// Publish packages.
func (Pipe) Publish(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}

	artifacts := ctx.Artifacts.Filter(
		artifact.ByType(artifact.PublishableChocolatey),
	).List()

	for _, artifact := range artifacts {
		if err := doPush(ctx, artifact); err != nil {
			return err
		}
	}

	return nil
}

func doRun(ctx *context.Context, cl client.Client, choco config.Chocolatey) error {
	filters := []artifact.Filter{
		artifact.ByGoos("windows"),
		artifact.ByType(artifact.UploadableArchive),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(choco.Goamd64),
			),
			artifact.ByGoarch("386"),
		),
	}

	if len(choco.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(choco.IDs...))
	}

	artifacts := ctx.Artifacts.
		Filter(artifact.And(filters...)).
		List()

	if len(artifacts) == 0 {
		return errors.New("chocolatey requires a windows build and archive")
	}

	// folderDir is the directory that then will be compressed to make the
	// chocolatey package.
	folderPath := filepath.Join(ctx.Config.Dist, choco.Name+".choco")
	toolsPath := filepath.Join(folderPath, "tools")
	if err := os.MkdirAll(toolsPath, 0o755); err != nil {
		return err
	}

	nuspecFile := filepath.Join(folderPath, choco.Name+".nuspec")
	nuspec, err := buildNuspec(ctx, choco)
	if err != nil {
		return err
	}

	if err = os.WriteFile(nuspecFile, nuspec, 0o644); err != nil {
		return err
	}

	data, err := dataFor(ctx, cl, choco, artifacts)
	if err != nil {
		return err
	}

	script, err := buildTemplate(choco.Name, scriptTemplate, data)
	if err != nil {
		return err
	}

	scriptFile := filepath.Join(toolsPath, "chocolateyinstall.ps1")
	log.WithField("file", scriptFile).Debug("creating")
	if err = os.WriteFile(scriptFile, script, 0o644); err != nil {
		return err
	}

	log.WithField("nuspec", nuspecFile).Info("packing")
	out, err := cmd.Exec(ctx, "choco", "pack", nuspecFile, "--out", ctx.Config.Dist)
	if err != nil {
		return fmt.Errorf("failed to generate chocolatey package: %w: %s", err, string(out))
	}

	if choco.SkipPublish {
		return nil
	}

	pkgFile := fmt.Sprintf("%s.%s.%s", choco.Name, ctx.Version, nupkgFormat)

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.PublishableChocolatey,
		Path: filepath.Join(ctx.Config.Dist, pkgFile),
		Name: pkgFile,
		Extra: map[string]interface{}{
			artifact.ExtraFormat: nupkgFormat,
			chocoConfigExtra:     choco,
		},
	})

	return nil
}

func doPush(ctx *context.Context, art *artifact.Artifact) error {
	choco, err := artifact.Extra[config.Chocolatey](*art, chocoConfigExtra)
	if err != nil {
		return err
	}

	key, err := tmpl.New(ctx).Apply(choco.APIKey)
	if err != nil {
		return err
	}

	log := log.WithField("name", choco.Name)
	if key == "" {
		log.Warn("skip pushing: no api key")
		return nil
	}

	log.Info("pushing package")

	args := []string{
		"push",
		"--source",
		choco.SourceRepo,
		"--api-key",
		key,
		art.Path,
	}

	if out, err := cmd.Exec(ctx, "choco", args...); err != nil {
		return fmt.Errorf("failed to push chocolatey package: %w: %s", err, string(out))
	}

	log.Info("package sent")

	return nil
}

func buildNuspec(ctx *context.Context, choco config.Chocolatey) ([]byte, error) {
	tpl := tmpl.New(ctx)
	summary, err := tpl.Apply(choco.Summary)
	if err != nil {
		return nil, err
	}

	description, err := tpl.Apply(choco.Description)
	if err != nil {
		return nil, err
	}

	releaseNotes, err := tpl.Apply(choco.ReleaseNotes)
	if err != nil {
		return nil, err
	}

	m := &Nuspec{
		Xmlns: schema,
		Metadata: Metadata{
			ID:                       choco.Name,
			Version:                  ctx.Version,
			PackageSourceURL:         choco.PackageSourceURL,
			Owners:                   choco.Owners,
			Title:                    choco.Title,
			Authors:                  choco.Authors,
			ProjectURL:               choco.ProjectURL,
			IconURL:                  choco.IconURL,
			Copyright:                choco.Copyright,
			LicenseURL:               choco.LicenseURL,
			RequireLicenseAcceptance: choco.RequireLicenseAcceptance,
			ProjectSourceURL:         choco.ProjectSourceURL,
			DocsURL:                  choco.DocsURL,
			BugTrackerURL:            choco.BugTrackerURL,
			Tags:                     choco.Tags,
			Summary:                  summary,
			Description:              description,
			ReleaseNotes:             releaseNotes,
		},
		Files: Files{File: []File{
			{Source: "tools\\**", Target: "tools"},
		}},
	}

	deps := make([]Dependency, len(choco.Dependencies))
	for i, dep := range choco.Dependencies {
		deps[i] = Dependency{ID: dep.ID, Version: dep.Version}
	}

	if len(deps) > 0 {
		m.Metadata.Dependencies = &Dependencies{Dependency: deps}
	}

	return m.Bytes()
}

func buildTemplate(name string, text string, data templateData) ([]byte, error) {
	tp, err := template.New(name).Parse(text)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err = tp.Execute(&out, data); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func dataFor(ctx *context.Context, cl client.Client, choco config.Chocolatey, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{}

	if choco.URLTemplate == "" {
		url, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return result, err
		}

		choco.URLTemplate = url
	}

	for _, artifact := range artifacts {
		sum, err := artifact.Checksum("sha256")
		if err != nil {
			return result, err
		}

		url, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(choco.URLTemplate)
		if err != nil {
			return result, err
		}

		pkg := releasePackage{
			DownloadURL: url,
			Checksum:    sum,
			Arch:        artifact.Goarch,
		}

		result.Packages = append(result.Packages, pkg)
	}

	return result, nil
}

// cmder is a special interface to execute external commands.
//
// The intention is to be used to wrap the standard exec and provide the
// ability to create a fake one for testing.
type cmder interface {
	// Exec executes an command.
	Exec(*context.Context, string, ...string) ([]byte, error)
}

// stdCmd uses the standard golang exec.
type stdCmd struct{}

var _ cmder = &stdCmd{}

func (stdCmd) Exec(ctx *context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
