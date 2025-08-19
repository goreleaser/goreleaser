package brew

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/commitauthor"
	"github.com/goreleaser/goreleaser/v2/internal/experimental"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const brewConfigExtra = "BrewConfig"

// ErrMultipleArchivesSameOS happens when the config yields multiple archives
// for linux or windows.
var ErrMultipleArchivesSameOS = errors.New("one tap can handle only one archive of an OS/Arch combination. Consider using ids in the brew section")

// ErrNoArchivesFound happens when 0 archives are found.
type ErrNoArchivesFound struct {
	goarm   string
	goamd64 string
	ids     []string
}

func (e ErrNoArchivesFound) Error() string {
	return fmt.Sprintf("no linux/macos archives found matching goos=[darwin linux] goarch=[amd64 arm64 arm] goamd64=%s goarm=%s ids=%v", e.goamd64, e.goarm, e.ids)
}

// Pipe for brew deployment.
//
// Deprecated: in favor of [cask.Pipe].
type Pipe struct{}

func (Pipe) String() string        { return "homebrew formula" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Homebrew) || len(ctx.Config.Brews) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	var warnOnce sync.Once
	for i := range ctx.Config.Brews {
		// TODO: add this back at some point:
		// deprecate.Notice(ctx, "brews")
		warnOnce.Do(func() {
			log.Warn(
				logext.Keyword("brews") +
					logext.Warning(" is being phased out in favor of ") +
					logext.Keyword("homebrew_casks") +
					logext.Warning(", check ") +
					logext.URL("https://goreleaser.com/deprecations#brews") +
					logext.Warning(" for more info"),
			)
		})

		brew := &ctx.Config.Brews[i]

		brew.CommitAuthor = commitauthor.Default(brew.CommitAuthor)

		if brew.CommitMessageTemplate == "" {
			brew.CommitMessageTemplate = "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if brew.Name == "" {
			brew.Name = ctx.Config.ProjectName
		}
		if brew.Goarm == "" {
			brew.Goarm = experimental.DefaultGOARM()
		}
		if brew.Goamd64 == "" {
			brew.Goamd64 = "v1"
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

// Publish brew formula.
func (Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
}

func runAll(ctx *context.Context, cli client.ReleaseURLTemplater) error {
	for _, brew := range ctx.Config.Brews {
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
	for _, formula := range ctx.Artifacts.Filter(artifact.ByType(artifact.BrewFormula)).List() {
		err := doPublish(ctx, formula, cli)
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

func doPublish(ctx *context.Context, formula *artifact.Artifact, cl client.Client) error {
	brew := artifact.MustExtra[config.Homebrew](*formula, brewConfigExtra)
	if strings.TrimSpace(brew.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}

	if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
	}

	repo := client.RepoFromRef(brew.Repository)

	gpath := buildFormulaPath(brew.Directory, formula.Name)

	msg, err := tmpl.New(ctx).Apply(brew.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, brew.CommitAuthor)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(formula.Path)
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
		log.Debug("brews.pull_request disabled")
		return nil
	}

	log.Info("brews.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return errors.New("client does not support pull requests")
	}

	return pcl.OpenPullRequest(ctx, base, repo, msg, brew.Repository.PullRequest.Draft)
}

func doRun(ctx *context.Context, brew config.Homebrew, cl client.ReleaseURLTemplater) error {
	if brew.Repository.Name == "" {
		return pipe.Skip("brew.repository.name is not set")
	}

	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
		),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(brew.Goamd64),
			),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("all"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.ByGoarm(brew.Goarm),
			),
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
			goamd64: brew.Goamd64,
			goarm:   brew.Goarm,
			ids:     brew.IDs,
		}
	}

	name, err := tmpl.New(ctx).Apply(brew.Name)
	if err != nil {
		return err
	}
	brew.Name = name

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, brew.Repository)
	if err != nil {
		return err
	}
	brew.Repository = ref

	skipUpload, err := tmpl.New(ctx).Apply(brew.SkipUpload)
	if err != nil {
		return err
	}
	brew.SkipUpload = skipUpload

	content, err := buildFormula(ctx, brew, cl, archives)
	if err != nil {
		return err
	}

	filename := brew.Name + ".rb"
	path := filepath.Join(ctx.Config.Dist, "homebrew", brew.Directory, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	log.WithField("formula", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write brew formula: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.BrewFormula,
		Extra: map[string]any{
			brewConfigExtra: brew,
		},
	})

	return nil
}

func buildFormulaPath(folder, filename string) string {
	return path.Join(folder, filename)
}

func buildFormula(ctx *context.Context, brew config.Homebrew, client client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, brew, client, artifacts)
	if err != nil {
		return "", err
	}
	return doBuildFormula(ctx, data)
}

func doBuildFormula(ctx *context.Context, data templateData) (string, error) {
	t := template.New("formula.rb")
	var err error
	t, err = t.Funcs(map[string]any{
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
		"join": func(in []string) string {
			items := make([]string, 0, len(in))
			for _, i := range in {
				items = append(items, fmt.Sprintf(`"%s"`, i))
			}
			return strings.Join(items, ",\n")
		},
	}).ParseFS(formulaTemplate, "templates/*.rb")
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

func installs(ctx *context.Context, cfg config.Homebrew, art *artifact.Artifact) ([]string, error) {
	tpl := tmpl.New(ctx).WithArtifact(art)

	extraInstall, err := tpl.Apply(cfg.ExtraInstall)
	if err != nil {
		return nil, err
	}

	install, err := tpl.Apply(cfg.Install)
	if err != nil {
		return nil, err
	}
	if install != "" {
		return append(split(install), split(extraInstall)...), nil
	}

	installMap := map[string]bool{}
	switch art.Type {
	case artifact.UploadableBinary:
		name := art.Name
		bin := artifact.MustExtra[string](*art, artifact.ExtraBinary)
		installMap[fmt.Sprintf("bin.install %q => %q", name, bin)] = true
	case artifact.UploadableArchive:
		for _, bin := range artifact.MustExtra[[]string](*art, artifact.ExtraBinaries) {
			installMap[fmt.Sprintf("bin.install %q", bin)] = true
		}
	}

	result := slices.Sorted(maps.Keys(installMap))
	log.WithField("install", result).Info("guessing install")

	return append(result, split(extraInstall)...), nil
}

func dataFor(ctx *context.Context, cfg config.Homebrew, cl client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (templateData, error) {
	slices.SortFunc(cfg.Dependencies, func(a, b config.HomebrewDependency) int {
		return cmp.Compare(a.Name, b.Name)
	})
	result := templateData{
		Name:          formulaNameFor(cfg.Name),
		Desc:          cfg.Description,
		Homepage:      cfg.Homepage,
		Version:       ctx.Version,
		License:       cfg.License,
		Caveats:       split(cfg.Caveats),
		Dependencies:  cfg.Dependencies,
		Conflicts:     cfg.Conflicts,
		Service:       split(cfg.Service),
		PostInstall:   split(cfg.PostInstall),
		Tests:         split(cfg.Test),
		CustomRequire: cfg.CustomRequire,
		CustomBlock:   split(cfg.CustomBlock),
	}

	counts := map[string]int{}
	for _, art := range artifacts {
		sum, err := art.Checksum("sha256")
		if err != nil {
			return result, err
		}

		if cfg.URLTemplate == "" {
			url, err := cl.ReleaseURLTemplate(ctx)
			if err != nil {
				return result, err
			}
			cfg.URLTemplate = url
		}

		url, err := tmpl.New(ctx).WithArtifact(art).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}

		install, err := installs(ctx, cfg, art)
		if err != nil {
			return result, err
		}

		pkg := releasePackage{
			DownloadURL:      url,
			SHA256:           sum,
			OS:               art.Goos,
			Arch:             art.Goarch,
			DownloadStrategy: cfg.DownloadStrategy,
			Headers:          cfg.URLHeaders,
			Install:          install,
		}

		counts[pkg.OS+pkg.Arch]++

		switch pkg.OS {
		case "darwin":
			result.MacOSPackages = append(result.MacOSPackages, pkg)
		case "linux":
			result.LinuxPackages = append(result.LinuxPackages, pkg)
		}
	}

	for _, v := range counts {
		if v > 1 {
			return result, ErrMultipleArchivesSameOS
		}
	}

	if len(result.MacOSPackages) == 1 && result.MacOSPackages[0].Arch == "amd64" {
		result.HasOnlyAmd64MacOsPkg = true
	}

	slices.SortStableFunc(result.LinuxPackages, compareByArch)
	slices.SortStableFunc(result.MacOSPackages, compareByArch)
	return result, nil
}

func compareByArch(a, b releasePackage) int {
	return cmp.Compare(a.Arch, b.Arch)
}

func split(s string) []string {
	strings := strings.Split(strings.TrimSpace(s), "\n")
	if len(strings) == 1 && strings[0] == "" {
		return []string{}
	}
	return strings
}

// formulaNameFor transforms the formula name into a form
// that more resembles a valid Ruby class name
// e.g. foo_bar@v6.0.0-rc is turned into FooBarATv6_0_0RC
//
// This function must match the behavior of Homebrew's Formulary.class_s function:
//
//	<https://github.com/Homebrew/brew/blob/587949bd8417c486795be04194f9e9baeaa9f5a7/Library/Homebrew/formulary.rb#L522-L528>
func formulaNameFor(name string) string {
	if len(name) == 0 {
		return name
	}

	var output strings.Builder
	name = strings.ToLower(name)

	// Capitalize the first character
	output.WriteByte(strings.ToUpper(name[:1])[0])

	// Traverse the rest of the string
	for i := 1; i < len(name); i++ {
		c := name[i]

		switch c {
		case '-', '_', '.', ' ':
			// Capitalize the next character after a symbol
			if i+1 < len(name) {
				output.WriteByte(strings.ToUpper(name[i+1 : i+2])[0])
				i++ // Skip the next character as it's already processed
			}
		case '+':
			// Replace '+' with 'x'
			output.WriteByte('x')
		case '@':
			// Replace occurrences of (.)@(\d) with \1AT\2
			if i+1 < len(name) && isDigit(name[i+1]) {
				output.WriteString("AT")
				output.WriteByte(name[i+1])
				i++ // Skip the next character as it's already processed
			} else {
				output.WriteByte(c)
			}
		default:
			output.WriteByte(c)
		}
	}

	return output.String()
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
