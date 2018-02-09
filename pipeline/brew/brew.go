package brew

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apex/log"

	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/pipeline"
)

// ErrNoDarwin64Build when there is no build for darwin_amd64
var ErrNoDarwin64Build = errors.New("brew tap requires one darwin amd64 build")

// ErrTooManyDarwin64Builds when there are too many builds for darwin_amd64
var ErrTooManyDarwin64Builds = errors.New("brew tap requires at most one darwin amd64 build")

// Pipe for brew deployment
type Pipe struct{}

func (Pipe) String() string {
	return "creating homebrew formula"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client, err := client.NewGitHub(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, client)
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Brew.Install == "" {
		var installs []string
		for _, build := range ctx.Config.Builds {
			if !isBrewBuild(build) {
				continue
			}
			installs = append(
				installs,
				fmt.Sprintf(`bin.install "%s"`, build.Binary),
			)
		}
		ctx.Config.Brew.Install = strings.Join(installs, "\n")
	}

	if ctx.Config.Brew.CommitAuthor.Name == "" {
		ctx.Config.Brew.CommitAuthor.Name = "goreleaserbot"
	}
	if ctx.Config.Brew.CommitAuthor.Email == "" {
		ctx.Config.Brew.CommitAuthor.Email = "goreleaser@carlosbecker.com"
	}
	return nil
}

func isBrewBuild(build config.Build) bool {
	for _, ignore := range build.Ignore {
		if ignore.Goos == "darwin" && ignore.Goarch == "amd64" {
			return false
		}
	}
	return contains(build.Goos, "darwin") && contains(build.Goarch, "amd64")
}

func contains(ss []string, s string) bool {
	for _, zs := range ss {
		if zs == s {
			return true
		}
	}
	return false
}

func doRun(ctx *context.Context, client client.Client) error {
	if ctx.Config.Brew.GitHub.Name == "" {
		return pipeline.Skip("brew section is not configured")
	}
	if ctx.Config.Archive.Format == "binary" {
		return pipeline.Skip("archive format is binary")
	}

	var archives = ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("darwin"),
			artifact.ByGoarch("amd64"),
			artifact.ByGoarm(""),
			artifact.ByType(artifact.UploadableArchive),
		),
	).List()
	if len(archives) == 0 {
		return ErrNoDarwin64Build
	}
	if len(archives) > 1 {
		return ErrTooManyDarwin64Builds
	}

	content, err := buildFormula(ctx, client, archives[0])
	if err != nil {
		return err
	}

	var filename = ctx.Config.ProjectName + ".rb"
	var path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("formula", path).Info("writing")
	if err := ioutil.WriteFile(path, content.Bytes(), 0644); err != nil {
		return err
	}

	if ctx.Config.Brew.SkipUpload {
		return pipeline.Skip("brew.skip_upload is set")
	}
	if !ctx.Publish {
		return pipeline.ErrSkipPublish
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}

	path = filepath.Join(ctx.Config.Brew.Folder, filename)
	log.WithField("formula", path).
		WithField("repo", ctx.Config.Brew.GitHub.String()).
		Info("pushing")
	return client.CreateFile(ctx, ctx.Config.Brew.CommitAuthor, ctx.Config.Brew.GitHub, content, path)
}

func buildFormula(ctx *context.Context, client client.Client, artifact artifact.Artifact) (bytes.Buffer, error) {
	data, err := dataFor(ctx, client, artifact)
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

func dataFor(ctx *context.Context, client client.Client, artifact artifact.Artifact) (result templateData, err error) {
	sum, err := checksum.SHA256(artifact.Path)
	if err != nil {
		return
	}
	var url = "https://github.com"
	if ctx.Config.GitHubURLs.Download != "" {
		url = ctx.Config.GitHubURLs.Download
	}
	var cfg = ctx.Config.Brew
	return templateData{
		Name:             formulaNameFor(ctx.Config.ProjectName),
		DownloadURL:      url,
		Desc:             cfg.Description,
		Homepage:         cfg.Homepage,
		Repo:             ctx.Config.Release.GitHub,
		Tag:              ctx.Git.CurrentTag,
		Version:          ctx.Version,
		Caveats:          cfg.Caveats,
		File:             artifact.Name,
		SHA256:           sum,
		Dependencies:     cfg.Dependencies,
		Conflicts:        cfg.Conflicts,
		Plist:            cfg.Plist,
		Install:          split(cfg.Install),
		Tests:            split(cfg.Test),
		DownloadStrategy: cfg.DownloadStrategy,
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
