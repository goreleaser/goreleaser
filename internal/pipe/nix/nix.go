package nix

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

type Pipe struct {
	prefetcher ShaPrefetcher
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
			nix.Install = fmt.Sprintf(`
			    mkdir -p $out/bin
				cp -vr ./%s $out/bin/%[1]s
			 `, nix.Name)
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
func (Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
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

func publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, nix := range ctx.Artifacts.Filter(artifact.ByType(artifact.Nixpkg)).List() {
		err := doPublish(ctx, nix, cli)
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
	if nix.Tap.Name == "" {
		return pipe.Skip("derivation name is not set")
	}

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
		),
		artifact.And(
			artifact.ByFormats("zip", "tar.gz"),
			artifact.ByType(artifact.UploadableArchive),
		),
	}
	if len(nix.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(nix.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound{
			goamd64: nix.Goamd64,
			ids:     nix.IDs,
		}
	}

	name, err := tmpl.New(ctx).Apply(nix.Name)
	if err != nil {
		return err
	}
	nix.Name = name

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, nix.Tap)
	if err != nil {
		return err
	}
	nix.Tap = ref

	skipUpload, err := tmpl.New(ctx).Apply(nix.SkipUpload)
	if err != nil {
		return err
	}
	nix.SkipUpload = skipUpload

	content, err := p.buildDerivation(ctx, nix, cl, archives)
	if err != nil {
		return err
	}

	filename := nix.Name + ".nix"
	path := filepath.Join(ctx.Config.Dist, filename)
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

func (p Pipe) buildDerivation(ctx *context.Context, nix config.Nix, cli client.ReleaserURLTemplater, artifacts []*artifact.Artifact) (string, error) {
	if nix.URLTemplate == "" {
		url, err := cli.ReleaseURLTemplate(ctx)
		if err != nil {
			return "", err
		}
		nix.URLTemplate = url
	}

	data := TemplateData{
		Name:       nix.Name,
		Version:    ctx.Version,
		Install:    nix.Install,
		Archives:   map[string]Archive{},
		SourceRoot: ".",
	}
	for _, art := range artifacts {
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(nix.URLTemplate)
		if err != nil {
			return "", err
		}
		sha, err := p.prefetcher.Prefetch(url)
		if err != nil {
			return "", err
		}
		data.Archives[art.Goos+art.Goarch] = Archive{
			URL: url,
			Sha: sha,
		}

	}

	return doBuildPkg(ctx, data)
}

func doPublish(ctx *context.Context, pkg *artifact.Artifact, cl client.Client) error {
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

	repo := client.RepoFromRef(nix.Tap)

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

	content, err := os.ReadFile(pkg.Path)
	if err != nil {
		return err
	}

	if nix.Tap.Git.URL != "" {
		return client.NewGitUploadClient(repo.Branch).
			CreateFile(ctx, author, repo, content, gpath, msg)
	}

	cl, err = client.NewIfToken(ctx, cl, nix.Tap.Token)
	if err != nil {
		return err
	}

	if !nix.Tap.PullRequest.Enabled {
		return cl.CreateFile(ctx, author, repo, content, gpath, msg)
	}

	log.Info("brews.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	if err := cl.CreateFile(ctx, author, repo, content, gpath, msg); err != nil {
		return err
	}

	title := fmt.Sprintf("Updated %s to %s", ctx.Config.ProjectName, ctx.Version)
	return pcl.OpenPullRequest(ctx, repo, nix.Tap.PullRequest.Base, title)
}

func doBuildPkg(ctx *context.Context, data TemplateData) (string, error) {
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

type ShaPrefetcher interface {
	Prefetch(url string) (string, error)
}

// ProductionShaPrefetcher uses `nix-prefetch-url` to prefetch hashes.
type ProductionShaPrefetcher struct{}

// Prefetch implements NixShaPrefetcher
func (ProductionShaPrefetcher) Prefetch(url string) (string, error) {
	out, err := exec.Command("nix-prefetch-url", url).Output()
	return strings.TrimSpace(string(out)), err
}

var _ ShaPrefetcher = ProductionShaPrefetcher{}
