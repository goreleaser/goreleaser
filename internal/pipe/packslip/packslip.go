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
			cfg.NameTemplate = packslip.BundleName(cfg.Project, repo)
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
		m, err := build(ctx, cfg, token, identity)
		if err != nil {
			return err
		}
		if m == nil {
			continue
		}
		if names[m.name] {
			return fmt.Errorf("found multiple packslips named '%s', set a distinct name_template for each", m.name)
		}
		if err := checkNoOverwrite(ctx, m.name); err != nil {
			return err
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

// checkNoOverwrite refuses a manifest name that matches an existing artifact,
// by upload name or by file. Writing the manifest would otherwise replace a
// file that was already hashed or signed.
func checkNoOverwrite(ctx *context.Context, name string) error {
	file := filepath.Join(ctx.Config.Dist, name)
	for _, a := range ctx.Artifacts.List() {
		if a.Name == name || samePath(a.Path, file) {
			return fmt.Errorf("packslips.name_template: '%s' would overwrite the %s artifact '%s', set a distinct name_template", name, a.Type, a.Name)
		}
	}
	return nil
}

// samePath reports whether two paths name the same file, including through
// relative paths, links and ../ segments.
func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil && absA == absB {
		return true
	}
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}

// build creates the statement for one packslip entry. It returns nil when the
// entry selects no artifacts.
func build(ctx *context.Context, cfg config.Packslip, token string, identity packslip.Identity) (*manifest, error) {
	filters := []artifact.Filter{
		artifact.ByTypes(artifact.UploadableArchive, artifact.UploadableBinary),
		artifact.Not(func(a *artifact.Artifact) bool {
			return artifact.ExtraOr(*a, artifact.ExtraSkipPackslip, false)
		}),
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}
	urlTemplate := cfg.URLTemplate
	if urlTemplate == "" {
		skipUpload, err := tmpl.New(ctx).Bool(ctx.Config.Release.SkipUpload)
		if err != nil {
			return nil, err
		}
		if skipUpload {
			return nil, fmt.Errorf("packslips.url_template must be set when release.skip_upload is set, as the release has no assets to link to")
		}
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
	if err := packslip.CheckProjectSigner(cfg.Project, token); err != nil {
		return nil, fmt.Errorf("packslips.project: %w", err)
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
	target := goTarget(a)
	out := packslip.Artifact{
		Name:    a.Name,
		OS:      a.Goos,
		Arch:    target.Arch(),
		URL:     url,
		Format:  format,
		Variant: target.Variant(),
	}
	if a.Goos == "linux" {
		switch {
		case !artifact.ExtraOr(*a, artifact.ExtranDynLink, false):
			// statically linked: fits glibc and musl hosts alike.
			out.Requires = &packslip.Requires{Libs: &[]string{}}
		default:
			libc := artifact.ExtraOr(*a, artifact.ExtraLibcFamily, "")
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
			out.Bin = []packslip.Bin{packslip.NewBareBin(a.Name, "raw", path.Base(slash(bin)))}
		}
	default:
		wrap := slash(artifact.ExtraOr(*a, artifact.ExtraWrappedIn, ""))
		bins := artifact.ExtraOr(
			*a,
			artifact.ExtraPackslipBins,
			artifact.ExtraOr(*a, artifact.ExtraBinaries, []string{}),
		)
		for _, bin := range bins {
			bin = slash(bin)
			if isBare(format) {
				out.Bin = append(out.Bin, packslip.NewBareBin(a.Name, format, path.Base(bin)))
				break
			}
			out.Bin = append(out.Bin, packslip.NewBin(path.Join(wrap, bin)))
		}
	}
	return out, nil
}

// slash converts backslashes to slashes, as the archive writer does for the
// name of every entry, so the signed path is the one in the archive.
func slash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
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

// goTarget is the Go settings of a build, as packslip reads them to pick the
// architecture and variant. The Go builder stores the float ABI separately
// from GOARM, as "softfloat" or "hardfloat". Rust and Zig keep the target's
// ABI, such as "gnueabi".
func goTarget(a *artifact.Artifact) packslip.GoTarget {
	return packslip.GoTarget{
		GOARCH:    a.Goarch,
		GOARM:     a.Goarm,
		GOAMD64:   a.Goamd64,
		GO386:     a.Go386,
		GOARM64:   a.Goarm64,
		GOMIPS:    a.Gomips,
		GOPPC64:   a.Goppc64,
		GORISCV64: a.Goriscv64,
		FloatABI:  artifact.ExtraOr(*a, artifact.ExtraABI, ""),
	}
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
