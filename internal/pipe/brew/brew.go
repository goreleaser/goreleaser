package brew

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
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
	return "creating homebrew formula"
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
	if !reflect.DeepEqual(ctx.Config.OldBrew, config.Homebrew{}) {
		deprecate.Notice("brew")
		ctx.Config.Brews = append(ctx.Config.Brews, ctx.Config.OldBrew)
	}
	for i, brew := range ctx.Config.Brews {
		if brew.Install == "" {
			var installs []string
			for _, build := range ctx.Config.Builds {
				if !isBrewBuild(brew, build) {
					continue
				}
				installs = append(
					installs,
					fmt.Sprintf(`bin.install "%s"`, build.Binary),
				)
			}
			brew.Install = strings.Join(installs, "\n")
		}

		if brew.CommitAuthor.Name == "" {
			brew.CommitAuthor.Name = "goreleaserbot"
		}
		if brew.CommitAuthor.Email == "" {
			brew.CommitAuthor.Email = "goreleaser@carlosbecker.com"
		}
		if brew.Name == "" {
			brew.Name = ctx.Config.ProjectName
		}
		ctx.Config.Brews[i] = brew
	}
	return nil
}

func isBrewBuild(brew config.Homebrew, build config.Build) bool {
	for _, ignore := range build.Ignore {
		if ignore.Goos == "darwin" && ignore.Goarch == "amd64" {
			return false
		}
	}
	if len(brew.Binaries) > 0 && !contains(brew.Binaries, build.Binary) {
		return false
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
	if len(ctx.Config.Brews) == 0 {
		return pipe.Skip("brews section is not configured")
	}
	var g = semerrgroup.New(ctx.Parallelism)
	for _, brew := range ctx.Config.Brews {
		brew := brew
		g.Go(func() error {
			return doRunForBrew(ctx, client, brew)
		})
	}
	return g.Wait()
}

func doRunForBrew(ctx *context.Context, client client.Client, brew config.Homebrew) error {
	if brew.SkipUpload {
		return pipe.Skip("brew.skip_upload is set")
	}
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Draft {
		return pipe.Skip("release is marked as draft")
	}

	var filter = artifact.And(
		artifact.ByGoos("darwin"),
		artifact.ByGoarch("amd64"),
		artifact.ByGoarm(""),
		artifact.ByType(artifact.UploadableArchive),
	)

	if len(brew.Binaries) > 0 {
		filter = artifact.And(filter, artifact.ByBinaryName(brew.Binaries...))
	}

	var archives = ctx.Artifacts.Filter(filter).List()

	if len(archives) == 0 {
		return ErrNoDarwin64Build
	}
	if len(archives) > 1 {
		return ErrTooManyDarwin64Builds
	}

	content, err := buildFormula(ctx, brew, archives[0])
	if err != nil {
		return err
	}

	var filename = brew.Name + ".rb"
	var path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("formula", path).Info("writing")
	if err := ioutil.WriteFile(path, content.Bytes(), 0644); err != nil {
		return err
	}

	path = filepath.Join(brew.Folder, filename)
	log.WithField("formula", path).
		WithField("repo", brew.GitHub.String()).
		Info("pushing")

	var msg = fmt.Sprintf("Brew formula update for %s version %s", ctx.Config.ProjectName, ctx.Git.CurrentTag)
	return client.CreateFile(ctx, brew.CommitAuthor, brew.GitHub, content, path, msg)
}

func buildFormula(ctx *context.Context, brew config.Homebrew, artifact artifact.Artifact) (bytes.Buffer, error) {
	data, err := dataFor(ctx, brew, artifact)
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

func dataFor(ctx *context.Context, brew config.Homebrew, artifact artifact.Artifact) (result templateData, err error) {
	sum, err := artifact.Checksum()
	if err != nil {
		return
	}

	if brew.URLTemplate == "" {
		brew.URLTemplate = fmt.Sprintf("%s/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
			ctx.Config.GitHubURLs.Download,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name)
	}
	url, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).Apply(brew.URLTemplate)
	if err != nil {
		return
	}

	return templateData{
		Name:             formulaNameFor(brew.Name),
		DownloadURL:      url,
		Desc:             brew.Description,
		Homepage:         brew.Homepage,
		Version:          ctx.Version,
		Caveats:          split(brew.Caveats),
		SHA256:           sum,
		Dependencies:     brew.Dependencies,
		Conflicts:        brew.Conflicts,
		Plist:            brew.Plist,
		Install:          split(brew.Install),
		Tests:            split(brew.Test),
		DownloadStrategy: brew.DownloadStrategy,
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
