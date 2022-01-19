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
	pkgBuildExtra     = "AURConfig"
	defaultSSHCommand = "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i {{ .KeyPath }} -F /dev/null"
	defaultCommitMsg  = "Update to {{ .Tag }}"
)

var ErrNoArchivesFound = errors.New("no linux archives found")

// Pipe for arch linux's AUR pkgbuild.
type Pipe struct{}

func (Pipe) String() string                 { return "aur pkgbuild" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.PkgBuilds) == 0 }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.PkgBuilds {
		pkg := &ctx.Config.PkgBuilds[i]

		pkg.CommitAuthor = commitauthor.Default(pkg.CommitAuthor)
		if pkg.CommitMessageTemplate == "" {
			pkg.CommitMessageTemplate = defaultCommitMsg
		}
		if pkg.Name == "" {
			pkg.Name = ctx.Config.ProjectName + "-bin"
			if len(pkg.Conflicts) == 0 {
				pkg.Conflicts = []string{ctx.Config.ProjectName}
			}
			if len(pkg.Provides) == 0 {
				pkg.Provides = []string{ctx.Config.ProjectName}
			}
		}
		if pkg.Rel == "" {
			pkg.Rel = "1"
		}
		if pkg.SSHCommand == "" {
			pkg.SSHCommand = defaultSSHCommand
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
	for _, pkgbuild := range ctx.Config.PkgBuilds {
		err := doRun(ctx, pkgbuild, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, pkgbuild config.PkgBuild, cl client.Client) error {
	name, err := tmpl.New(ctx).Apply(pkgbuild.Name)
	if err != nil {
		return err
	}
	pkgbuild.Name = name

	if pkgbuild.Name == "" {
		return pipe.Skip("package name is not set")
	}

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
	if len(pkgbuild.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(pkgbuild.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	pkg, err := tmpl.New(ctx).Apply(pkgbuild.Package)
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
				pkg = fmt.Sprintf(`install -Dm755 "./%s "${pkgdir}/usr/bin/%[1]s"`, bin)
				break
			}
		}
		log.Warnf("guessing package to be %q", pkg)
	}
	pkgbuild.Package = pkg

	content, err := buildPkgBuild(ctx, pkgbuild, cl, archives)
	if err != nil {
		return err
	}

	pkgbuildPath := filepath.Join(ctx.Config.Dist, "aur", "pkgbuilds", pkgbuild.Name)
	if err := os.MkdirAll(filepath.Dir(pkgbuildPath), 0o755); err != nil {
		return fmt.Errorf("failed to write PKGBUILD: %w", err)
	}
	log.WithField("pkgbuild", pkgbuildPath).Info("writing")
	if err := os.WriteFile(pkgbuildPath, []byte(content), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write PKGBUILD: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "PKGBUILD",
		Path: pkgbuildPath,
		Type: artifact.PkgBuild,
		Extra: map[string]interface{}{
			pkgBuildExtra: pkgbuild,
		},
	})

	return nil
}

func buildPkgBuild(ctx *context.Context, goFish config.PkgBuild, client client.Client, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, goFish, client, artifacts)
	if err != nil {
		return "", err
	}
	return doBuildPkgBuild(ctx, data)
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

func doBuildPkgBuild(ctx *context.Context, data templateData) (string, error) {
	t := template.Must(
		template.New(data.Name).
			Funcs(template.FuncMap{
				"fixLines": fixLines,
				"pkgArray": toPkgBuildArray,
			}).
			Parse(pkgBuildTemplate),
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

func dataFor(ctx *context.Context, cfg config.PkgBuild, cl client.Client, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{
		Name:         cfg.Name,
		Desc:         cfg.Description,
		Homepage:     cfg.Homepage,
		Version:      ctx.Version,
		License:      cfg.License,
		Rel:          cfg.Rel,
		Maintainer:   cfg.Maintainer,
		Contributors: cfg.Contributors,
		Provides:     cfg.Provides,
		Conflicts:    cfg.Conflicts,
		Depends:      cfg.Depends,
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

// Publish the PKGBUILD to the AUR repository.
func (Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
}

func publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, pkg := range ctx.Artifacts.Filter(artifact.ByType(artifact.PkgBuild)).List() {
		err := doPublish(ctx, pkg, cli)
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

func doPublish(ctx *context.Context, pkg *artifact.Artifact, _ client.Client) error {
	cfg := pkg.Extra[pkgBuildExtra].(config.PkgBuild)

	if strings.TrimSpace(cfg.SkipUpload) == "true" {
		return pipe.Skip("pkgbuild.skip_upload is set")
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
		return pipe.Skip("pkgbuild.git_url is empty")
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(cfg.SSHCommand)
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

	cwd := filepath.Join(ctx.Config.Dist, "aur", "repos")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		return err
	}

	if _, err := git.Clean(git.Run(
		"-C", cwd,
		"-c", "ssh.variant=ssh",
		"-c", fmt.Sprintf(`core.sshCommand="%s"`, sshcmd),
		"clone", url, cfg.Name,
	)); err != nil {
		return fmt.Errorf("failed to clone %q: %w", url, err)
	}

	bts, err := os.ReadFile(pkg.Path)
	if err != nil {
		return fmt.Errorf("failed to clone %q: %w", url, err)
	}

	if err := os.WriteFile(filepath.Join(cwd, cfg.Name, "PKGBUILD"), bts, 0o644); err != nil {
		return err
	}

	if _, err := git.Clean(git.Run(
		"-C", filepath.Join(cwd, cfg.Name),
		"add", "-A", ".",
	)); err != nil {
		return fmt.Errorf("failed to clone %q (%q): %w", cfg.Name, url, err)
	}

	log.WithField("repo", url).WithField("name", cfg.Name).Info("pushing")

	if _, err := git.Clean(git.Run(
		"-C", filepath.Join(cwd, cfg.Name),
		"-c", fmt.Sprintf("user.name='%s'", author.Name),
		"-c", fmt.Sprintf("user.email='%s'", author.Email),
		"-c", "commit.gpgSign=false",
		"commit", "-m", msg,
	)); err != nil {
		return fmt.Errorf("failed to commit PKGBUILD: %w", err)
	}

	if _, err := git.Clean(git.Run(
		"-C", filepath.Join(cwd, cfg.Name),
		"-c", "ssh.variant=ssh",
		"-c", fmt.Sprintf(`core.sshCommand="%s"`, sshcmd),
		"push", "origin", "HEAD",
	)); err != nil {
		return fmt.Errorf("failed to push %q (%q): %w", cfg.Name, url, err)
	}

	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("pkgbuild.private_key is empty")
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
		if err := os.Chmod(f.Name(), 0400); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		return f.Name(), nil
	}

	if _, err := os.Stat(key); os.IsNotExist(err) {
		return "", fmt.Errorf("key %q does not exist", key)
	}
	return key, nil
}
