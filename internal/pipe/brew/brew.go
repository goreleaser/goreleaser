package brew

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
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

const brewConfigExtra = "BrewConfig"

// ErrNoArchivesFound happens when 0 archives are found.
var ErrNoArchivesFound = errors.New("no linux/macos archives found")

// ErrMultipleArchivesSameOS happens when the config yields multiple archives
// for linux or windows.
var ErrMultipleArchivesSameOS = errors.New("one tap can handle only archive of an OS/Arch combination. Consider using ids in the brew section")

// Pipe for brew deployment.
type Pipe struct{}

func (Pipe) String() string                 { return "homebrew tap formula" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Brews) == 0 }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Brews {
		brew := &ctx.Config.Brews[i]

		if brew.CommitAuthor.Name == "" {
			brew.CommitAuthor.Name = "goreleaserbot"
		}
		if brew.CommitAuthor.Email == "" {
			brew.CommitAuthor.Email = "goreleaser@carlosbecker.com"
		}
		if brew.CommitMessageTemplate == "" {
			brew.CommitMessageTemplate = "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
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

func (Pipe) Run(ctx *context.Context) error {
	cli, err := client.New(ctx)
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

func runAll(ctx *context.Context, cli client.Client) error {
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
	for _, formula := range ctx.Artifacts.Filter(artifact.ByType(artifact.BrewTap)).List() {
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
	brew := formula.Extra[brewConfigExtra].(config.Homebrew)
	var err error
	cl, err = client.NewIfToken(ctx, cl, brew.Tap.Token)
	if err != nil {
		return err
	}

	if strings.TrimSpace(brew.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}

	if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
	}

	repo := client.RepoFromRef(brew.Tap)

	gpath := buildFormulaPath(brew.Folder, formula.Name)
	log.WithField("formula", gpath).
		WithField("repo", repo.String()).
		Info("pushing")

	msg, err := tmpl.New(ctx).Apply(brew.CommitMessageTemplate)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(formula.Path)
	if err != nil {
		return err
	}

	return cl.CreateFile(ctx, brew.CommitAuthor, repo, content, gpath, msg)
}

func doRun(ctx *context.Context, brew config.Homebrew, cl client.Client) error {
	if brew.Tap.Name == "" {
		return pipe.Skip("brew tap name is not set")
	}

	// TODO: properly cover this with tests
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
		),
		artifact.ByFormats("zip", "tar.gz"),
		artifact.Or(
			artifact.ByGoarch("amd64"),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("all"),
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

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	name, err := tmpl.New(ctx).Apply(brew.Name)
	if err != nil {
		return err
	}
	brew.Name = name

	tapOwner, err := tmpl.New(ctx).Apply(brew.Tap.Owner)
	if err != nil {
		return err
	}
	brew.Tap.Owner = tapOwner

	tapName, err := tmpl.New(ctx).Apply(brew.Tap.Name)
	if err != nil {
		return err
	}
	brew.Tap.Name = tapName

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
	path := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("formula", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write brew formula: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.BrewTap,
		Extra: map[string]interface{}{
			brewConfigExtra: brew,
		},
	})

	return nil
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
	t, err := template.
		New(data.Name).
		Funcs(template.FuncMap{
			"join": strings.Join,
		}).
		Parse(formulaTemplate)
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

func installs(cfg config.Homebrew, artifacts []*artifact.Artifact) []string {
	if cfg.Install != "" {
		return split(cfg.Install)
	}
	install := []string{}
	bins := map[string]bool{}
	for _, a := range artifacts {
		for _, bin := range a.ExtraOr("Binaries", []string{}).([]string) {
			if !bins[bin] {
				install = append(install, fmt.Sprintf("bin.install %q", bin))
			}
			bins[bin] = true
		}
	}
	log.Warnf("guessing install to be `%s`", strings.Join(install, " "))
	return install
}

func dataFor(ctx *context.Context, cfg config.Homebrew, cl client.Client, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{
		Name:          formulaNameFor(cfg.Name),
		Desc:          cfg.Description,
		Homepage:      cfg.Homepage,
		Version:       ctx.Version,
		License:       cfg.License,
		Caveats:       split(cfg.Caveats),
		Dependencies:  cfg.Dependencies,
		Conflicts:     cfg.Conflicts,
		Plist:         cfg.Plist,
		Install:       installs(cfg, artifacts),
		PostInstall:   cfg.PostInstall,
		Tests:         split(cfg.Test),
		CustomRequire: cfg.CustomRequire,
		CustomBlock:   split(cfg.CustomBlock),
	}

	counts := map[string]int{}
	for _, artifact := range artifacts {
		sum, err := artifact.Checksum("sha256")
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

		url, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}

		pkg := releasePackage{
			DownloadURL:      url,
			SHA256:           sum,
			OS:               artifact.Goos,
			Arch:             artifact.Goarch,
			DownloadStrategy: cfg.DownloadStrategy,
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

	sort.Slice(result.LinuxPackages, lessFnFor(result.LinuxPackages))
	sort.Slice(result.MacOSPackages, lessFnFor(result.MacOSPackages))
	return result, nil
}

func lessFnFor(list []releasePackage) func(i, j int) bool {
	return func(i, j int) bool { return list[i].OS > list[j].OS && list[i].Arch > list[j].Arch }
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
// The order of these replacements is important
func formulaNameFor(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "@", "AT")
	return strings.ReplaceAll(strings.Title(name), " ", "")
}
