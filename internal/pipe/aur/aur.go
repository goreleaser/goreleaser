package aur

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/apex/log"
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

const (
	aurExtra          = "AURConfig"
	defaultSSHCommand = "ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null"
	defaultCommitMsg  = "Update to {{ .Tag }}"
)

var ErrNoArchivesFound = errors.New("no linux archives found")

// Pipe for arch linux's AUR pkgbuild.
type Pipe struct{}

func (Pipe) String() string                 { return "arch user repositories" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.AURs) == 0 }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.AURs {
		pkg := &ctx.Config.AURs[i]

		pkg.CommitAuthor = commitauthor.Default(pkg.CommitAuthor)
		if pkg.CommitMessageTemplate == "" {
			pkg.CommitMessageTemplate = defaultCommitMsg
		}
		if pkg.Name == "" {
			pkg.Name = ctx.Config.ProjectName
		}
		if !strings.HasSuffix(pkg.Name, "-bin") {
			pkg.Name += "-bin"
		}
		if len(pkg.Conflicts) == 0 {
			pkg.Conflicts = []string{ctx.Config.ProjectName}
		}
		if len(pkg.Provides) == 0 {
			pkg.Provides = []string{ctx.Config.ProjectName}
		}
		if pkg.Rel == "" {
			pkg.Rel = "1"
		}
		if pkg.GitSSHCommand == "" {
			pkg.GitSSHCommand = defaultSSHCommand
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

func runAll(ctx *context.Context, cli client.Client) error {
	for _, aur := range ctx.Config.AURs {
		err := doRun(ctx, aur, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, aur config.AUR, cl client.Client) error {
	name, err := tmpl.New(ctx).Apply(aur.Name)
	if err != nil {
		return err
	}
	aur.Name = name

	filters := []artifact.Filter{
		artifact.ByGoos("linux"),
		artifact.Or(
			artifact.ByGoarch("amd64"),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("386"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.Or(
					artifact.ByGoarm("7"),
					artifact.ByGoarm("6"),
				),
			),
		),
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
		),
	}
	if len(aur.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(aur.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	pkg, err := tmpl.New(ctx).Apply(aur.Package)
	if err != nil {
		return err
	}
	if strings.TrimSpace(pkg) == "" {
		art := archives[0]
		switch art.Type {
		case artifact.UploadableBinary:
			name := art.Name
			bin := art.ExtraOr(artifact.ExtraBinary, art.Name).(string)
			pkg = fmt.Sprintf(`install -Dm755 "./%s "${pkgdir}/usr/bin/%s"`, name, bin)
		case artifact.UploadableArchive:
			for _, bin := range art.ExtraOr(artifact.ExtraBinaries, []string{}).([]string) {
				pkg = fmt.Sprintf(`install -Dm755 "./%s" "${pkgdir}/usr/bin/%[1]s"`, bin)
				break
			}
		}
		log.Warnf("guessing package to be %q", pkg)
	}
	aur.Package = pkg

	for _, info := range []struct {
		name, tpl, ext string
		kind           artifact.Type
	}{
		{
			name: "PKGBUILD",
			tpl:  aurTemplateData,
			ext:  ".pkgbuild",
			kind: artifact.PkgBuild,
		},
		{
			name: ".SRCINFO",
			tpl:  srcInfoTemplate,
			ext:  ".srcinfo",
			kind: artifact.SrcInfo,
		},
	} {
		pkgContent, err := buildPkgFile(ctx, aur, cl, archives, info.tpl)
		if err != nil {
			return err
		}

		path := filepath.Join(ctx.Config.Dist, "aur", aur.Name+info.ext)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("failed to write %s: %w", info.kind, err)
		}
		log.WithField("file", path).Info("writing")
		if err := os.WriteFile(path, []byte(pkgContent), 0o644); err != nil { //nolint: gosec
			return fmt.Errorf("failed to write %s: %w", info.kind, err)
		}

		ctx.Artifacts.Add(&artifact.Artifact{
			Name: info.name,
			Path: path,
			Type: info.kind,
			Extra: map[string]interface{}{
				aurExtra:         aur,
				artifact.ExtraID: aur.Name,
			},
		})
	}

	return nil
}

func buildPkgFile(ctx *context.Context, pkg config.AUR, client client.Client, artifacts []*artifact.Artifact, tpl string) (string, error) {
	data, err := dataFor(ctx, pkg, client, artifacts)
	if err != nil {
		return "", err
	}
	return applyTemplate(ctx, tpl, data)
}

func fixLines(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			result = append(result, "")
			continue
		}
		result = append(result, "  "+l)
	}
	return strings.Join(result, "\n")
}

func applyTemplate(ctx *context.Context, tpl string, data templateData) (string, error) {
	t := template.Must(
		template.New(data.Name).
			Funcs(template.FuncMap{
				"fixLines": fixLines,
				"pkgArray": toPkgBuildArray,
			}).
			Parse(tpl),
	)

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

func toPkgBuildArray(ss []string) string {
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		result = append(result, fmt.Sprintf("'%s'", s))
	}
	return strings.Join(result, " ")
}

func toPkgBuildArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "386":
		return "i686"
	case "arm64":
		return "aarch64"
	case "arm6":
		return "armv6h"
	case "arm7":
		return "armv7h"
	default:
		return "invalid" // should never get here
	}
}

func dataFor(ctx *context.Context, cfg config.AUR, cl client.Client, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{
		Name:         cfg.Name,
		Desc:         cfg.Description,
		Homepage:     cfg.Homepage,
		Version:      fmt.Sprintf("%d.%d.%d", ctx.Semver.Major, ctx.Semver.Minor, ctx.Semver.Patch),
		License:      cfg.License,
		Rel:          cfg.Rel,
		Maintainers:  cfg.Maintainers,
		Contributors: cfg.Contributors,
		Provides:     cfg.Provides,
		Conflicts:    cfg.Conflicts,
		Depends:      cfg.Depends,
		OptDepends:   cfg.OptDepends,
		Package:      cfg.Package,
	}

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

		releasePackage := releasePackage{
			DownloadURL: url,
			SHA256:      sum,
			Arch:        toPkgBuildArch(art.Goarch + art.Goarm),
		}
		result.ReleasePackages = append(result.ReleasePackages, releasePackage)
		result.Arches = append(result.Arches, releasePackage.Arch)
	}

	sort.Strings(result.Arches)
	sort.Slice(result.ReleasePackages, func(i, j int) bool {
		return result.ReleasePackages[i].Arch < result.ReleasePackages[j].Arch
	})
	return result, nil
}

// Publish the PKGBUILD and .SRCINFO files to the AUR repository.
func (Pipe) Publish(ctx *context.Context) error {
	skips := pipe.SkipMemento{}
	for _, pkgs := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.PkgBuild),
			artifact.ByType(artifact.SrcInfo),
		),
	).GroupByID() {
		err := doPublish(ctx, pkgs)
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

func doPublish(ctx *context.Context, pkgs []*artifact.Artifact) error {
	cfg := pkgs[0].Extra[aurExtra].(config.AUR)

	if strings.TrimSpace(cfg.SkipUpload) == "true" {
		return pipe.Skip("aur.skip_upload is set")
	}

	if strings.TrimSpace(cfg.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping aur publish")
	}

	key, err := tmpl.New(ctx).Apply(cfg.PrivateKey)
	if err != nil {
		return err
	}

	key, err = keyPath(key)
	if err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(cfg.GitURL)
	if err != nil {
		return err
	}

	if url == "" {
		return pipe.Skip("aur.git_url is empty")
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(cfg.GitSSHCommand)
	if err != nil {
		return err
	}

	msg, err := tmpl.New(ctx).Apply(cfg.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, cfg.CommitAuthor)
	if err != nil {
		return err
	}

	parent := filepath.Join(ctx.Config.Dist, "aur", "repos")
	cwd := filepath.Join(parent, cfg.Name)

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshcmd)}

	if err := runGitCmds(parent, env, [][]string{
		{"clone", url, cfg.Name},
	}); err != nil {
		return fmt.Errorf("failed to setup local AUR repo: %w", err)
	}

	if err := runGitCmds(cwd, env, [][]string{
		// setup auth et al
		{"config", "--local", "user.name", author.Name},
		{"config", "--local", "user.email", author.Email},
		{"config", "--local", "commit.gpgSign", "false"},
		{"config", "--local", "init.defaultBranch", "master"},
	}); err != nil {
		return fmt.Errorf("failed to setup local AUR repo: %w", err)
	}

	for _, pkg := range pkgs {
		bts, err := os.ReadFile(pkg.Path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", pkg.Name, err)
		}

		if err := os.WriteFile(filepath.Join(cwd, pkg.Name), bts, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", pkg.Name, err)
		}
	}

	log.WithField("repo", url).WithField("name", cfg.Name).Info("pushing")
	if err := runGitCmds(cwd, env, [][]string{
		{"add", "-A", "."},
		{"commit", "-m", msg},
		{"push", "origin", "HEAD"},
	}); err != nil {
		return fmt.Errorf("failed to push %q (%q): %w", cfg.Name, url, err)
	}

	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("aur.private_key is empty")
	}
	if _, err := ssh.ParsePrivateKey([]byte(key)); err == nil {
		f, err := os.CreateTemp("", "id_*")
		if err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		defer f.Close()
		if _, err := fmt.Fprint(f, key); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		if err := os.Chmod(f.Name(), 0o400); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		return f.Name(), nil
	}

	if _, err := os.Stat(key); os.IsNotExist(err) {
		return "", fmt.Errorf("key %q does not exist", key)
	}
	return key, nil
}

func runGitCmds(cwd string, env []string, cmds [][]string) error {
	for _, cmd := range cmds {
		args := append([]string{"-C", cwd}, cmd...)
		if _, err := git.Clean(git.RunWithEnv(env, args...)); err != nil {
			return fmt.Errorf("%q failed: %w", strings.Join(cmd, " "), err)
		}
	}
	return nil
}
