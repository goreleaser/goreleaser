package brew

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
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

// ErrNoArchivesFound happens when 0 archives are found
var ErrNoArchivesFound = errors.New("brew tap: no archives found matching criteria")

// ErrMultipleArchivesSameOS happens when the config yields multiple archives
// for linux or windows.
// TODO: improve this confusing error message
var ErrMultipleArchivesSameOS = errors.New("brew tap: one tap can handle only 1 linux and 1 macos archive")

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
	var g = semerrgroup.New(ctx.Parallelism)
	for _, brew := range ctx.Config.Brews {
		brew := brew
		g.Go(func() error {
			return doRun(ctx, brew, client)
		})
	}
	return g.Wait()
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if len(ctx.Config.Brews) == 0 {
		ctx.Config.Brews = append(ctx.Config.Brews, ctx.Config.Brew)
		if !reflect.DeepEqual(ctx.Config.Brew, config.Homebrew{}) {
			deprecate.Notice("brew")
		}
	}
	for i := range ctx.Config.Brews {
		var brew = &ctx.Config.Brews[i]
		if brew.Install == "" {
			// TODO: maybe replace this with a simplear also optimistic
			// approach of just doing `bin.install "project_name"`?
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
			brew.Install = strings.Join(installs, "\n")
			log.Warnf("optimistically guessing `brew[%d].installs`, double check", i)
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

func doRun(ctx *context.Context, brew config.Homebrew, client client.Client) error {
	if brew.GitHub.Name == "" {
		return pipe.Skip("brew section is not configured")
	}
	// If we'd use 'ctx.TokenType != context.TokenTypeGitHub' we'd have to adapt all the tests
	// For simplicity we use this check because the functionality will be implemented later for
	// all types of releases. See https://github.com/goreleaser/goreleaser/pull/1038#issuecomment-498891464
	if ctx.TokenType == context.TokenTypeGitLab {
		return pipe.Skip("brew pipe is only configured for github releases")
	}

	var filters = []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
		),
		artifact.ByFormats("zip", "tar.gz"),
		artifact.ByGoarch("amd64"),
		artifact.ByType(artifact.UploadableArchive),
	}
	if len(brew.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(brew.IDs...))
	}

	var archives = ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	content, err := buildFormula(ctx, brew, archives)
	if err != nil {
		return err
	}

	var filename = brew.Name + ".rb"
	var path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("formula", path).Info("writing")
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	if strings.TrimSpace(brew.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Draft {
		return pipe.Skip("release is marked as draft")
	}
	if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
	}

	var gpath = ghFormulaPath(brew.Folder, filename)
	log.WithField("formula", gpath).
		WithField("repo", brew.GitHub.String()).
		Info("pushing")

	var msg = fmt.Sprintf("Brew formula update for %s version %s", ctx.Config.ProjectName, ctx.Git.CurrentTag)
	return client.CreateFile(ctx, brew.CommitAuthor, brew.GitHub, []byte(content), gpath, msg)
}

func ghFormulaPath(folder, filename string) string {
	return path.Join(folder, filename)
}

func buildFormula(ctx *context.Context, brew config.Homebrew, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, brew, artifacts)
	if err != nil {
		return "", err
	}
	return doBuildFormula(ctx, data)
}

func doBuildFormula(ctx *context.Context, data templateData) (string, error) {
	t, err := template.New(data.Name).Parse(formulaTemplate)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return "", err
	}
	return tmpl.New(ctx).Apply(out.String())
}

func dataFor(ctx *context.Context, cfg config.Homebrew, artifacts []*artifact.Artifact) (templateData, error) {
	var result = templateData{
		Name:             formulaNameFor(cfg.Name),
		Desc:             cfg.Description,
		Homepage:         cfg.Homepage,
		Version:          ctx.Version,
		Caveats:          split(cfg.Caveats),
		Dependencies:     cfg.Dependencies,
		Conflicts:        cfg.Conflicts,
		Plist:            cfg.Plist,
		Install:          split(cfg.Install),
		Tests:            split(cfg.Test),
		DownloadStrategy: cfg.DownloadStrategy,
		CustomRequire:    cfg.CustomRequire,
		CustomBlock:      split(cfg.CustomBlock),
	}

	for _, artifact := range artifacts {
		sum, err := artifact.Checksum("sha256")
		if err != nil {
			return result, err
		}

		if cfg.URLTemplate == "" {
			cfg.URLTemplate = fmt.Sprintf(
				"%s/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
				ctx.Config.GitHubURLs.Download,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
			)
		}
		url, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}
		var down = downloadable{
			DownloadURL: url,
			SHA256:      sum,
		}
		if artifact.Goos == "darwin" {
			if result.MacOS.DownloadURL != "" {
				return result, ErrMultipleArchivesSameOS
			}
			result.MacOS = down
		} else if artifact.Goos == "linux" {
			switch artifact.Goarch {
			case "386", "amd64":
				if result.Linux.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.Linux = down
			case "arm":
				if result.Arm.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.Arm = down
			case "arm64":
				if result.Arm64.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.Arm64 = down
			}
		}
	}

	return result, nil
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
