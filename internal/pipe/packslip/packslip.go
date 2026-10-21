// Package packslip creates signed packslip release manifests describing the
// release artifacts.
package packslip

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/elf"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	packslip "github.com/jdx/packslip/go"
)

// Pipe that creates the packslip manifests.
type Pipe struct{}

func (Pipe) String() string { return "creating packslip manifests" }

func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Packslip) || len(ctx.Config.Packslips) == 0
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	repo := repoProject(ctx)
	projects := map[string]bool{}
	for i := range ctx.Config.Packslips {
		cfg := &ctx.Config.Packslips[i]
		if cfg.Project == "" {
			cfg.Project = repo
		}
		if cfg.NameTemplate == "" {
			cfg.NameTemplate = defaultName(repo, cfg.Project)
		}
		if cfg.Project == "" {
			continue
		}
		if projects[cfg.Project] {
			return fmt.Errorf("found multiple packslips with the project '%s', each project needs its own entry", cfg.Project)
		}
		projects[cfg.Project] = true
	}
	return nil
}

// repoProject is the packslip project of the GitHub repository the release
// is published to, or "" when it is not published to GitHub.
func repoProject(ctx *context.Context) string {
	if gh := ctx.Config.Release.GitHub; gh.Owner != "" && gh.Name != "" {
		return "github.com/" + gh.Owner + "/" + gh.Name
	}
	return ""
}

// defaultName is the bundle name the packslip spec gives a project: a
// monorepo tool github.com/acme/repo/crates/cli is
// packslip.crates-cli.sigstore.json, and anything else is
// packslip.sigstore.json.
func defaultName(repo, project string) string {
	if sub, ok := strings.CutPrefix(project, repo+"/"); ok && repo != "" {
		return "packslip." + strings.ReplaceAll(sub, "/", "-") + ".sigstore.json"
	}
	return "packslip.sigstore.json"
}

// signFunc signs a statement keylessly. It is a variable so tests do not need
// to reach Sigstore.
var signFunc = func(ctx *context.Context, statement packslip.Statement, token string) ([]byte, error) {
	options, err := packslip.ProductionKeylessOptions(ctx, token)
	if err != nil {
		return nil, err
	}
	return packslip.SignKeyless(statement, options)
}

// manifest is a statement ready to be signed.
type manifest struct {
	name      string
	statement packslip.Statement
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	var configs []config.Packslip
	for _, cfg := range ctx.Config.Packslips {
		disable, err := tmpl.New(ctx).Bool(cfg.Disable)
		if err != nil {
			return err
		}
		if disable {
			continue
		}
		if cfg.Project == "" {
			return errors.New("packslips.project must be set when the release is not on GitHub")
		}
		configs = append(configs, cfg)
	}
	if len(configs) == 0 {
		return pipe.Skip("all packslips are disabled")
	}
	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}
	if skips.Any(ctx, skips.Publish) {
		// the manifests are signed into a public transparency log, so they
		// are only created for releases that are actually published.
		return pipe.ErrSkipPublishEnabled
	}

	token, err := packslip.AmbientIDToken(ctx)
	if err != nil {
		return err
	}
	identity, err := packslip.IdentityFromToken(token)
	if err != nil {
		return err
	}

	// build every statement before signing any, so a bad entry does not
	// leave the others recorded in the transparency log.
	var manifests []manifest
	names := map[string]bool{}
	for _, cfg := range configs {
		m, err := build(ctx, cfg, identity)
		if err != nil {
			return err
		}
		if m == nil {
			continue
		}
		if names[m.name] {
			return fmt.Errorf("found multiple packslips named '%s', set a distinct name_template for each", m.name)
		}
		names[m.name] = true
		manifests = append(manifests, *m)
	}
	if len(manifests) == 0 {
		return pipe.Skip("no archives or binaries to describe")
	}

	for _, m := range manifests {
		log.WithField("file", m.name).Info("signing packslip manifest")
		bundle, err := signFunc(ctx, m.statement, token)
		if err != nil {
			return fmt.Errorf("could not sign packslip manifest %s: %w", m.name, err)
		}
		file := filepath.Join(ctx.Config.Dist, m.name)
		if err := os.WriteFile(file, bundle, 0o644); err != nil {
			return err
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableFile,
			Name: m.name,
			Path: file,
			Extra: map[string]any{
				artifact.ExtraID:           "packslip",
				artifact.ExtraSkipChecksum: true,
			},
		})
	}
	return nil
}

// build creates the statement for one packslip entry. It returns nil when the
// entry selects no artifacts.
func build(ctx *context.Context, cfg config.Packslip, identity packslip.Identity) (*manifest, error) {
	filters := []artifact.Filter{
		artifact.ByTypes(artifact.UploadableArchive, artifact.UploadableBinary),
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}
	urlTemplate := cfg.URLTemplate
	if urlTemplate == "" {
		// the URLs point at the release, so only describe what it uploads.
		filters = append(filters, artifact.ByIDs(ctx.Config.Release.IDs...))
	}
	artifacts := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(artifacts) == 0 {
		log.WithField("project", cfg.Project).Warn("no archives or binaries to describe")
		return nil, nil
	}

	if urlTemplate == "" {
		cl, err := client.NewReleaseClient(ctx)
		if err != nil {
			return nil, err
		}
		if urlTemplate, err = cl.ReleaseURLTemplate(ctx); err != nil {
			return nil, fmt.Errorf("packslips.url_template: %w", err)
		}
	}

	inputs := make([]packslip.ArtifactInput, 0, len(artifacts))
	for _, a := range artifacts {
		url, err := tmpl.New(ctx).WithArtifact(a).Apply(urlTemplate)
		if err != nil {
			return nil, fmt.Errorf("packslips.url_template: %w", err)
		}
		described, err := describe(a, url)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, packslip.ArtifactInput{
			Path:     a.Path,
			Artifact: described,
		})
	}

	predicate := packslip.Predicate{
		Project:     cfg.Project,
		Version:     ctx.Version,
		PublishedAt: time.Now().UTC(),
		Identity:    identity,
	}
	if repo := sourceRepo(ctx.Git.URL); repo != "" {
		predicate.Source = &packslip.Source{
			Repo:   repo,
			Commit: ctx.Git.FullCommit,
			Tag:    ctx.Git.CurrentTag,
		}
	}
	if err := validateProjectIdentity(cfg.Project, identity); err != nil {
		return nil, err
	}

	statement, err := packslip.Build(predicate, inputs)
	if err != nil {
		return nil, fmt.Errorf("could not build packslip statement for %s: %w", cfg.Project, err)
	}
	name, err := tmpl.New(ctx).Apply(cfg.NameTemplate)
	if err != nil {
		return nil, fmt.Errorf("packslips.name_template: %w", err)
	}
	return &manifest{name: name, statement: statement}, nil
}

// describe maps a GoReleaser artifact to a packslip artifact.
func describe(a *artifact.Artifact, url string) (packslip.Artifact, error) {
	format := artifact.ExtraOr(*a, artifact.ExtraFormat, "")
	if format == "binary" {
		format = "raw"
	}
	format = normalizeFormat(format)
	out := packslip.Artifact{
		Name:    a.Name,
		OS:      a.Goos,
		Arch:    packslip.ArchFromGo(a.Goarch, a.Goarm),
		URL:     url,
		Format:  format,
		Variant: cpuVariant(a),
	}
	if a.Goos == "linux" {
		switch {
		case !artifact.ExtraOr(*a, artifact.ExtranDynLink, false):
			// statically linked: fits glibc and musl hosts alike.
			out.Requires = &packslip.Requires{Libs: &[]string{}}
		default:
			libc := artifact.ExtraOr(*a, artifact.ExtraLibc, "")
			if libc == "" && a.Type == artifact.UploadableBinary {
				libc = elf.Libc(a.Path)
			}
			if libc != "gnu" && libc != "musl" {
				return packslip.Artifact{}, fmt.Errorf("cannot determine libc for dynamically linked Linux artifact %q", a.Name)
			}
			out.Libc = libc
		}
	}

	switch a.Type {
	case artifact.UploadableBinary:
		bin := artifact.ExtraOr(*a, artifact.ExtraBinary, "")
		if bin != "" {
			out.Bin = []packslip.Bin{packslip.NewBareBin(a.Name, "raw", path.Base(bin))}
		}
	default:
		wrap := artifact.ExtraOr(*a, artifact.ExtraWrappedIn, "")
		for _, bin := range artifact.ExtraOr(*a, artifact.ExtraBinaries, []string{}) {
			if isBare(format) {
				out.Bin = append(out.Bin, packslip.NewBareBin(a.Name, format, path.Base(bin)))
				break
			}
			out.Bin = append(out.Bin, packslip.NewBin(path.Join(wrap, bin)))
		}
	}
	return out, nil
}

// normalizeFormat maps GoReleaser's archive aliases to the Packslip format
// vocabulary. Artifact names and download URLs intentionally remain untouched.
func normalizeFormat(format string) string {
	switch format {
	case "txz":
		return "tar.xz"
	case "tzst":
		return "tar.zst"
	default:
		return format
	}
}

func isBare(format string) bool {
	switch format {
	case "gz", "xz", "zst", "bz2":
		return true
	}
	return false
}

// cpuVariant returns the packslip variant for a build that targets a CPU
// level other than Go's default, so that builds of the same platform stay
// distinct. Default-level builds have no variant, so consumers that do not
// ask for one get the most compatible build.
func cpuVariant(a *artifact.Artifact) string {
	level, def := "", ""
	switch a.Goarch {
	case "amd64":
		level, def = a.Goamd64, "v1"
	case "386":
		level, def = a.Go386, "sse2"
	case "arm64":
		level, def = a.Goarm64, "v8.0"
	case "mips", "mipsle", "mips64", "mips64le":
		level, def = a.Gomips, "hardfloat"
	case "ppc64", "ppc64le":
		level, def = a.Goppc64, "power8"
	case "riscv64":
		level, def = a.Goriscv64, "rva20u64"
	case "arm":
		// The Go builder stores the float ABI separately from GOARM. ArchFromGo
		// keeps the GOARM version; only a non-default float option distinguishes.
		if artifact.ExtraOr(*a, artifact.ExtraABI, "") == "softfloat" && a.Goarm != "5" {
			return "softfloat"
		}
		return ""
	}
	if level == "" || level == def {
		return ""
	}
	// GOARM64 options are comma separated, which a variant cannot contain.
	return strings.ReplaceAll(level, ",", "-")
}

// sourceRepo normalizes a git remote URL, such as git@github.com:acme/tool.git,
// to https://github.com/acme/tool. It returns "" when the URL is not
// understood.
func sourceRepo(remote string) string {
	s := strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(remote), "/"), ".git")
	var scheme, host, repoPath string
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return ""
		}
		repoPath = u.Path
		switch u.Scheme {
		case "http", "https":
			host = u.Host
			scheme = u.Scheme
		case "ssh":
			host = u.Hostname()
			scheme = "https"
		default:
			return ""
		}
	} else {
		// scp-like syntax: [user@]host:owner/name
		var ok bool
		host, repoPath, ok = strings.Cut(s, ":")
		if !ok {
			return ""
		}
		if _, h, ok := strings.Cut(host, "@"); ok {
			host = h
		}
		scheme = "https"
	}
	repoPath = strings.Trim(repoPath, "/")
	if host == "" || !strings.Contains(repoPath, "/") {
		return ""
	}
	return scheme + "://" + host + "/" + repoPath
}

// validateProjectIdentity ensures a GitHub Actions keyless certificate is
// associated with the GitHub repository named by the manifest project. A
// workflow repository may publish a checkout from another source repository;
// that checkout remains in predicate.source and is intentionally independent.
func validateProjectIdentity(project string, identity packslip.Identity) error {
	repo := githubWorkflowRepo(identity.KeyID)
	if repo == "" || !strings.HasPrefix(project, "github.com/") {
		return nil
	}
	if project == repo || strings.HasPrefix(project, repo+"/") {
		return nil
	}
	return fmt.Errorf("packslips.project %q must match the GitHub Actions workflow repository %q", project, repo)
}

func githubWorkflowRepo(keyID string) string {
	u, err := url.Parse(keyID)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != ".github" || parts[3] != "workflows" {
		return ""
	}
	return "github.com/" + parts[0] + "/" + parts[1]
}
