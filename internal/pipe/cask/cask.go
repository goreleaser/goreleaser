package cask

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/commitauthor"
	"github.com/goreleaser/goreleaser/v2/internal/deprecate"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const brewConfigExtra = "BrewCaskConfig"

// ErrMultipleArchivesSameOS happens when the config yields multiple archives
// for linux or windows.
var ErrMultipleArchivesSameOS = errors.New("one tap can handle only one archive of an OS/Arch combination. Consider using ids in the homebrew_casks section")

// ErrNoArchivesFound happens when 0 archives are found.
type ErrNoArchivesFound struct {
	ids []string
}

func (e ErrNoArchivesFound) Error() string {
	return fmt.Sprintf("no linux/macos archives found matching goos=[darwin linux] goarch=[amd64 arm64] ids=%v", e.ids)
}

// Pipe for brew deployment.
type Pipe struct{}

func (Pipe) String() string        { return "homebrew cask" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Homebrew) || len(ctx.Config.Casks) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Casks {
		brew := &ctx.Config.Casks[i]

		brew.CommitAuthor = commitauthor.Default(brew.CommitAuthor)

		if brew.CommitMessageTemplate == "" {
			brew.CommitMessageTemplate = "Brew cask update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if brew.Name == "" {
			brew.Name = ctx.Config.ProjectName
		}
		if brew.Directory == "" {
			brew.Directory = "Casks"
		}
		if brew.Directory != "Casks" {
			log.Warnf("%q might not work properly for your end users, for reference, the default is \"Casks\"", brew.Directory)
		}
		if brew.Binary == "" {
			brew.Binary = brew.Name
		}
		if brew.Manpage != "" {
			deprecate.Notice(ctx, "homebrew_casks.manpage")
			brew.Manpages = append(brew.Manpages, brew.Manpage)
		}
	}

	return nil
}

func (Pipe) Run(ctx *context.Context) error {
	cli, err := client.NewReleaseClient(ctx)
	if err != nil {
		return err
	}

	return runAll(ctx, cli)
}

// Publish brew cask.
func (Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
}

func runAll(ctx *context.Context, cli client.ReleaseURLTemplater) error {
	for _, brew := range ctx.Config.Casks {
		err := doRun(ctx, brew, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func publishAll(ctx *context.Context, cli client.Client) error {
	// even if one of them skips, we run them all, and then show return the skips all at once.
	// this is needed so we actually create the `dist/foo.rb` file, which is useful for debugging.
	skips := pipe.SkipMemento{}
	for _, cask := range ctx.Artifacts.Filter(artifact.ByType(artifact.BrewCask)).List() {
		err := doPublish(ctx, cask, cli)
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

func doPublish(ctx *context.Context, cask *artifact.Artifact, cl client.Client) error {
	brew := artifact.MustExtra[config.HomebrewCask](*cask, brewConfigExtra)
	if strings.TrimSpace(brew.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}

	if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
	}

	repo := client.RepoFromRef(brew.Repository)

	gpath := buildCaskPath(brew.Directory, cask.Name)

	msg, err := tmpl.New(ctx).Apply(brew.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, brew.CommitAuthor)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(cask.Path)
	if err != nil {
		return err
	}

	if brew.Repository.Git.URL != "" {
		return client.NewGitUploadClient(repo.Branch).
			CreateFile(ctx, author, repo, content, gpath, msg)
	}

	cl, err = client.NewIfToken(ctx, cl, brew.Repository.Token)
	if err != nil {
		return err
	}

	base := client.Repo{
		Name:   brew.Repository.PullRequest.Base.Name,
		Owner:  brew.Repository.PullRequest.Base.Owner,
		Branch: brew.Repository.PullRequest.Base.Branch,
	}

	// try to sync branch
	fscli, ok := cl.(client.ForkSyncer)
	if ok && brew.Repository.PullRequest.Enabled {
		if err := fscli.SyncFork(ctx, repo, base); err != nil {
			log.WithError(err).Warn("could not sync fork")
		}
	}

	if err := cl.CreateFile(ctx, author, repo, content, gpath, msg); err != nil {
		return err
	}

	if !brew.Repository.PullRequest.Enabled {
		log.Debug("homebrew_casks.pull_request disabled")
		return nil
	}

	log.Info("homebrew_casks.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return errors.New("client does not support pull requests")
	}

	return pcl.OpenPullRequest(ctx, base, repo, msg, brew.Repository.PullRequest.Draft)
}

func doRun(ctx *context.Context, brew config.HomebrewCask, cl client.ReleaseURLTemplater) error {
	if brew.Repository.Name == "" {
		return pipe.Skip("homebrew_casks.repository.name is not set")
	}

	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
		),
		artifact.Or(
			artifact.ByGoarch("amd64"),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("all"),
		),
		artifact.Or(
			artifact.And(
				artifact.Not(artifact.ByFormats("gz")),
				artifact.ByType(artifact.UploadableArchive),
			),
			artifact.ByType(artifact.UploadableBinary),
		),
		artifact.OnlyReplacingUnibins,
	}
	if len(brew.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(brew.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound{
			ids: brew.IDs,
		}
	}

	if err := tmpl.New(ctx).ApplyAll(
		&brew.Name,
		&brew.SkipUpload,
		&brew.Binary,
		&brew.Completions.Bash,
		&brew.Completions.Zsh,
		&brew.Completions.Fish,
		&brew.SkipUpload,
	); err != nil {
		return err
	}

	if err := tmpl.New(ctx).ApplySlice(&brew.Manpages); err != nil {
		return err
	}

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, brew.Repository)
	if err != nil {
		return err
	}
	brew.Repository = ref

	content, err := buildCask(ctx, brew, cl, archives)
	if err != nil {
		return err
	}

	filename := brew.Name + ".rb"
	path := filepath.Join(ctx.Config.Dist, "homebrew", brew.Directory, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	log.WithField("cask", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write homebrew cask: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.BrewCask,
		Extra: map[string]any{
			brewConfigExtra: brew,
		},
	})

	return nil
}

func buildCaskPath(folder, filename string) string {
	return path.Join(folder, filename)
}

func buildCask(ctx *context.Context, brew config.HomebrewCask, client client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, brew, client, artifacts)
	if err != nil {
		return "", err
	}
	return doBuildCask(ctx, data)
}

func doBuildCask(ctx *context.Context, data templateData) (string, error) {
	t := template.New("cask.rb")
	var err error
	t, err = t.Funcs(map[string]any{
		"split": split,
		"include": func(name string, data any) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := t.ExecuteTemplate(buf, name, data); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.ReplaceAll(v, "\n", "\n"+pad)
		},
		"uninstall": uninstallString,
		"zap":       zapString,
		"conflicts": conflictsString,
		"depends":   dependsString,
	}).ParseFS(templates, "templates/*.rb")
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
		r  = strings.NewReader(content)
		s  = bufio.NewScanner(r)
		el = false
	)
	for s.Scan() {
		l := strings.TrimRight(s.Text(), " ")
		if strings.TrimSpace(l) == "" {
			if !el {
				_ = out.WriteByte('\n')
				el = true
			}
		} else {
			_, _ = out.WriteString(l)
			_ = out.WriteByte('\n')
			el = false
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}

	return out.String(), nil
}

func dataFor(ctx *context.Context, cfg config.HomebrewCask, cl client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (templateData, error) {
	slices.SortFunc(cfg.Dependencies, func(a, b config.HomebrewCaskDependency) int {
		return cmp.Compare(cmp.Or(a.Cask, a.Formula), cmp.Or(b.Cask, b.Formula))
	})
	result := templateData{
		HomebrewCask: cfg,
		Name:         caskNameFor(cfg.Name),
		Version:      ctx.Version,
	}

	platformCounts := map[string]int{}
	formatCounts := map[artifact.Type]int{}
	for _, art := range artifacts {
		sum, err := art.Checksum("sha256")
		if err != nil {
			return result, err
		}

		if cfg.URL.Template == "" {
			url, err := cl.ReleaseURLTemplate(ctx)
			if err != nil {
				return result, err
			}
			cfg.URL.Template = url
		}

		url := downloadURL{
			Verified:  cfg.URL.Verified,
			Using:     cfg.URL.Using,
			Cookies:   cfg.URL.Cookies,
			Referer:   cfg.URL.Referer,
			Headers:   cfg.URL.Headers,
			UserAgent: cfg.URL.UserAgent,
			Data:      cfg.URL.Data,
		}

		url.Download, err = tmpl.New(ctx).WithArtifact(art).Apply(cfg.URL.Template)
		if err != nil {
			return result, err
		}

		pkg := releasePackage{
			URL:    url,
			SHA256: sum,
			OS:     art.Goos,
			Arch:   art.Goarch,
		}

		if art.Type == artifact.UploadableBinary {
			pkg.Binary = artifact.MustExtra[string](*art, artifact.ExtraBinary)
			pkg.Name = art.Name
		}

		formatCounts[art.Type]++
		platformCounts[pkg.OS+pkg.Arch]++

		switch pkg.OS {
		case "darwin":
			result.MacOSPackages = append(result.MacOSPackages, pkg)
		case "linux":
			result.LinuxPackages = append(result.LinuxPackages, pkg)
		}
	}

	for _, v := range platformCounts {
		if v > 1 {
			return result, ErrMultipleArchivesSameOS
		}
	}

	result.HasOnlyAmd64MacOsPkg = len(result.MacOSPackages) == 1 && result.MacOSPackages[0].Arch == "amd64"
	result.HasOnlyBinaryPkgs = len(formatCounts) == 1 && formatCounts[artifact.UploadableBinary] > 0

	slices.SortStableFunc(result.LinuxPackages, compareByArch)
	slices.SortStableFunc(result.MacOSPackages, compareByArch)
	return result, nil
}

func compareByArch(a, b releasePackage) int {
	return cmp.Compare(a.Arch, b.Arch)
}

func caskNameFor(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
