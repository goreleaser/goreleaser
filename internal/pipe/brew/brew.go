package brew

import (
	"bufio"
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
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoArchivesFound happens when 0 archives are found.
var ErrNoArchivesFound = errors.New("no linux/macos archives found")

// ErrMultipleArchivesSameOS happens when the config yields multiple archives
// for linux or windows.
var ErrMultipleArchivesSameOS = errors.New("one tap can handle only archive of an OS/Arch combination. Consider using ids in the brew section")

// ErrTokenTypeNotImplementedForBrew indicates that a new token type was not implemented for this pipe.
type ErrTokenTypeNotImplementedForBrew struct {
	TokenType context.TokenType
}

func (e ErrTokenTypeNotImplementedForBrew) Error() string {
	if e.TokenType != "" {
		return fmt.Sprintf("token type %q not implemented for brew pipe", e.TokenType)
	}
	return "token type not implemented for brew pipe"
}

// Pipe for brew deployment.
type Pipe struct{}

func (Pipe) String() string {
	return "homebrew tap formula"
}

// Publish brew formula.
func (Pipe) Publish(ctx *context.Context) error {
	// we keep GitHub as default for now, in line with releases
	if string(ctx.TokenType) == "" {
		ctx.TokenType = context.TokenTypeGitHub
	}

	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
}

func publishAll(ctx *context.Context, cli client.Client) error {
	// even if one of them skips, we run them all, and then show return the skips all at once.
	// this is needed so we actually create the `dist/foo.rb` file, which is useful for debugging.
	var skips = pipe.SkipMemento{}
	for _, brew := range ctx.Config.Brews {
		var err = doRun(ctx, brew, cli)
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

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Brews {
		var brew = &ctx.Config.Brews[i]

		if brew.Install == "" {
			// TODO: maybe replace this with a simpler also optimistic
			// approach of just doing `bin.install "project_name"`?
			var installs []string
			for _, build := range ctx.Config.Builds {
				if !isBrewBuild(build) {
					continue
				}
				install := fmt.Sprintf(`bin.install "%s"`, build.Binary)
				// Do not add duplicate "bin.install" statements when binary names overlap.
				var found bool
				for _, instruction := range installs {
					if instruction == install {
						found = true
						break
					}
				}
				if !found {
					installs = append(installs, install)
				}
			}
			brew.Install = strings.Join(installs, "\n")
			log.Warnf("optimistically guessing `brew[%d].install`, double check", i)
		}

		//nolint: staticcheck
		if brew.GitHub.String() != "" {
			deprecate.Notice(ctx, "brews.github")
			brew.Tap.Owner = brew.GitHub.Owner
			brew.Tap.Name = brew.GitHub.Name
		}
		if brew.GitLab.String() != "" {
			deprecate.Notice(ctx, "brews.gitlab")
			brew.Tap.Owner = brew.GitLab.Owner
			brew.Tap.Name = brew.GitLab.Name
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
		if brew.Goarm == "" {
			brew.Goarm = "6"
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

func doRun(ctx *context.Context, brew config.Homebrew, cl client.Client) error {
	if brew.Tap.Name == "" {
		return pipe.Skip("brew section is not configured")
	}

	if brew.Tap.Token != "" {
		token, err := tmpl.New(ctx).ApplySingleEnvOnly(brew.Tap.Token)
		if err != nil {
			return err
		}
		log.Debug("using custom token to publish homebrew formula")
		c, err := client.NewWithToken(ctx, token)
		if err != nil {
			return err
		}
		cl = c
	}

	// TODO: properly cover this with tests
	var filters = []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
		),
		artifact.ByFormats("zip", "tar.gz"),
		artifact.Or(
			artifact.ByGoarch("amd64"),
			artifact.ByGoarch("arm64"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.ByGoarm(brew.Goarm),
			),
		),
		artifact.ByType(artifact.UploadableArchive),
	}
	if len(brew.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(brew.IDs...))
	}

	var archives = ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	name, err := tmpl.New(ctx).Apply(brew.Name)
	if err != nil {
		return err
	}
	brew.Name = name

	content, err := buildFormula(ctx, brew, cl, archives)
	if err != nil {
		return err
	}

	var filename = brew.Name + ".rb"
	var path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("formula", path).Info("writing")
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write brew formula: %w", err)
	}

	if strings.TrimSpace(brew.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
	}

	repo := client.RepoFromRef(brew.Tap)

	var gpath = buildFormulaPath(brew.Folder, filename)
	log.WithField("formula", gpath).
		WithField("repo", repo.String()).
		Info("pushing")

	var msg = fmt.Sprintf("Brew formula update for %s version %s", ctx.Config.ProjectName, ctx.Git.CurrentTag)
	return cl.CreateFile(ctx, brew.CommitAuthor, repo, []byte(content), gpath, msg)
}

func buildFormulaPath(folder, filename string) string {
	return path.Join(folder, filename)
}

func buildFormula(ctx *context.Context, brew config.Homebrew, client client.Client, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, brew, client, artifacts)
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

	content, err := tmpl.New(ctx).Apply(out.String())
	if err != nil {
		return "", err
	}
	out.Reset()

	// Sanitize the template output and get rid of trailing whitespace.
	var (
		r = strings.NewReader(content)
		s = bufio.NewScanner(r)
	)
	for s.Scan() {
		l := strings.TrimRight(s.Text(), " ")
		_, _ = out.WriteString(l)
		_ = out.WriteByte('\n')
	}
	if err := s.Err(); err != nil {
		return "", err
	}

	return out.String(), nil
}

func dataFor(ctx *context.Context, cfg config.Homebrew, cl client.Client, artifacts []*artifact.Artifact) (templateData, error) {
	var result = templateData{
		Name:             formulaNameFor(cfg.Name),
		Desc:             cfg.Description,
		Homepage:         cfg.Homepage,
		Version:          ctx.Version,
		License:          cfg.License,
		Caveats:          split(cfg.Caveats),
		Dependencies:     cfg.Dependencies,
		Conflicts:        cfg.Conflicts,
		Plist:            cfg.Plist,
		Install:          split(cfg.Install),
		PostInstall:      cfg.PostInstall,
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
			url, err := cl.ReleaseURLTemplate(ctx)
			if err != nil {
				if client.IsNotImplementedErr(err) {
					return result, ErrTokenTypeNotImplementedForBrew{ctx.TokenType}
				}
				return result, err
			}
			cfg.URLTemplate = url
		}
		url, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}
		var down = downloadable{
			DownloadURL: url,
			SHA256:      sum,
		}
		// TODO: refactor
		if artifact.Goos == "darwin" { // nolint: nestif
			switch artifact.Goarch {
			case "amd64":
				if result.MacOSAmd64.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.MacOSAmd64 = down
			case "arm64":
				if result.MacOSArm64.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.MacOSArm64 = down
			}
		} else if artifact.Goos == "linux" {
			switch artifact.Goarch {
			case "amd64":
				if result.LinuxAmd64.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.LinuxAmd64 = down
			case "arm":
				if result.LinuxArm.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.LinuxArm = down
			case "arm64":
				if result.LinuxArm64.DownloadURL != "" {
					return result, ErrMultipleArchivesSameOS
				}
				result.LinuxArm64 = down
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
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "@", "AT")
	return strings.ReplaceAll(strings.Title(name), " ", "")
}
