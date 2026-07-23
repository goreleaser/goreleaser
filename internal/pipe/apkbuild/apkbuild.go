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
	"github.com/goreleaser/goreleaser/v2/internal/experimental"
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

// Pipe generates and publishes Alpine Linux APKBUILD files.
type Pipe struct{}

func (Pipe) String() string        { return "alpine linux packages" }
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
		if len(pkg.Options) == 0 {
			pkg.Options = []string{"!check"}
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
	skipped := pipe.SkipMemento{}
	for _, apk := range ctx.Config.APKBuilds {
		disable, err := tmpl.New(ctx).Bool(apk.Disable)
		if err != nil {
			return err
		}
		if disable {
			skipped.Remember(pipe.Skip("configuration is disabled"))
			continue
		}
		if err := doRun(ctx, apk, cli); err != nil {
			return err
		}
	}
	return skipped.Evaluate()
}

func doRun(ctx *context.Context, apk config.APKBuild, cli client.ReleaseURLTemplater) error {
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
			artifact.ByGoarch("386"),
			artifact.ByGoarch("arm64"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.ByGoarms("6", "7"),
			),
			artifact.ByGoarches("ppc64le", "s390x", "riscv64"),
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
	slices.SortFunc(archives, func(a, b *artifact.Artifact) int {
		return cmp.Or(
			cmp.Compare(a.Goarch, b.Goarch),
			cmp.Compare(a.Goarm, b.Goarm),
			cmp.Compare(a.Name, b.Name),
		)
	})

	pkg, err := tmpl.New(ctx).Apply(apk.Package)
	if err != nil {
		return err
	}
	if strings.TrimSpace(pkg) == "" {
		pkg = defaultPackage(archives[0])
		log.Warnf("guessing package to be %q", pkg)
	}
	apk.Package = pkg

	content, err := buildAPKBuild(ctx, apk, cli, archives)
	if err != nil {
		return err
	}

	file := filepath.Join(ctx.Config.Dist, "apkbuild", apk.Name+".apkbuild")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return fmt.Errorf("failed to write APKBUILD: %w", err)
	}
	log.WithField("file", file).Info("writing")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write APKBUILD: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "APKBUILD",
		Path: file,
		Type: artifact.APKBuild,
		Extra: map[string]any{
			apkbuildExtra:    apk,
			artifact.ExtraID: apk.Name,
		},
	})
	return nil
}

func defaultPackage(art *artifact.Artifact) string {
	switch art.Type {
	case artifact.UploadableBinary:
		bin := artifact.MustExtra[string](*art, artifact.ExtraBinary)
		return fmt.Sprintf("install -Dm755 %q %q", "$srcdir/$_source", "$pkgdir/usr/bin/"+bin)
	case artifact.UploadableArchive:
		folder := artifact.ExtraOr(*art, artifact.ExtraWrappedIn, ".")
		var commands []string
		for _, bin := range artifact.MustExtra[[]string](*art, artifact.ExtraBinaries) {
			src := filepath.ToSlash(filepath.Clean(filepath.Join("$srcdir", folder, bin)))
			commands = append(commands, fmt.Sprintf("install -Dm755 %q %q", src, "$pkgdir/usr/bin/"+bin))
		}
		return strings.Join(commands, "\n")
	default:
		return ""
	}
}

func buildAPKBuild(ctx *context.Context, pkg config.APKBuild, cli client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, pkg, cli, artifacts)
	if err != nil {
		return "", err
	}
	return applyTemplate(ctx, apkbuildTemplate, data)
}

func applyTemplate(ctx *context.Context, source string, data templateData) (string, error) {
	tpl := template.Must(template.New(data.Name).Funcs(template.FuncMap{
		"fixLines":   fixLines,
		"shellJoin":  shellJoin,
		"shellQuote": shellQuote,
	}).Parse(source))

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", err
	}
	content, err := tmpl.New(ctx).Apply(out.String())
	if err != nil {
		return "", err
	}

	out.Reset()
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		_, _ = out.WriteString(strings.TrimRight(scanner.Text(), " "))
		_ = out.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func fixLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		lines[i] = line
		if line != "" {
			lines[i] = "\t" + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func shellJoin(values []string) string {
	return shellQuote(strings.Join(values, " "))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func toAPKArch(goarch, goarm string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "386":
		return "x86"
	case "arm64":
		return "aarch64"
	case "arm":
		if goarm == "" {
			goarm = experimental.DefaultGOARM()
		}
		switch goarm {
		case "6":
			return "armhf"
		case "7":
			return "armv7"
		}
	case "ppc64le", "s390x", "riscv64":
		return goarch
	}
	return ""
}

func dataFor(ctx *context.Context, cfg config.APKBuild, cli client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{
		Name:         cfg.Name,
		Description:  cfg.Description,
		Homepage:     cfg.Homepage,
		Version:      strings.ReplaceAll(ctx.Version, "-", "_"),
		License:      cfg.License,
		Maintainers:  cfg.Maintainers,
		Contributors: cfg.Contributors,
		Provides:     cfg.Provides,
		Depends:      cfg.Depends,
		MakeDepends:  cfg.MakeDepends,
		Replaces:     cfg.Replaces,
		Options:      cfg.Options,
		Rel:          cfg.Rel,
		Package:      cfg.Package,
	}

	urlTemplate := cfg.URLTemplate
	if urlTemplate == "" {
		url, err := cli.ReleaseURLTemplate(ctx)
		if err != nil {
			return result, err
		}
		urlTemplate = url
	}

	seen := make(map[string]struct{}, len(artifacts))
	for _, art := range artifacts {
		arch := toAPKArch(art.Goarch, art.Goarm)
		if arch == "" {
			continue
		}
		if _, ok := seen[arch]; ok {
			return result, fmt.Errorf("multiple artifacts found for Alpine architecture %s; use ids to select one", arch)
		}
		seen[arch] = struct{}{}

		sum, err := art.Checksum("sha512")
		if err != nil {
			return result, err
		}
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(urlTemplate)
		if err != nil {
			return result, err
		}

		sourceName := fmt.Sprintf("%s-%s-%s", cfg.Name, result.Version, arch)
		if format := art.Format(); format != "" && format != "binary" {
			sourceName += "." + strings.TrimPrefix(format, ".")
		}
		result.Arches = append(result.Arches, arch)
		result.ReleasePackages = append(result.ReleasePackages, releasePackage{
			DownloadURL: url,
			SHA512:      sum,
			Arch:        arch,
			SourceName:  sourceName,
		})
	}

	slices.Sort(result.Arches)
	slices.SortFunc(result.ReleasePackages, func(a, b releasePackage) int {
		return cmp.Compare(a.Arch, b.Arch)
	})
	return result, nil
}

// Publish commits the generated APKBUILD files to their configured repositories.
func (Pipe) Publish(ctx *context.Context) error {
	skipped := pipe.SkipMemento{}
	for _, files := range ctx.Artifacts.Filter(artifact.ByType(artifact.APKBuild)).GroupByID() {
		err := doPublish(ctx, files)
		if err != nil && pipe.IsSkip(err) {
			skipped.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skipped.Evaluate()
}

func doPublish(ctx *context.Context, files []*artifact.Artifact) error {
	cfg := artifact.MustExtra[config.APKBuild](*files[0], apkbuildExtra)
	if strings.TrimSpace(cfg.SkipUpload) == "true" {
		return pipe.Skip("apkbuild.skip_upload is set")
	}
	if strings.TrimSpace(cfg.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping apkbuild publish")
	}

	author, err := commitauthor.Get(ctx, cfg.CommitAuthor)
	if err != nil {
		return err
	}
	message, err := tmpl.New(ctx).Apply(cfg.CommitMessageTemplate)
	if err != nil {
		return err
	}

	uploader := client.NewGitUploadClient("master")
	repo := client.RepoFromRef(config.RepoRef{
		Git: config.GitRepoRef{
			PrivateKey: cfg.PrivateKey,
			URL:        cfg.GitURL,
			SSHCommand: cfg.GitSSHCommand,
		},
		Name: fmt.Sprintf("%x", sha256.Sum256([]byte(cfg.GitURL))),
	})

	repoFiles := make([]client.RepoFile, 0, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file.Path)
		if err != nil {
			return err
		}
		repoFiles = append(repoFiles, client.RepoFile{
			Path:    path.Join(cfg.Directory, file.Name),
			Content: content,
		})
	}
	return uploader.CreateFiles(ctx, author, repo, message, repoFiles)
}
