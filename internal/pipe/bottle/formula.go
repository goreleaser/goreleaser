package bottle

import (
	"bufio"
	"bytes"
	"cmp"
	"embed"
	"errors"
	"fmt"
	"maps"
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
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/summary"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

//go:embed templates
var formulaTemplate embed.FS

// ErrNoSourceArchive happens when the source archive is not enabled: the
// formula's url must point at it so --build-from-source works.
var ErrNoSourceArchive = errors.New("the source archive is required to build from source: set `source.enabled: true`")

// Formula generates and publishes the formula that pours the bottles built by
// [Archive].
type Formula struct{}

func (Formula) String() string        { return "homebrew bottle formula" }
func (Formula) ContinueOnError() bool { return true }
func (Formula) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Homebrew) || len(ctx.Config.HomebrewBottles) == 0
}

func (Formula) Run(ctx *context.Context) error {
	cli, err := client.NewReleaseClient(ctx)
	if err != nil {
		return err
	}
	for _, cfg := range ctx.Config.HomebrewBottles {
		if err := doRun(ctx, cfg, cli); err != nil {
			return err
		}
	}
	return nil
}

func (Formula) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	skips := pipe.SkipMemento{}
	for _, formula := range ctx.Artifacts.Filter(artifact.ByType(artifact.HomebrewBottleFormula)).List() {
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

type bottleData struct {
	Tag    string
	SHA256 string
}

type templateData struct {
	Class        string
	Desc         string
	Homepage     string
	Version      string
	License      string
	SourceURL    string
	SourceSHA256 string
	RootURL      string
	Bottles      []bottleData
	Dependencies []config.HomebrewDependency
	Conflicts    []string
	CustomBlock  []string
	Install      []string
	PostInstall  []string
	Caveats      []string
	Service      []string
	Tests        []string
}

func doRun(ctx *context.Context, cfg config.HomebrewBottle, cl client.ReleaseURLTemplater) error {
	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, cfg.Repository)
	if err != nil {
		return err
	}
	cfg.Repository = ref
	if cfg.Repository.Name == "" && cfg.Repository.Git.URL == "" {
		return pipe.Skip("homebrew_bottles.repository.name is not set")
	}
	if cfg.SkipUpload, err = tmpl.New(ctx).Apply(cfg.SkipUpload); err != nil {
		return err
	}

	content, err := buildFormula(ctx, cfg, cl)
	if err != nil {
		return err
	}

	filename := cfg.Name + ".rb"
	dst := filepath.Join(ctx.Config.Dist, "homebrew_bottles", cfg.Directory, filename)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	log.WithField("formula", dst).Info("writing")
	if err := os.WriteFile(dst, []byte(content), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write homebrew formula: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:  filename,
		Path:  dst,
		Type:  artifact.HomebrewBottleFormula,
		Extra: map[string]any{extraConfig: cfg},
	})
	return nil
}

func buildFormula(ctx *context.Context, cfg config.HomebrewBottle, cl client.ReleaseURLTemplater) (string, error) {
	data, err := dataFor(ctx, cfg, cl)
	if err != nil {
		return "", err
	}
	return render(data)
}

func dataFor(ctx *context.Context, cfg config.HomebrewBottle, cl client.ReleaseURLTemplater) (templateData, error) {
	var data templateData
	urlTemplate := cfg.URLTemplate
	if urlTemplate == "" {
		u, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return data, err
		}
		urlTemplate = u
	}

	sources := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableSourceArchive)).List()
	if len(sources) == 0 {
		return data, ErrNoSourceArchive
	}
	source := sources[0]
	sourceURL, err := tmpl.New(ctx).WithArtifact(source).Apply(urlTemplate)
	if err != nil {
		return data, err
	}
	sourceSum, err := source.Checksum("sha256")
	if err != nil {
		return data, err
	}

	bottles := ctx.Artifacts.Filter(artifact.And(
		artifact.ByType(artifact.HomebrewBottle),
		artifact.ByIDs(cfg.Name),
	)).List()
	if len(bottles) == 0 {
		return data, ErrNoBinaries
	}
	slices.SortFunc(bottles, func(a, b *artifact.Artifact) int {
		return cmp.Compare(bottleOrder(a), bottleOrder(b))
	})

	var rootURL string
	for _, b := range bottles {
		u, err := tmpl.New(ctx).WithArtifact(b).Apply(urlTemplate)
		if err != nil {
			return data, err
		}
		root, ok := strings.CutSuffix(u, "/"+b.Name)
		if !ok {
			return data, fmt.Errorf("url_template must end in the artifact name, got %q", u)
		}
		if rootURL != "" && root != rootURL {
			return data, fmt.Errorf("bottles resolve to different root URLs: %q and %q", rootURL, root)
		}
		rootURL = root
		sum, err := b.Checksum("sha256")
		if err != nil {
			return data, err
		}
		data.Bottles = append(data.Bottles, bottleData{
			Tag:    artifact.MustExtra[string](*b, extraTag),
			SHA256: sum,
		})
	}

	install, err := installFor(ctx, cfg)
	if err != nil {
		return data, err
	}

	deps := slices.Clone(cfg.Dependencies)
	if !slices.ContainsFunc(deps, func(d config.HomebrewDependency) bool { return d.Name == "go" }) && cfg.Install == "" {
		deps = append(deps, config.HomebrewDependency{Name: "go", Type: "build"})
	}
	slices.SortStableFunc(deps, func(a, b config.HomebrewDependency) int {
		if c := cmp.Compare(depPriority(a.Type), depPriority(b.Type)); c != 0 {
			return c
		}
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	t := tmpl.New(ctx)
	apply := func(s string) ([]string, error) {
		out, err := t.Apply(s)
		return split(out), err
	}
	desc, err := t.Apply(cfg.Description)
	if err != nil {
		return data, err
	}
	homepage, err := t.Apply(cfg.Homepage)
	if err != nil {
		return data, err
	}
	data = templateData{
		Class:        formulaClass(cfg.Name),
		Desc:         desc,
		Homepage:     homepage,
		Version:      ctx.Version,
		License:      cfg.License,
		SourceURL:    sourceURL,
		SourceSHA256: sourceSum,
		RootURL:      rootURL,
		Bottles:      data.Bottles,
		Dependencies: deps,
		Conflicts:    cfg.Conflicts,
		Install:      install,
	}
	for dst, src := range map[*[]string]string{
		&data.CustomBlock: cfg.CustomBlock,
		&data.PostInstall: cfg.PostInstall,
		&data.Caveats:     cfg.Caveats,
		&data.Service:     cfg.Service,
		&data.Tests:       cfg.Test,
	} {
		if *dst, err = apply(src); err != nil {
			return data, err
		}
	}
	return data, nil
}

// bottleOrder sorts bottles the way Homebrew writes them: macOS ARM, macOS
// Intel, then Linux.
func bottleOrder(a *artifact.Artifact) string {
	tag := artifact.MustExtra[string](*a, extraTag)
	switch {
	case strings.HasSuffix(tag, "_linux"):
		return "2" + tag
	case strings.HasPrefix(tag, "arm64_"):
		return "0" + tag
	default:
		return "1" + tag
	}
}

// installFor returns the formula's install block, used only when building
// from source. By default it rebuilds every bottled binary with the same
// main package, ldflags and CGO setting GoReleaser used.
func installFor(ctx *context.Context, cfg config.HomebrewBottle) ([]string, error) {
	t := tmpl.New(ctx)
	extra, err := t.Apply(cfg.ExtraInstall)
	if err != nil {
		return nil, err
	}
	if cfg.Install != "" {
		install, err := t.Apply(cfg.Install)
		if err != nil {
			return nil, err
		}
		return append(split(install), split(extra)...), nil
	}

	byTag, err := binaries(ctx, cfg)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var lines []string
	for _, tag := range slices.Sorted(maps.Keys(byTag)) {
		for _, bin := range byTag[tag] {
			if seen[bin.Name] {
				continue
			}
			seen[bin.Name] = true
			l, err := goBuildLines(ctx, bin)
			if err != nil {
				return nil, err
			}
			lines = append(lines, l...)
		}
	}
	return append(lines, split(extra)...), nil
}

func goBuildLines(ctx *context.Context, bin *artifact.Artifact) ([]string, error) {
	id := artifact.ExtraOr(*bin, artifact.ExtraID, "")
	idx := slices.IndexFunc(ctx.Config.Builds, func(b config.Build) bool { return b.ID == id })
	if idx < 0 {
		return nil, fmt.Errorf("no build with id %q for binary %q; set install", id, bin.Name)
	}
	build := ctx.Config.Builds[idx]
	if build.Builder != "" && build.Builder != "go" {
		return nil, fmt.Errorf("build %q uses the %s builder; set install", id, build.Builder)
	}

	ldflags, err := tmpl.New(ctx).WithArtifact(bin).Apply(strings.Join(build.Ldflags, " "))
	if err != nil {
		return nil, err
	}
	main := cmp.Or(build.UnproxiedMain, build.Main, ".")
	if !strings.HasPrefix(main, ".") && !strings.HasPrefix(main, "/") {
		main = "./" + main
	}

	args := []string{fmt.Sprintf("output: bin/%s", rubyQuote(bin.Name))}
	if ldflags != "" {
		args = append([]string{"ldflags: " + rubyQuote(ldflags)}, args...)
	}
	cmd := fmt.Sprintf(`system "go", "build", *std_go_args(%s), %s`, strings.Join(args, ", "), rubyQuote(main))

	var lines []string
	for _, env := range build.Env {
		if k, v, ok := strings.Cut(env, "="); ok && k == "CGO_ENABLED" {
			lines = append(lines, fmt.Sprintf(`ENV["CGO_ENABLED"] = %s`, rubyQuote(v)))
		}
	}
	if dir := cmp.Or(build.UnproxiedDir, build.Dir); dir != "" && path.Clean(dir) != "." {
		return append(lines, fmt.Sprintf("cd %s do", rubyQuote(dir)), "  "+cmd, "end"), nil
	}
	return append(lines, cmd), nil
}

func render(data templateData) (string, error) {
	t, err := template.New("formula.rb").
		Funcs(template.FuncMap{"ruby": rubyQuote}).
		ParseFS(formulaTemplate, "templates/formula.rb")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	var out strings.Builder
	s := bufio.NewScanner(&buf)
	for s.Scan() {
		out.WriteString(strings.TrimRight(s.Text(), " "))
		out.WriteByte('\n')
	}
	return out.String(), s.Err()
}

func doPublish(ctx *context.Context, formula *artifact.Artifact, cl client.Client) error {
	cfg := artifact.MustExtra[config.HomebrewBottle](*formula, extraConfig)
	switch strings.TrimSpace(cfg.SkipUpload) {
	case "true":
		return pipe.Skip("homebrew_bottles.skip_upload is set")
	case "auto":
		if ctx.Semver.Prerelease != "" {
			return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
		}
	}

	repo := client.RepoFromRef(cfg.Repository)
	gpath := path.Join(cfg.Directory, formula.Name)
	msg, err := tmpl.New(ctx).Apply(cfg.CommitMessageTemplate)
	if err != nil {
		return err
	}
	author, err := commitauthor.Get(ctx, cfg.CommitAuthor)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(formula.Path)
	if err != nil {
		return err
	}

	if cfg.Repository.Git.URL != "" {
		if err := client.NewGitUploadClient(repo.Branch).
			CreateFile(ctx, author, repo, content, gpath, msg); err != nil {
			return err
		}
		summary.Appendf("Updated homebrew formula `%s` in `%s`", formula.Name, cmp.Or(repo.String(), cfg.Repository.Git.URL))
		return nil
	}

	cl, err = client.NewIfToken(ctx, cl, cfg.Repository.Token)
	if err != nil {
		return err
	}
	base := client.Repo{
		Name:   cfg.Repository.PullRequest.Base.Name,
		Owner:  cfg.Repository.PullRequest.Base.Owner,
		Branch: cfg.Repository.PullRequest.Base.Branch,
	}
	if fscli, ok := cl.(client.ForkSyncer); ok && cfg.Repository.PullRequest.Enabled {
		if err := fscli.SyncFork(ctx, repo, base); err != nil {
			log.WithError(err).Warn("could not sync fork")
		}
	}
	if err := cl.CreateFile(ctx, author, repo, content, gpath, msg); err != nil {
		return err
	}
	if !cfg.Repository.PullRequest.Enabled {
		summary.Appendf("Updated homebrew formula `%s` in `%s`", formula.Name, repo.String())
		return nil
	}

	prcl, err := client.NewIfToken(ctx, cl, cfg.Repository.PullRequest.Token)
	if err != nil {
		return err
	}
	pcl, ok := prcl.(client.PullRequestOpener)
	if !ok {
		return errors.New("client does not support pull requests")
	}
	url, err := pcl.OpenPullRequest(ctx, base, repo, msg, cfg.Repository.PullRequest.Draft)
	if err != nil {
		return err
	}
	if url != "" {
		summary.Appendf("Opened pull request to `%s` (homebrew formula `%s`): %s", cmp.Or(base.String(), repo.String()), formula.Name, url)
	}
	return nil
}

// rubyQuote returns s as a double-quoted Ruby string with no interpolation.
func rubyQuote(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`, "#{", `\#{`).Replace(s)
	return `"` + s + `"`
}

func depPriority(t string) int {
	switch t {
	case "build":
		return 0
	case "test":
		return 1
	case "recommended":
		return 3
	case "optional":
		return 4
	default:
		return 2
	}
}

func split(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// formulaClass mirrors Homebrew's Formulary.class_s.
func formulaClass(name string) string {
	var out strings.Builder
	upper := true
	name = strings.ToLower(name)
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '-' || c == '_' || c == '.' || c == ' ':
			upper = true
			continue
		case c == '+':
			out.WriteByte('x')
		case c == '@' && i+1 < len(name) && name[i+1] >= '0' && name[i+1] <= '9':
			out.WriteString("AT")
		case upper:
			out.WriteString(strings.ToUpper(string(c)))
		default:
			out.WriteByte(c)
		}
		upper = false
	}
	return out.String()
}
