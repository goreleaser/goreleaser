package aur

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
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
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	aurExtra         = "AURConfig"
	defaultCommitMsg = "Update to {{ .Tag }}"
)

var ErrNoArchivesFound = errors.New("no linux archives found")

// Pipe for arch linux's AUR pkgbuild.
type Pipe struct{}

func (Pipe) String() string        { return "arch user repositories" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.AUR) || len(ctx.Config.AURs) == 0
}

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
		if pkg.Goamd64 == "" {
			pkg.Goamd64 = "v1"
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

func runAll(ctx *context.Context, cli client.ReleaseURLTemplater) error {
	for _, aur := range ctx.Config.AURs {
		err := doRun(ctx, aur, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, aur config.AUR, cl client.ReleaseURLTemplater) error {
	if err := tmpl.New(ctx).ApplyAll(
		&aur.Name,
		&aur.Directory,
	); err != nil {
		return err
	}

	filters := []artifact.Filter{
		artifact.ByGoos("linux"),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(aur.Goamd64),
			),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("386"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.Or(
					artifact.ByGoarm("7"),
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
			bin := artifact.ExtraOr(*art, artifact.ExtraBinary, art.Name)
			pkg = fmt.Sprintf(`install -Dm755 "./%s "${pkgdir}/usr/bin/%s"`, name, bin)
		case artifact.UploadableArchive:
			folder := artifact.ExtraOr(*art, artifact.ExtraWrappedIn, ".")
			for _, bin := range artifact.ExtraOr(*art, artifact.ExtraBinaries, []string{}) {
				path := filepath.ToSlash(filepath.Clean(filepath.Join(folder, bin)))
				pkg = fmt.Sprintf(`install -Dm755 "./%s" "${pkgdir}/usr/bin/%s"`, path, bin)
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

func buildPkgFile(ctx *context.Context, pkg config.AUR, client client.ReleaseURLTemplater, artifacts []*artifact.Artifact, tpl string) (string, error) {
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

func dataFor(ctx *context.Context, cfg config.AUR, cl client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (templateData, error) {
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
		Backup:       cfg.Backup,
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
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}

		releasePackage := releasePackage{
			DownloadURL: url,
			SHA256:      sum,
			Arch:        toPkgBuildArch(art.Goarch + art.Goarm),
			Format:      artifact.ExtraOr(*art, artifact.ExtraFormat, ""),
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
	cfg, err := artifact.Extra[config.AUR](*pkgs[0], aurExtra)
	if err != nil {
		return err
	}

	if strings.TrimSpace(cfg.SkipUpload) == "true" {
		return pipe.Skip("aur.skip_upload is set")
	}

	if strings.TrimSpace(cfg.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping aur publish")
	}

	author, err := commitauthor.Get(ctx, cfg.CommitAuthor)
	if err != nil {
		return err
	}

	msg, err := tmpl.New(ctx).Apply(cfg.CommitMessageTemplate)
	if err != nil {
		return err
	}

	cli := client.NewGitUploadClient("master")
	repo := client.RepoFromRef(config.RepoRef{
		Git: config.GitRepoRef{
			PrivateKey: cfg.PrivateKey,
			URL:        cfg.GitURL,
			SSHCommand: cfg.GitSSHCommand,
		},
		Name: fmt.Sprintf("%x", sha256.Sum256([]byte(cfg.GitURL))),
	})

	files := make([]client.RepoFile, 0, len(pkgs))
	for _, pkg := range pkgs {
		content, err := os.ReadFile(pkg.Path)
		if err != nil {
			return err
		}
		files = append(files, client.RepoFile{
			Path:    path.Join(cfg.Directory, pkg.Name),
			Content: content,
		})
	}
	return cli.CreateFiles(ctx, author, repo, msg, files)
}
