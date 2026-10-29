package apkbuild

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/commitauthor"
	"github.com/goreleaser/goreleaser/v2/internal/experimental"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/summary"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	apkbuildExtra    = "APKBuildConfig"
	defaultCommitMsg = "Update to {{ .Tag }}"
)

var ErrNoArchivesFound = errors.New("no linux binaries or archives found")

var supportedArchiveFormats = []string{"tar.gz", "tgz", "tar.xz", "tar", "zip"}

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
		if !slices.Contains(pkg.Options, "!check") {
			pkg.Options = append(pkg.Options, "!check")
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
	skipped := pipe.SkipMemento{}
	generated := map[string]int{}
	for _, apk := range ctx.Config.APKBuilds {
		disable, err := tmpl.New(ctx).Bool(apk.Disable)
		if err != nil {
			return err
		}
		if disable {
			skipped.Remember(pipe.Skip("configuration is disabled"))
			continue
		}
		if err := doRun(ctx, apk, cli, generated); err != nil {
			return err
		}
	}
	return skipped.Evaluate()
}

func doRun(ctx *context.Context, apk config.APKBuild, cli client.ReleaseURLTemplater, generated map[string]int) error {
	if err := tmpl.New(ctx).ApplyAll(
		&apk.Name,
		&apk.Directory,
		&apk.SkipUpload,
		&apk.Description,
		&apk.Homepage,
		&apk.License,
		&apk.Rel,
	); err != nil {
		return err
	}
	for _, values := range []*[]string{
		&apk.Maintainers,
		&apk.Contributors,
		&apk.Provides,
		&apk.Depends,
		&apk.MakeDepends,
		&apk.Replaces,
		&apk.Options,
	} {
		if err := tmpl.New(ctx).ApplySlice(values); err != nil {
			return err
		}
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
		artifact.Or(
			artifact.ByType(artifact.UploadableBinary),
			artifact.And(
				artifact.ByType(artifact.UploadableArchive),
				artifact.ByFormats(supportedArchiveFormats...),
			),
		),
	}
	if len(apk.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(apk.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}
	if err := validate(apk); err != nil {
		return err
	}
	slices.SortFunc(archives, func(a, b *artifact.Artifact) int {
		return cmp.Or(
			cmp.Compare(a.Goarch, b.Goarch),
			cmp.Compare(a.Goarm, b.Goarm),
			cmp.Compare(a.Name, b.Name),
		)
	})
	archives = selectArtifacts(archives)

	pkg, err := tmpl.New(ctx).Apply(apk.Package)
	if err != nil {
		return err
	}
	apk.Package = strings.Trim(pkg, "\r\n")
	if strings.TrimSpace(apk.Package) == "" {
		apk.Package = ""
		log.Warn("guessing package instructions")
	}

	content, err := buildAPKBuild(ctx, apk, cli, archives)
	if err != nil {
		return err
	}

	occurrence := generated[apk.Name]
	generated[apk.Name]++
	id := apk.Name
	file := filepath.Join(ctx.Config.Dist, "apkbuild", apk.Name+".apkbuild")
	if occurrence > 0 {
		id = fmt.Sprintf("%s-%d", apk.Name, occurrence+1)
		file = filepath.Join(ctx.Config.Dist, "apkbuild", strconv.Itoa(occurrence+1), apk.Name+".apkbuild")
	}
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
			artifact.ExtraID: id,
		},
	})
	return nil
}

func selectArtifacts(artifacts []*artifact.Artifact) []*artifact.Artifact {
	type key struct{ id, arch string }
	best := make(map[key]*artifact.Artifact, len(artifacts))
	for _, art := range artifacts {
		if art.Type != artifact.UploadableArchive {
			continue
		}
		k := key{art.ID(), toAPKArch(art.Goarch, art.Goarm)}
		if current, ok := best[k]; !ok || formatPriority(art.Format()) < formatPriority(current.Format()) {
			best[k] = art
		}
	}

	result := make([]*artifact.Artifact, 0, len(artifacts))
	for _, art := range artifacts {
		archive, ok := best[key{art.ID(), toAPKArch(art.Goarch, art.Goarm)}]
		if ok && (art.Type != artifact.UploadableArchive || art.Format() != archive.Format()) {
			continue
		}
		result = append(result, art)
	}
	return result
}

func formatPriority(format string) int {
	return slices.Index(supportedArchiveFormats, format)
}

func defaultPackage(art *artifact.Artifact) (string, error) {
	switch art.Type {
	case artifact.UploadableBinary:
		bin := artifact.MustExtra[string](*art, artifact.ExtraBinary)
		return fmt.Sprintf("install -Dm755 %q %q", "$srcdir/$_source", "$pkgdir/usr/bin/"+filepath.Base(bin)), nil
	case artifact.UploadableArchive:
		folder := artifact.ExtraOr(*art, artifact.ExtraWrappedIn, ".")
		var commands []string
		installed := map[string]string{}
		for _, bin := range artifact.MustExtra[[]string](*art, artifact.ExtraBinaries) {
			name := filepath.Base(bin)
			if previous, ok := installed[name]; ok {
				return "", fmt.Errorf("binaries %q and %q would both be installed as %q: set apkbuild.package to install them", previous, bin, name)
			}
			installed[name] = bin
			src := filepath.ToSlash(filepath.Clean(filepath.Join("$srcdir", folder, bin)))
			commands = append(commands, fmt.Sprintf("install -Dm755 %q %q", src, "$pkgdir/usr/bin/"+name))
		}
		return strings.Join(commands, "\n"), nil
	default:
		return "", nil
	}
}

func validate(cfg config.APKBuild) error {
	switch {
	case strings.TrimSpace(cfg.Description) == "":
		return errors.New("apkbuild.description is required")
	case strings.TrimSpace(cfg.Homepage) == "":
		return errors.New("apkbuild.homepage is required")
	case strings.TrimSpace(cfg.License) == "":
		return errors.New("apkbuild.license is required")
	}
	if len(cfg.Maintainers) > 1 {
		return errors.New("apkbuild.maintainers must contain at most one entry")
	}
	return nil
}

func buildAPKBuild(ctx *context.Context, pkg config.APKBuild, cli client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, pkg, cli, artifacts)
	if err != nil {
		return "", err
	}
	return applyTemplate(apkbuildTemplate, data)
}

func applyTemplate(source string, data templateData) (string, error) {
	tpl := template.Must(template.New(data.Name).Funcs(template.FuncMap{
		"fixLines":   fixLines,
		"shellJoin":  shellJoin,
		"shellQuote": shellQuote,
	}).Parse(source))

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func fixLines(s string, depth int) string {
	lines := strings.Split(s, "\n")
	indent := strings.Repeat("\t", depth)
	for i, line := range lines {
		line = strings.TrimSpace(line)
		lines[i] = line
		if line != "" {
			lines[i] = indent + lines[i]
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

func toAPKVersion(version string) (string, error) {
	version, snapshot, hasSnapshot := strings.Cut(version, "-SNAPSHOT-")
	version, metadata, hasMetadata := strings.Cut(version, "+")
	base, prerelease, hasPrerelease := strings.Cut(version, "-")
	result := base
	if hasPrerelease {
		suffix, err := toAPKPrerelease(prerelease)
		if err != nil {
			return "", err
		}
		result += suffix
	}
	if hasMetadata {
		result += "_p" + toAPKVersionNumber(metadata)
	}
	if hasSnapshot {
		result += "_git" + toAPKVersionNumber("snapshot-"+strings.ToLower(snapshot))
	}
	return result, nil
}

func toAPKPrerelease(prerelease string) (string, error) {
	value := strings.ToLower(prerelease)
	label := value[:len(value)-len(strings.TrimLeft(value, "abcdefghijklmnopqrstuvwxyz"))]
	rest := value[len(label):]

	switch label {
	case "alpha", "beta", "pre", "rc":
	default:
		return "", fmt.Errorf("prerelease %q cannot be converted to an ordered apk version: supported labels are alpha, beta, pre, and rc", prerelease)
	}

	if strings.Trim(rest, "0123456789") != "" || (len(rest) > 1 && rest[0] == '0') {
		return "", fmt.Errorf("prerelease %q cannot be converted to an ordered apk version: use only digits without leading zeros after the label, e.g. %q", prerelease, "rc1")
	}
	return "_" + label + rest, nil
}

func toAPKVersionNumber(value string) string {
	sum := sha256.Sum256([]byte(value))
	return strconv.FormatUint(binary.BigEndian.Uint64(sum[:]), 10)
}

func dataFor(ctx *context.Context, cfg config.APKBuild, cli client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (templateData, error) {
	version, err := toAPKVersion(ctx.Version)
	if err != nil {
		return templateData{}, err
	}
	result := templateData{
		Name:         cfg.Name,
		Description:  cfg.Description,
		Homepage:     cfg.Homepage,
		Version:      version,
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
		packageCommand := ""
		if cfg.Package == "" {
			packageCommand, err = defaultPackage(art)
			if err != nil {
				return result, err
			}
		}
		result.Arches = append(result.Arches, arch)
		result.ReleasePackages = append(result.ReleasePackages, releasePackage{
			DownloadURL: url,
			SHA512:      sum,
			Arch:        arch,
			SourceName:  sourceName,
			Package:     packageCommand,
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
	for _, file := range ctx.Artifacts.Filter(artifact.ByType(artifact.APKBuild)).List() {
		err := doPublish(ctx, file)
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

func doPublish(ctx *context.Context, file *artifact.Artifact) error {
	cfg := artifact.MustExtra[config.APKBuild](*file, apkbuildExtra)
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

	uploader := client.NewGitUploadClient("")
	repo := client.RepoFromRef(config.RepoRef{
		Git: config.GitRepoRef{
			PrivateKey: cfg.PrivateKey,
			URL:        cfg.GitURL,
			SSHCommand: cfg.GitSSHCommand,
		},
		Name: fmt.Sprintf("%x", sha256.Sum256([]byte(cfg.GitURL))),
	})

	content, err := os.ReadFile(file.Path)
	if err != nil {
		return err
	}
	if err := uploader.CreateFiles(ctx, author, repo, message, []client.RepoFile{{
		Path:    path.Join(cfg.Directory, file.Name),
		Content: content,
	}}); err != nil {
		return err
	}
	summary.Appendf("Pushed Alpine Linux package `%s` to `%s`", cfg.Name, cfg.GitURL)
	return nil
}
