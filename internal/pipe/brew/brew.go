package brew

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoDarwin64Build when there is no build for darwin_amd64
var ErrNoDarwin64Build = errors.New("brew tap requires one darwin amd64 build")

// ErrTooManyDarwin64Builds when there are too many builds for darwin_amd64
var ErrTooManyDarwin64Builds = errors.New("brew tap requires at most one darwin amd64 build")

// Pipe for brew deployment
type Pipe struct{}

func (Pipe) String() string {
	return "homebrew tap formula"
}

// Publish brew formula
func (Pipe) Publish(ctx *context.Context) error {
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
	if ctx.Config.Brew.Name == "" {
		ctx.Config.Brew.Name = ctx.Config.ProjectName
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
		return pipe.Skip("brew section is not configured")
	}
	if getFormat(ctx) == "binary" {
		return pipe.Skip("archive format is binary")
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

	content, err := buildFormula(ctx, archives[0])
	if err != nil {
		return err
	}

	var filename = ctx.Config.Brew.Name + ".rb"
	var path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("formula", path).Info("writing")
	if err := ioutil.WriteFile(path, content.Bytes(), 0644); err != nil {
		return err
	}

	if ctx.Config.Brew.SkipUpload {
		return pipe.Skip("brew.skip_upload is set")
	}
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Draft {
		return pipe.Skip("release is marked as draft")
	}

	var gpath = ghFormulaPath(ctx.Config.Brew.Folder, filename)
	log.WithField("formula", gpath).
		WithField("repo", ctx.Config.Brew.GitHub.String()).
		Info("pushing")

	var msg = fmt.Sprintf("Brew formula update for %s version %s", ctx.Config.ProjectName, ctx.Git.CurrentTag)
	return client.CreateFile(ctx, ctx.Config.Brew.CommitAuthor, ctx.Config.Brew.GitHub, content, gpath, msg)
}

func ghFormulaPath(folder, filename string) string {
	return path.Join(folder, filename)
}

func getFormat(ctx *context.Context) string {
	for _, override := range ctx.Config.Archive.FormatOverrides {
		if strings.HasPrefix("darwin", override.Goos) {
			return override.Format
		}
	}
	return ctx.Config.Archive.Format
}

func buildFormula(ctx *context.Context, artifact artifact.Artifact) (bytes.Buffer, error) {
	data, err := dataFor(ctx, artifact)
	if err != nil {
		return bytes.Buffer{}, err
	}
	return doBuildFormula(data)
}

func doBuildFormula(data templateData) (out bytes.Buffer, err error) {
	t, err := template.New(data.Name).Parse(formulaTemplate)
	if err != nil {
		return out, err
	}
	err = t.Execute(&out, data)
	return
}

func dataFor(ctx *context.Context, artifact artifact.Artifact) (result templateData, err error) {
	sum, err := artifact.Checksum()
	if err != nil {
		return
	}
	var cfg = ctx.Config.Brew

	if ctx.Config.Brew.URLTemplate == "" {
		ctx.Config.Brew.URLTemplate = fmt.Sprintf("%s/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
			ctx.Config.GitHubURLs.Download,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name)
	}
	url, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).Apply(ctx.Config.Brew.URLTemplate)
	if err != nil {
		return
	}

	return templateData{
		Name:             formulaNameFor(ctx.Config.Brew.Name),
		DownloadURL:      url,
		Desc:             cfg.Description,
		Homepage:         cfg.Homepage,
		Version:          ctx.Version,
		Caveats:          split(cfg.Caveats),
		SHA256:           sum,
		Dependencies:     cfg.Dependencies,
		Conflicts:        cfg.Conflicts,
		Plist:            cfg.Plist,
		Install:          split(cfg.Install),
		Tests:            split(cfg.Test),
		DownloadStrategy: cfg.DownloadStrategy,
		CustomRequire:    cfg.CustomRequire,
	}, nil
}

func split(s string) []string {
	strings := strings.Split(strings.TrimSpace(s), "\n")
	if len(strings) == 1 && strings[0] == "" {
		return []string{}
	}
	return strings
}

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
