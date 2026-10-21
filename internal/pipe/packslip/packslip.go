// Package packslip creates a signed packslip release manifest describing the
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
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	packslip "github.com/jdx/packslip/go"
)

const defaultNameTemplate = "packslip.sigstore.json"

// Pipe that creates the packslip manifest.
type Pipe struct{}

func (Pipe) String() string { return "creating packslip manifest" }

func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Packslip)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	cfg := &ctx.Config.Packslip
	if cfg.NameTemplate == "" {
		cfg.NameTemplate = defaultNameTemplate
	}
	if cfg.Project == "" {
		if gh := ctx.Config.Release.GitHub; gh.Owner != "" && gh.Name != "" {
			cfg.Project = "github.com/" + gh.Owner + "/" + gh.Name
		}
	}
	return nil
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

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Packslip
	enabled, err := tmpl.New(ctx).Bool(cfg.Enabled)
	if err != nil {
		return err
	}
	if !enabled {
		return pipe.Skip("packslip.enabled is not set")
	}
	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}
	if skips.Any(ctx, skips.Publish) {
		// the manifest is signed into a public transparency log, so it is
		// only created for releases that are actually published.
		return pipe.ErrSkipPublishEnabled
	}
	if cfg.Project == "" {
		return errors.New("packslip.project must be set when the release is not on GitHub")
	}

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
		return pipe.Skip("no archives or binaries to describe")
	}

	if urlTemplate == "" {
		cl, err := client.NewReleaseClient(ctx)
		if err != nil {
			return err
		}
		if urlTemplate, err = cl.ReleaseURLTemplate(ctx); err != nil {
			return fmt.Errorf("packslip.url_template: %w", err)
		}
	}

	token, err := packslip.AmbientIDToken(ctx)
	if err != nil {
		return err
	}
	identity, err := packslip.IdentityFromToken(token)
	if err != nil {
		return err
	}

	inputs := make([]packslip.ArtifactInput, 0, len(artifacts))
	for _, a := range artifacts {
		url, err := tmpl.New(ctx).WithArtifact(a).Apply(urlTemplate)
		if err != nil {
			return fmt.Errorf("packslip.url_template: %w", err)
		}
		inputs = append(inputs, packslip.ArtifactInput{
			Path:     a.Path,
			Artifact: describe(a, url),
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

	statement, err := packslip.Build(predicate, inputs)
	if err != nil {
		return fmt.Errorf("could not build packslip statement: %w", err)
	}

	name, err := tmpl.New(ctx).Apply(cfg.NameTemplate)
	if err != nil {
		return fmt.Errorf("packslip.name_template: %w", err)
	}
	log.WithField("file", name).Info("signing packslip manifest")
	bundle, err := signFunc(ctx, statement, token)
	if err != nil {
		return fmt.Errorf("could not sign packslip manifest: %w", err)
	}

	file := filepath.Join(ctx.Config.Dist, name)
	if err := os.WriteFile(file, bundle, 0o644); err != nil {
		return err
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableFile,
		Name: name,
		Path: file,
		Extra: map[string]any{
			artifact.ExtraID: "packslip",
		},
	})
	return nil
}

// describe maps a GoReleaser artifact to a packslip artifact.
func describe(a *artifact.Artifact, url string) packslip.Artifact {
	format := artifact.ExtraOr(*a, artifact.ExtraFormat, "")
	if format == "binary" {
		format = "raw"
	}
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
		case strings.Contains(a.Target, "musl"):
			out.Libc = "musl"
		default:
			out.Libc = "gnu"
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
	return out
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
		// ArchFromGo keeps the GOARM version; only a float option that is
		// not the version's default is left to distinguish.
		version, float, _ := strings.Cut(a.Goarm, ",")
		if float == "softfloat" && version != "5" {
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
	var host, repoPath string
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return ""
		}
		host, repoPath = u.Hostname(), u.Path
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
	}
	repoPath = strings.Trim(repoPath, "/")
	if host == "" || !strings.Contains(repoPath, "/") {
		return ""
	}
	return "https://" + host + "/" + repoPath
}
