package nix

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const nixConfigExtra = "NixConfig"

// ErrNoArchivesFound happens when 0 archives are found.
type ErrNoArchivesFound struct {
	goamd64 string
	ids     []string
}

func (e ErrNoArchivesFound) Error() string {
	return fmt.Sprintf("no linux/macos archives found matching goos=[darwin linux] goarch=[amd64 arm64] goamd64=%s ids=%v", e.goamd64, e.ids)
}

// NewBuild returns a pipe to be used in the build phase.
func NewBuild() Pipe {
	return Pipe{buildShaPrefetcher{}}
}

// NewPublish returns a pipe to be used in the publish phase.
func NewPublish() Pipe {
	return Pipe{prodShaPrefetcher{}}
}

type Pipe struct {
	prefetcher shaPrefetcher
}

func (Pipe) String() string                 { return "nixpkgs" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Brews) == 0 }

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
		if nix.Install == "" {
			nix.Install = `
			    mkdir -p $out/bin
				cp -vr ./{{.Binary}} $out/bin/{{.Binary}}
			 `
		}
	}

	return nil
}

func (p Pipe) Run(ctx *context.Context) error {
	cli, err := client.New(ctx)
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

func (p Pipe) runAll(ctx *context.Context, cli client.Client) error {
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
		err := doPublish(ctx, p.prefetcher, cli, nix)
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

func (p Pipe) doRun(ctx *context.Context, nix config.Nix, cl client.ReleaserURLTemplater) error {
	if nix.Repository.Name == "" {
		return pipe.Skip("derivation name is not set")
	}

	name, err := tmpl.New(ctx).Apply(nix.Name)
	if err != nil {
		return err
	}
	nix.Name = name

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, nix.Repository)
	if err != nil {
		return err
	}
	nix.Repository = ref

	skipUpload, err := tmpl.New(ctx).Apply(nix.SkipUpload)
	if err != nil {
		return err
	}
	nix.SkipUpload = skipUpload

	filename := nix.Name + ".nix"
	path := filepath.Join(ctx.Config.Dist, filename)

	content, err := preparePkg(ctx, nix, cl, p.prefetcher)
	if err != nil {
		return err
	}

	log.WithField("nixpkg", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write nixpkg: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.Nixpkg,
		Extra: map[string]interface{}{
			nixConfigExtra: nix,
		},
	})

	return nil
}

func preparePkg(
	ctx *context.Context,
	nix config.Nix,
	cli client.ReleaserURLTemplater,
	prefetcher shaPrefetcher,
) (string, error) {
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
		),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(nix.Goamd64),
			),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("386"),
			artifact.ByGoarch("all"),
		),
		artifact.And(
			artifact.ByFormats("zip", "tar.gz"),
			artifact.ByType(artifact.UploadableArchive),
		),
		artifact.OnlyReplacingUnibins,
	}
	if len(nix.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(nix.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return "", ErrNoArchivesFound{
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

	data := templateData{
		Name:        nix.Name,
		Version:     ctx.Version,
		Install:     installs,
		Archives:    map[string]Archive{},
		SourceRoot:  ".",
		Description: nix.Description,
		Homepage:    nix.Homepage,
		License:     nix.License,
	}
	platforms := map[string]bool{}
	for _, art := range archives {
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(nix.URLTemplate)
		if err != nil {
			return "", err
		}
		sha, err := prefetcher.Prefetch(url)
		if err != nil {
			return "", err
		}
		archive := Archive{
			URL: url,
			Sha: sha,
		}

		for _, goarch := range expandGoarch(art.Goarch) {
			data.Archives[art.Goos+goarch] = archive
			plat, ok := goosToPlatform[art.Goos+goarch]
			if !ok {
				return "", fmt.Errorf("invalid platform: %s/%s", art.Goos, goarch)
			}
			platforms[plat] = true
		}
	}
	data.Platforms = keys(platforms)
	sort.Strings(data.Platforms)

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
	"linux386":    "i686-linux",
	"darwinamd64": "x86_64-darwin",
	"darwinarm64": "aarch64-darwin",
}

func keys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func doPublish(ctx *context.Context, prefetcher shaPrefetcher, cl client.Client, pkg *artifact.Artifact) error {
	nix, err := artifact.Extra[config.Nix](*pkg, nixConfigExtra)
	if err != nil {
		return err
	}

	if strings.TrimSpace(nix.SkipUpload) == "true" {
		return pipe.Skip("brew.skip_upload is set")
	}

	if strings.TrimSpace(nix.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping nixpkg publish")
	}

	repo := client.RepoFromRef(nix.Repository)

	gpath := path.Join("pkgs", nix.Name, "default.nix")
	log.WithField("path", gpath).
		WithField("repo", repo.String()).
		Info("pushing")

	msg, err := tmpl.New(ctx).Apply(nix.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, nix.CommitAuthor)
	if err != nil {
		return err
	}

	content, err := preparePkg(ctx, nix, cl, prefetcher)
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

	if !nix.Repository.PullRequest.Enabled {
		return cl.CreateFile(ctx, author, repo, []byte(content), gpath, msg)
	}

	log.Info("brews.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	if err := cl.CreateFile(ctx, author, repo, []byte(content), gpath, msg); err != nil {
		return err
	}

	title := fmt.Sprintf("Updated %s to %s", ctx.Config.ProjectName, ctx.Version)
	return pcl.OpenPullRequest(ctx, repo, nix.Repository.PullRequest.Base, title)
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

func installs(ctx *context.Context, nix config.Nix, art *artifact.Artifact) ([]string, error) {
	applied, err := tmpl.New(ctx).WithArtifact(art).Apply(nix.Install)
	if err != nil {
		return nil, err
	}
	if applied != "" {
		return split(applied), nil
	}

	result := []string{"mkdir -p $out/bin"}
	switch art.Type {
	case artifact.UploadableBinary:
		name := art.Name
		bin := artifact.ExtraOr(*art, artifact.ExtraBinary, art.Name)
		result = append(result, fmt.Sprintf("cp -vr ./%s $out/bin/%s", name, bin))
	case artifact.UploadableArchive:
		for _, bin := range artifact.ExtraOr(*art, artifact.ExtraBinaries, []string{}) {
			result = append(result, fmt.Sprintf("cp -vr ./%s $out/bin/%[1]s", bin))
		}
	}

	log.WithField("install", result).Warnf("guessing install")
	return result, nil
}

func split(s string) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line := strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

type shaPrefetcher interface {
	Prefetch(url string) (string, error)
}

const zeroHash = "0000000000000000000000000000000000000000000000000000"

type buildShaPrefetcher struct{}

func (buildShaPrefetcher) Prefetch(_ string) (string, error) { return zeroHash, nil }

type prodShaPrefetcher struct{}

func (prodShaPrefetcher) Prefetch(url string) (string, error) {
	out, err := exec.Command("nix-prefetch-url", url).Output()
	return strings.TrimSpace(string(out)), err
}
