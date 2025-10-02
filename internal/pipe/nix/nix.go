// Package nix creates nix packages.
package nix

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	stdctx "context"

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

const nixConfigExtra = "NixConfig"

// ErrMultipleArchivesSamePlatform happens when the config yields multiple
// archives for the same platform.
var ErrMultipleArchivesSamePlatform = errors.New("one nixpkg can handle only one archive of each OS/Arch combination")

type errNoArchivesFound struct {
	goamd64 string
	ids     []string
}

func (e errNoArchivesFound) Error() string {
	return fmt.Sprintf("no archives found matching goos=[darwin linux] goarch=[amd64 arm arm64 386] goarm=[6 7] goamd64=%s ids=%v", e.goamd64, e.ids)
}

var (
	errNoRepoName     = pipe.Skip("repository name is not set")
	errSkipUpload     = pipe.Skip("nix.skip_upload is set")
	errSkipUploadAuto = pipe.Skip("nix.skip_upload is set to 'auto', and current version is a pre-release")
	errInvalidLicense = errors.New("nix.license is invalid")
)

// New returns a pipe to be used in the publish phase.
func New() Pipe {
	return Pipe{realHasher}
}

type Pipe struct {
	hasher fileHasher
}

func (Pipe) String() string                           { return "nixpkgs" }
func (Pipe) ContinueOnError() bool                    { return true }
func (Pipe) Dependencies(_ *context.Context) []string { return []string{nixHashBin} }

func (p Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Nix) || len(ctx.Config.Nix) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Nix {
		nix := &ctx.Config.Nix[i]

		nix.CommitAuthor = commitauthor.Default(nix.CommitAuthor)

		if nix.CommitMessageTemplate == "" {
			nix.CommitMessageTemplate = "{{ .ProjectName }}: {{ .PreviousTag }} -> {{ .Tag }}"
		}
		if nix.Name == "" {
			nix.Name = ctx.Config.ProjectName
		}
		if nix.Goamd64 == "" {
			nix.Goamd64 = "v1"
		}
		if nix.License != "" && !slices.Contains(validLicenses, nix.License) {
			return fmt.Errorf("%w: %s", errInvalidLicense, nix.License)
		}
	}

	return nil
}

func (p Pipe) Run(ctx *context.Context) error {
	if !p.hasher.Available() {
		return pipe.Skipf("%s is not available", nixHashBin)
	}
	cli, err := client.NewReleaseClient(ctx)
	if err != nil {
		return err
	}

	return p.runAll(ctx, cli)
}

// Publish .
func (p Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return p.publishAll(ctx, cli)
}

func (p Pipe) runAll(ctx *context.Context, cli client.ReleaseURLTemplater) error {
	for _, nix := range ctx.Config.Nix {
		err := p.doRun(ctx, nix, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Pipe) publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, nix := range ctx.Artifacts.Filter(artifact.ByType(artifact.Nixpkg)).List() {
		err := doPublish(ctx, p.hasher, cli, nix)
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

func (p Pipe) doRun(ctx *context.Context, nix config.Nix, cl client.ReleaseURLTemplater) error {
	if nix.Repository.Name == "" {
		return errNoRepoName
	}

	tp := tmpl.New(ctx)

	err := tp.ApplyAll(
		&nix.Name,
		&nix.SkipUpload,
		&nix.Homepage,
		&nix.Description,
		&nix.Path,
	)
	if err != nil {
		return err
	}

	nix.Repository, err = client.TemplateRef(tmpl.New(ctx).Apply, nix.Repository)
	if err != nil {
		return err
	}

	if nix.Path == "" {
		nix.Path = path.Join("pkgs", nix.Name, "default.nix")
	}

	path := filepath.Join(ctx.Config.Dist, "nix", nix.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := preparePkg(ctx, nix, cl, p.hasher)
	if err != nil {
		return err
	}

	log.WithField("nixpkg", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write nixpkg: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filepath.Base(path),
		Path: path,
		Type: artifact.Nixpkg,
		Extra: map[string]any{
			nixConfigExtra: nix,
		},
	})

	return nil
}

func preparePkg(
	ctx *context.Context,
	nix config.Nix,
	cli client.ReleaseURLTemplater,
	hasher fileHasher,
) (string, error) {
	filters := []artifact.Filter{
		artifact.ByGooses("darwin", "linux"),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(nix.Goamd64),
			),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.Or(
					artifact.ByGoarm("6"),
					artifact.ByGoarm("7"),
				),
			),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("386"),
			artifact.ByGoarch("all"),
		),
		artifact.And(
			artifact.Not(artifact.ByFormats("gz")),
			artifact.ByType(artifact.UploadableArchive),
		),
		artifact.OnlyReplacingUnibins,
	}
	if len(nix.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(nix.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return "", errNoArchivesFound{
			goamd64: nix.Goamd64,
			ids:     nix.IDs,
		}
	}

	if nix.URLTemplate == "" {
		url, err := cli.ReleaseURLTemplate(ctx)
		if err != nil {
			return "", err
		}
		nix.URLTemplate = url
	}

	installs, err := installs(ctx, nix, archives[0])
	if err != nil {
		return "", err
	}

	postInstall, err := postInstall(ctx, nix, archives[0])
	if err != nil {
		return "", err
	}

	inputs := []string{"installShellFiles"}
	dependencies := depNames(nix.Dependencies)
	if len(dependencies) > 0 {
		inputs = append(inputs, "makeWrapper")
		dependencies = append(dependencies, "makeWrapper")
	}
	for _, arch := range archives {
		if arch.Format() == "zip" {
			inputs = append(inputs, "unzip")
			dependencies = append(dependencies, "unzip")
			break
		}
	}

	data := templateData{
		Name:         nix.Name,
		Version:      ctx.Version,
		Install:      installs,
		PostInstall:  postInstall,
		Archives:     map[string]Archive{},
		SourceRoots:  map[string]string{},
		Description:  nix.Description,
		Homepage:     nix.Homepage,
		License:      nix.License,
		Inputs:       inputs,
		Dependencies: dependencies,
	}

	platforms := map[string]bool{}
	for _, art := range archives {
		sha, err := hasher.Hash(ctx, art.Path)
		if err != nil {
			return "", err
		}
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(nix.URLTemplate)
		if err != nil {
			return "", err
		}
		archive := Archive{
			URL: url,
			Sha: sha,
		}

		for _, goarch := range expandGoarch(art.Goarch) {
			key := art.Goos + goarch + art.Goarm
			if _, ok := data.Archives[key]; ok {
				return "", ErrMultipleArchivesSamePlatform
			}
			folder := cmp.Or(artifact.ExtraOr(*art, artifact.ExtraWrappedIn, ""), ".")
			data.SourceRoots[key] = folder
			data.Archives[key] = archive
			plat := goosToPlatform[art.Goos+goarch+art.Goarm]
			if plat == "" {
				return "", errors.New("invalid platform: " + art.Goos + goarch + art.Goarm)
			}
			platforms[plat] = true
		}
	}

	if roots := slices.Compact(slices.Collect(maps.Values(data.SourceRoots))); len(roots) == 1 {
		data.SourceRoot = roots[0]
	}
	data.Platforms = slices.Sorted(maps.Keys(platforms))

	return doBuildPkg(ctx, data)
}

func expandGoarch(goarch string) []string {
	if goarch == "all" {
		return []string{"amd64", "arm64"}
	}
	return []string{goarch}
}

var goosToPlatform = map[string]string{
	"linuxamd64":  "x86_64-linux",
	"linuxarm64":  "aarch64-linux",
	"linuxarm":    "armv" + experimental.DefaultGOARM() + "l-linux",
	"linuxarm6":   "armv6l-linux",
	"linuxarm7":   "armv7l-linux",
	"linux386":    "i686-linux",
	"darwinamd64": "x86_64-darwin",
	"darwinarm64": "aarch64-darwin",
}

func doPublish(ctx *context.Context, hasher fileHasher, cl client.Client, pkg *artifact.Artifact) error {
	nix := artifact.MustExtra[config.Nix](*pkg, nixConfigExtra)
	if strings.TrimSpace(nix.SkipUpload) == "true" {
		return errSkipUpload
	}

	if strings.TrimSpace(nix.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return errSkipUploadAuto
	}

	repo := client.RepoFromRef(nix.Repository)

	gpath := nix.Path

	msg, err := tmpl.New(ctx).Apply(nix.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, nix.CommitAuthor)
	if err != nil {
		return err
	}

	content, err := preparePkg(ctx, nix, cl, hasher)
	if err != nil {
		return err
	}

	if nix.Repository.Git.URL != "" {
		return client.NewGitUploadClient(repo.Branch).
			CreateFile(ctx, author, repo, []byte(content), gpath, msg)
	}

	cl, err = client.NewIfToken(ctx, cl, nix.Repository.Token)
	if err != nil {
		return err
	}

	base := client.Repo{
		Name:   nix.Repository.PullRequest.Base.Name,
		Owner:  nix.Repository.PullRequest.Base.Owner,
		Branch: nix.Repository.PullRequest.Base.Branch,
	}

	// try to sync branch
	fscli, ok := cl.(client.ForkSyncer)
	if ok && nix.Repository.PullRequest.Enabled {
		if err := fscli.SyncFork(ctx, repo, base); err != nil {
			log.WithError(err).Warn("could not sync fork")
		}
	}

	if err := cl.CreateFile(ctx, author, repo, []byte(content), gpath, msg); err != nil {
		return err
	}

	if !nix.Repository.PullRequest.Enabled {
		log.Debug("nix.pull_request disabled")
		return nil
	}

	log.Info("nix.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return errors.New("client does not support pull requests")
	}

	return pcl.OpenPullRequest(ctx, base, repo, msg, nix.Repository.PullRequest.Draft)
}

func doBuildPkg(ctx *context.Context, data templateData) (string, error) {
	t, err := template.
		New(data.Name).
		Parse(string(pkgTmpl))
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

func postInstall(ctx *context.Context, nix config.Nix, art *artifact.Artifact) ([]string, error) {
	applied, err := tmpl.New(ctx).WithArtifact(art).Apply(nix.PostInstall)
	if err != nil {
		return nil, err
	}
	return split(applied), nil
}

func installs(ctx *context.Context, nix config.Nix, art *artifact.Artifact) ([]string, error) {
	tpl := tmpl.New(ctx).WithArtifact(art)

	extraInstall, err := tpl.Apply(nix.ExtraInstall)
	if err != nil {
		return nil, err
	}

	install, err := tpl.Apply(nix.Install)
	if err != nil {
		return nil, err
	}
	if install != "" {
		return append(split(install), split(extraInstall)...), nil
	}

	result := []string{"mkdir -p $out/bin"}
	binInstallFormat := binInstallFormats(nix)
	for _, bin := range artifact.MustExtra[[]string](*art, artifact.ExtraBinaries) {
		for _, format := range binInstallFormat {
			result = append(result, fmt.Sprintf(format, bin))
		}
	}

	log.WithField("install", result).Info("guessing install")

	return append(result, split(extraInstall)...), nil
}

func binInstallFormats(nix config.Nix) []string {
	formats := []string{"cp -vr ./%[1]s $out/bin/%[1]s"}
	if len(nix.Dependencies) == 0 {
		return formats
	}
	var deps, linuxDeps, darwinDeps []string

	for _, dep := range nix.Dependencies {
		switch dep.OS {
		case "darwin":
			darwinDeps = append(darwinDeps, dep.Name)
		case "linux":
			linuxDeps = append(linuxDeps, dep.Name)
		default:
			deps = append(deps, dep.Name)
		}
	}

	var depStrings []string

	if len(darwinDeps) > 0 {
		depStrings = append(depStrings, fmt.Sprintf("lib.optionals stdenvNoCC.isDarwin [ %s ]", strings.Join(darwinDeps, " ")))
	}
	if len(linuxDeps) > 0 {
		depStrings = append(depStrings, fmt.Sprintf("lib.optionals stdenvNoCC.isLinux [ %s ]", strings.Join(linuxDeps, " ")))
	}
	if len(deps) > 0 {
		depStrings = append(depStrings, fmt.Sprintf("[ %s ]", strings.Join(deps, " ")))
	}

	depString := strings.Join(depStrings, " ++ ")
	return append(
		formats,
		"wrapProgram $out/bin/%[1]s --prefix PATH : ${lib.makeBinPath ("+depString+")}",
	)
}

func split(s string) []string {
	var result []string
	for line := range strings.SplitSeq(strings.TrimSpace(s), "\n") {
		line := strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func depNames(deps []config.NixDependency) []string {
	var result []string
	for _, dep := range deps {
		result = append(result, dep.Name)
	}
	return result
}

type fileHasher interface {
	Hash(ctx stdctx.Context, name string) (string, error)
	Available() bool
}

const nixHashBin = "nix-hash"

var realHasher fileHasher = nixHasher{bin: nixHashBin}

type nixHasher struct{ bin string }

func (p nixHasher) Available() bool {
	_, err := exec.LookPath(p.bin)
	return err == nil
}

func (p nixHasher) Hash(ctx stdctx.Context, name string) (string, error) {
	// $ nix-hash --type sha256 --flat --base32 <(echo test)
	out, err := exec.CommandContext(
		ctx,
		p.bin,
		"--type", "sha256",
		"--flat",
		"--base32",
		name,
	).Output()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("could not hash file: %s: %w: %s", name, err, outStr)
	}
	return outStr, nil
}
