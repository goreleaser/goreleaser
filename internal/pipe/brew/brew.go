package brew

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/crypto/ssh"
)

const brewConfigExtra = "BrewConfig"

var (
	// ErrNoArchivesFound happens when 0 archives are found.
	ErrNoArchivesFound = errors.New("no linux/macos archives found")

	// ErrMultipleArchivesSameOS happens when the config yields multiple archives
	// for linux or windows.
	ErrMultipleArchivesSameOS = errors.New("one tap can handle only archive of an OS/Arch combination. Consider using ids in the brew section")
)

// Pipe for brew deployment.
type Pipe struct{}

func (Pipe) String() string                 { return "homebrew tap formula" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Brews) == 0 }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Brews {
		brew := &ctx.Config.Brews[i]

		brew.CommitAuthor = commitauthor.Default(brew.CommitAuthor)

		if brew.CommitMessageTemplate == "" {
			brew.CommitMessageTemplate = "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if brew.Name == "" {
			brew.Name = ctx.Config.ProjectName
		}
		if brew.Goarm == "" {
			brew.Goarm = "6"
		}
		if brew.Goamd64 == "" {
			brew.Goamd64 = "v1"
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
	brew, err := artifact.Extra[config.Homebrew](*formula, brewConfigExtra)
	if err != nil {
		return err
	}

	if strings.TrimSpace(brew.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}

	if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
	}

	gpath := buildFormulaPath(brew.Folder, formula.Name)

	if len(brew.Tap.GitURL) > 0 {
		log.WithField("formula", gpath).WithField("repo", brew.Tap.GitURL).Info("pushing")
		return doGitPublish(ctx, brew, formula)
	}

	cl, err = client.NewIfToken(ctx, cl, brew.Tap.Token)
	if err != nil {
		return err
	}

	repo := client.RepoFromRef(brew.Tap)

	log.WithField("formula", gpath).
		WithField("repo", repo.String()).
		Info("pushing")

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

	return cl.CreateFile(ctx, author, repo, content, gpath, msg)
}

func doGitPublish(ctx *context.Context, brew config.Homebrew, formula *artifact.Artifact) error {
	key, err := tmpl.New(ctx).Apply(brew.Tap.PrivateKey)
	if err != nil {
		return err
	}

	key, err = keyPath(key)
	if err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(brew.Tap.GitURL)
	if err != nil {
		return err
	}

	if url == "" {
		return pipe.Skip("brew.tap.git_url is empty")
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(brew.Tap.GitSSHCommand)
	if err != nil {
		return err
	}

	msg, err := tmpl.New(ctx).Apply(brew.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, brew.CommitAuthor)
	if err != nil {
		return err
	}

	parent := filepath.Join(ctx.Config.Dist, "brew", "repos")
	cwd := filepath.Join(parent, brew.Name)

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshcmd)}

	if err := runGitCmds(ctx, parent, env, [][]string{
		{"clone", url, brew.Name},
	}); err != nil {
		return fmt.Errorf("failed to setup local Brew repo: %w", err)
	}

	if err := runGitCmds(ctx, cwd, env, [][]string{
		// setup auth et al
		{"config", "--local", "user.name", author.Name},
		{"config", "--local", "user.email", author.Email},
		{"config", "--local", "commit.gpgSign", "false"},
		{"config", "--local", "init.defaultBranch", "master"},
	}); err != nil {
		return fmt.Errorf("failed to setup local Brew repo: %w", err)
	}

	bts, err := os.ReadFile(formula.Path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", formula.Name, err)
	}

	if err := os.WriteFile(filepath.Join(cwd, formula.Name), bts, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", formula.Name, err)
	}

	log.WithField("repo", url).WithField("name", brew.Name).Info("pushing")
	if err := runGitCmds(ctx, cwd, env, [][]string{
		{"add", "-A", "."},
		{"commit", "-m", msg},
		{"push", "origin", "HEAD"},
	}); err != nil {
		return fmt.Errorf("failed to push %q (%q): %w", brew.Name, url, err)
	}

	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("brew.tap.private_key is empty")
	}

	path := key
	if _, err := ssh.ParsePrivateKey([]byte(key)); err == nil {
		// if it can be parsed as a valid private key, we write it to a
		// temp file and use that path on GIT_SSH_COMMAND.
		f, err := os.CreateTemp("", "id_*")
		if err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		defer f.Close()

		// the key needs to EOF at an empty line, seems like github actions
		// is somehow removing them.
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}

		if _, err := io.WriteString(f, key); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		path = f.Name()
	}

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("could not stat brew.tap.private_key: %w", err)
	}

	// in any case, ensure the key has the correct permissions.
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("failed to ensure brew.tap.private_key permissions: %w", err)
	}

	return path, nil
}

func runGitCmds(ctx *context.Context, cwd string, env []string, cmds [][]string) error {
	for _, cmd := range cmds {
		args := append([]string{"-C", cwd}, cmd...)
		if _, err := git.Clean(git.RunWithEnv(ctx, env, args...)); err != nil {
			return fmt.Errorf("%q failed: %w", strings.Join(cmd, " "), err)
		}
	}
	return nil
}

func doRun(ctx *context.Context, brew config.Homebrew, cl client.Client) error {
	if brew.Tap.Name == "" {
		return pipe.Skip("brew tap name is not set")
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
				artifact.ByFormats("zip", "tar.gz"),
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
		return ErrNoArchivesFound
	}

	name, err := tmpl.New(ctx).Apply(brew.Name)
	if err != nil {
		return err
	}
	brew.Name = name

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, brew.Tap)
	if err != nil {
		return err
	}
	brew.Tap = ref

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

func installs(cfg config.Homebrew, art *artifact.Artifact) []string {
	if cfg.Install != "" {
		return split(cfg.Install)
	}

	install := map[string]bool{}
	switch art.Type {
	case artifact.UploadableBinary:
		name := art.Name
		bin := artifact.ExtraOr(*art, artifact.ExtraBinary, art.Name)
		install[fmt.Sprintf("bin.install %q => %q", name, bin)] = true
	case artifact.UploadableArchive:
		for _, bin := range artifact.ExtraOr(*art, artifact.ExtraBinaries, []string{}) {
			install[fmt.Sprintf("bin.install %q", bin)] = true
		}
	}

	result := keys(install)
	sort.Strings(result)
	log.Warnf("guessing install to be %q", strings.Join(result, ", "))
	return result
}

func keys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

		url, err := tmpl.New(ctx).WithArtifact(art, map[string]string{}).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}

		pkg := releasePackage{
			DownloadURL:      url,
			SHA256:           sum,
			OS:               art.Goos,
			Arch:             art.Goarch,
			DownloadStrategy: cfg.DownloadStrategy,
			Install:          installs(cfg, art),
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
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "@", "AT")
	return strings.ReplaceAll(strings.Title(name), " ", "") // nolint:staticcheck
}
