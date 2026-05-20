package apkbuild

import (
	"bufio"
	"bytes"
	"cmp"
	"crypto/sha256"
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
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	apkbuildExtra    = "APKBuildConfig"
	defaultCommitMsg = "Update to {{ .Tag }}"
)

var ErrNoArchivesFound = errors.New("no linux archives found")

type Pipe struct{}

func (Pipe) String() string        { return "alpine linux apkbuild" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.APKBuild) || len(ctx.Config.APKBuilds) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.APKBuilds {
		pkg := &ctx.Config.APKBuilds[i]

		pkg.CommitAuthor = commitauthor.Default(pkg.CommitAuthor)

		if pkg.CommitMessageTemplate == "" {
			pkg.CommitMessageTemplate = defaultCommitMsg
		}
		if pkg.Name == "" {
			pkg.Name = ctx.Config.ProjectName
		}
		if pkg.Rel == "" {
			pkg.Rel = "0"
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
	skips := pipe.SkipMemento{}

	for _, apk := range ctx.Config.APKBuilds {
		disable, err := tmpl.New(ctx).Bool(apk.Disable)
		if err != nil {
			return err
		}
		if disable {
			skips.Remember(pipe.Skip("configuration is disabled"))
			continue
		}

		if err := doRun(ctx, apk, cli); err != nil {
			return err
		}
	}

	return skips.Evaluate()
}

func doRun(ctx *context.Context, apk config.APKBuild, cl client.ReleaseURLTemplater) error {
	if err := tmpl.New(ctx).ApplyAll(
		&apk.Name,
		&apk.Directory,
		&apk.SkipUpload,
	); err != nil {
		return err
	}

	filters := []artifact.Filter{
		artifact.ByGoos("linux"),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(apk.Goamd64),
			),
			artifact.ByGoarch("arm64"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.ByGoarm("7"),
			),
			artifact.ByGoarch("386"),
		),
		artifact.ByTypes(
			artifact.UploadableArchive,
			artifact.UploadableBinary,
		),
	}

	if len(apk.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(apk.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	// deterministic selection
	slices.SortFunc(archives, func(a, b *artifact.Artifact) int {
		return cmp.Compare(a.Goarch+a.Goarm, b.Goarch+b.Goarm)
	})

	art := archives[0]

	pkg, err := tmpl.New(ctx).Apply(apk.Package)
	if err != nil {
		return err
	}

	if strings.TrimSpace(pkg) == "" {
		switch art.Type {
		case artifact.UploadableBinary:
			name := art.Name
			bin := artifact.MustExtra[string](*art, artifact.ExtraBinary)

			pkg = fmt.Sprintf(
				"install -Dm755 %q %q",
				"./"+name,
				"$pkgdir/usr/bin/"+bin,
			)

		case artifact.UploadableArchive:
			folder := artifact.ExtraOr(*art, artifact.ExtraWrappedIn, ".")

			var cmds []string
			for _, bin := range artifact.MustExtra[[]string](*art, artifact.ExtraBinaries) {
				p := filepath.ToSlash(filepath.Clean(filepath.Join(folder, bin)))
				cmds = append(cmds, fmt.Sprintf(
					"install -Dm755 %q %q",
					"./"+p,
					"$pkgdir/usr/bin/"+bin,
				))
			}
			pkg = strings.Join(cmds, "\n")
		}

		log.Warnf("guessing package to be %q", pkg)
	}

	apk.Package = pkg

	pkgContent, err := buildAPKBuildFile(ctx, apk, cl, archives)
	if err != nil {
		return err
	}

	outPath := filepath.Join(ctx.Config.Dist, "apkbuild", apk.Name+".apkbuild")

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("failed to create apkbuild dir: %w", err)
	}

	log.WithField("file", outPath).Info("writing")

	if err := os.WriteFile(outPath, []byte(pkgContent), 0o644); err != nil {
		return fmt.Errorf("failed to write APKBUILD: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "APKBUILD",
		Path: outPath,
		Type: artifact.APKBuild,
		Extra: map[string]any{
			apkbuildExtra:    apk,
			artifact.ExtraID: apk.Name,
		},
	})

	return nil
}

func buildAPKBuildFile(ctx *context.Context, pkg config.APKBuild, cl client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, pkg, cl, artifacts)
	if err != nil {
		return "", err
	}
	return applyTemplate(ctx, apkbuildTemplate, data)
}

func applyTemplate(ctx *context.Context, tpl string, data templateData) (string, error) {
	t := template.New("apkbuild").Funcs(template.FuncMap{
		"fixLines":   fixLines,
		"replaceAll": strings.ReplaceAll,
	})

	t = template.Must(t.Parse(tpl))

	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return "", err
	}

	content, err := tmpl.New(ctx).Apply(out.String())
	if err != nil {
		return "", err
	}

	out.Reset()

	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		out.WriteString(strings.TrimRight(sc.Text(), " "))
		out.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return "", err
	}

	return out.String(), nil
}

func fixLines(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			out = append(out, "")
			continue
		}
		out = append(out, "\t"+l)
	}
	return strings.Join(out, "\n")
}

// FIXED: proper Goarch/Goarm separation
func toAPKArch(goarch, goarm string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "386":
		return "x86"
	case "arm64":
		return "aarch64"
	case "arm":
		if goarm == "7" {
			return "armv7"
		}
		return "arm"
	default:
		return "invalid"
	}
}

func dataFor(ctx *context.Context, cfg config.APKBuild, cl client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{
		Name:         cfg.Name,
		Desc:         cfg.Description,
		Homepage:     cfg.Homepage,
		Version:      strings.ReplaceAll(ctx.Version, "-", "_"),
		License:      cfg.License,
		Rel:          cfg.Rel,
		Maintainers:  cfg.Maintainers,
		Contributors: cfg.Contributors,
		Depends:      cfg.Depends,
		MakeDepends:  cfg.MakeDepends,
		Package:      cfg.Package,
	}

	urlTemplate := cfg.URLTemplate
	if urlTemplate == "" {
		u, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return result, err
		}
		urlTemplate = u
	}

	for _, art := range artifacts {
		sum, err := art.Checksum("sha512")
		if err != nil {
			sum, err = art.Checksum("sha256")
			if err != nil {
				return result, err
			}
		}

		url, err := tmpl.New(ctx).WithArtifact(art).Apply(urlTemplate)
		if err != nil {
			return result, err
		}

		rp := releasePackage{
			DownloadURL: url,
			SHA512:      sum,
			Arch:        toAPKArch(art.Goarch, art.Goarm),
			Format:      art.Format(),
		}

		result.ReleasePackages = append(result.ReleasePackages, rp)
		result.Arches = append(result.Arches, rp.Arch)
	}

	slices.Sort(result.Arches)
	slices.SortFunc(result.ReleasePackages, func(a, b releasePackage) int {
		return cmp.Compare(a.Arch, b.Arch)
	})

	return result, nil
}

func (Pipe) Publish(ctx *context.Context) error {
	skips := pipe.SkipMemento{}

	for _, pkgs := range ctx.Artifacts.Filter(
		artifact.ByType(artifact.APKBuild),
	).GroupByID() {
		if err := doPublish(ctx, pkgs); err != nil && pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		} else if err != nil {
			return err
		}
	}

	return skips.Evaluate()
}

func doPublish(ctx *context.Context, pkgs []*artifact.Artifact) error {
	cfg := artifact.MustExtra[config.APKBuild](*pkgs[0], apkbuildExtra)

	if strings.TrimSpace(cfg.SkipUpload) == "true" {
		return pipe.Skip("apkbuild.skip_upload is set")
	}
	if strings.TrimSpace(cfg.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected, skipping upload")
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
