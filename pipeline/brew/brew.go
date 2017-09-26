// Package brew implements the Pipe, providing formula generation and
// uploading it to a configured repo.
package brew

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/archiveformat"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/pipeline"
)

// ErrNoDarwin64Build when there is no build for darwin_amd64 (goos doesn't
// contain darwin and/or goarch doesn't contain amd64)
var ErrNoDarwin64Build = errors.New("brew tap requires a darwin amd64 build")

const platform = "darwinamd64"

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating homebrew formula"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client, err := client.NewGitHub(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, client)
}

func doRun(ctx *context.Context, client client.Client) error {
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	if ctx.Config.Brew.GitHub.Name == "" {
		return pipeline.Skip("brew section is not configured")
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	if ctx.Config.Archive.Format == "binary" {
		return pipeline.Skip("archive format is binary")
	}

	var group = ctx.Binaries["darwinamd64"]
	if group == nil {
		return ErrNoDarwin64Build
	}
	var folder string
	for f := range group {
		folder = f
		break
	}
	var path = filepath.Join(ctx.Config.Brew.Folder, ctx.Config.ProjectName+".rb")
	log.WithField("formula", path).
		WithField("repo", ctx.Config.Brew.GitHub.String()).
		Info("pushing")
	content, err := buildFormula(ctx, client, folder)
	if err != nil {
		return err
	}
	return client.CreateFile(ctx, content, path)
}

func buildFormula(ctx *context.Context, client client.Client, folder string) (bytes.Buffer, error) {
	data, err := dataFor(ctx, client, folder)
	if err != nil {
		return bytes.Buffer{}, err
	}
	return doBuildFormula(data)
}

func doBuildFormula(data templateData) (out bytes.Buffer, err error) {
	tmpl, err := template.New(data.Name).Parse(formulaTemplate)
	if err != nil {
		return out, err
	}
	err = tmpl.Execute(&out, data)
	return
}

func dataFor(ctx *context.Context, client client.Client, folder string) (result templateData, err error) {
	var file = folder + "." + archiveformat.For(ctx, platform)
	sum, err := checksum.SHA256(filepath.Join(ctx.Config.Dist, file))
	if err != nil {
		return
	}
	var url = "https://github.com"
	if ctx.Config.GitHubURLs.Download != "" {
		url = ctx.Config.GitHubURLs.Download
	}
	return templateData{
		Name:         formulaNameFor(ctx.Config.ProjectName),
		DownloadURL:  url,
		Desc:         ctx.Config.Brew.Description,
		Homepage:     ctx.Config.Brew.Homepage,
		Repo:         ctx.Config.Release.GitHub,
		Tag:          ctx.Git.CurrentTag,
		Version:      ctx.Version,
		Caveats:      ctx.Config.Brew.Caveats,
		File:         file,
		SHA256:       sum,
		Dependencies: ctx.Config.Brew.Dependencies,
		Conflicts:    ctx.Config.Brew.Conflicts,
		Plist:        ctx.Config.Brew.Plist,
		Install:      split(ctx.Config.Brew.Install),
		Tests:        split(ctx.Config.Brew.Test),
	}, nil
}

func split(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
