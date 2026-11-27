package bottle

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/commitauthor"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	extraConfig = "HomebrewBottleConfig"
	extraTag    = "BottleTag"

	// defaultMacOS matches the oldest macOS supported by current Go
	// toolchains. Homebrew pours a bottle on its tagged release and newer.
	defaultMacOS = "monterey"
)

// ErrNoBinaries happens when no darwin/linux binaries match a config.
var ErrNoBinaries = errors.New("no darwin or linux binaries found")

// Archive builds bottles out of the release binaries. It runs before
// checksums, so bottles are checksummed, signed and uploaded like any other
// release artifact.
type Archive struct{}

func (Archive) String() string { return "homebrew bottles" }

func (Archive) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Homebrew) || len(ctx.Config.HomebrewBottles) == 0
}

func (Archive) Default(ctx *context.Context) error {
	for i := range ctx.Config.HomebrewBottles {
		cfg := &ctx.Config.HomebrewBottles[i]
		cfg.CommitAuthor = commitauthor.Default(cfg.CommitAuthor)
		if cfg.CommitMessageTemplate == "" {
			cfg.CommitMessageTemplate = "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if cfg.Name == "" {
			cfg.Name = ctx.Config.ProjectName
		}
		if cfg.Directory == "" {
			cfg.Directory = "Formula"
		}
		if cfg.Goamd64 == "" {
			cfg.Goamd64 = "v1"
		}
		if cfg.Bottles.MacOS == "" {
			cfg.Bottles.MacOS = defaultMacOS
		}
	}
	return nil
}

func (Archive) Run(ctx *context.Context) error {
	for i := range ctx.Config.HomebrewBottles {
		cfg := &ctx.Config.HomebrewBottles[i]
		name, err := tmpl.New(ctx).Apply(cfg.Name)
		if err != nil {
			return err
		}
		cfg.Name = name
		if err := buildBottles(ctx, *cfg); err != nil {
			return err
		}
	}
	return nil
}

// binaries returns the binaries of cfg grouped by bottle tag.
func binaries(ctx *context.Context, cfg config.HomebrewBottle) (map[string][]*artifact.Artifact, error) {
	filters := []artifact.Filter{
		artifact.ByTypes(artifact.Binary, artifact.UniversalBinary),
		artifact.Or(artifact.ByGoos("darwin"), artifact.ByGoos("linux")),
		artifact.Or(
			artifact.And(artifact.ByGoarch("amd64"), artifact.ByGoamd64(cfg.Goamd64)),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("all"),
		),
		artifact.OnlyReplacingUnibins,
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}

	byTag := map[string][]*artifact.Artifact{}
	for _, art := range ctx.Artifacts.Filter(artifact.And(filters...)).List() {
		for _, tag := range tagsFor(art.Goos, art.Goarch, cfg.Bottles.MacOS) {
			byTag[tag] = append(byTag[tag], art)
		}
	}
	if len(byTag) == 0 {
		return nil, ErrNoBinaries
	}
	return byTag, nil
}

// tagsFor maps a GOOS/GOARCH pair to Homebrew bottle tags.
func tagsFor(goos, goarch, macos string) []string {
	switch goos + "/" + goarch {
	case "darwin/arm64":
		return []string{"arm64_" + macos}
	case "darwin/amd64":
		return []string{macos}
	case "darwin/all":
		return []string{"arm64_" + macos, macos}
	case "linux/arm64":
		return []string{"arm64_linux"}
	case "linux/amd64":
		return []string{"x86_64_linux"}
	}
	return nil
}

// bottleFilename is the asset name Homebrew downloads from a non-GitHub
// Packages root_url: "<name>-<version>.<tag>.bottle.tar.gz".
func bottleFilename(name, version, tag string) string {
	return fmt.Sprintf("%s-%s.%s.bottle.tar.gz", name, version, tag)
}

func buildBottles(ctx *context.Context, cfg config.HomebrewBottle) error {
	byTag, err := binaries(ctx, cfg)
	if err != nil {
		return err
	}

	files := make([]config.HomebrewBottleFile, 0, len(cfg.Bottles.Files))
	for _, f := range cfg.Bottles.Files {
		src, err := tmpl.New(ctx).Apply(f.Src)
		if err != nil {
			return err
		}
		files = append(files, config.HomebrewBottleFile{Src: src, Dst: f.Dst})
	}

	dir := filepath.Join(ctx.Config.Dist, "homebrew_bottles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, tag := range slices.Sorted(maps.Keys(byTag)) {
		filename := bottleFilename(cfg.Name, ctx.Version, tag)
		dst := filepath.Join(dir, filename)
		log.WithField("bottle", dst).Info("creating")
		if err := writeBottle(dst, cfg.Name, ctx.Version, ctx.Git.CommitDate, byTag[tag], files); err != nil {
			return fmt.Errorf("could not create %s: %w", filename, err)
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.HomebrewBottle,
			Name: filename,
			Path: dst,
			Extra: map[string]any{
				artifact.ExtraID: cfg.Name,
				extraTag:         tag,
				extraConfig:      cfg,
			},
		})
	}
	return nil
}

// writeBottle writes a bottle tarball laid out as Homebrew expects:
// <name>/<version>/bin/<binary>, plus any extra files under the keg.
func writeBottle(dst, name, version string, mtime time.Time, bins []*artifact.Artifact, files []config.HomebrewBottleFile) (err error) {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	keg := path.Join(name, version)

	type entry struct{ src, dst string }
	var entries []entry
	for _, bin := range bins {
		entries = append(entries, entry{bin.Path, path.Join(keg, "bin", bin.Name)})
	}
	for _, file := range files {
		entries = append(entries, entry{file.Src, path.Join(keg, path.Clean(file.Dst))})
	}
	seen := map[string]bool{}
	for _, e := range entries {
		for d := path.Dir(e.dst); d != "." && !seen[d]; d = path.Dir(d) {
			seen[d] = true
		}
	}
	for _, d := range slices.Sorted(maps.Keys(seen)) {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     d + "/",
			Mode:     0o755,
			ModTime:  mtime,
		}); err != nil {
			return err
		}
	}
	for _, e := range entries {
		if err := addFile(tw, e.src, e.dst, mtime); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}

func addFile(tw *tar.Writer, src, dst string, mtime time.Time) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	mode := int64(0o644)
	if info.Mode()&0o111 != 0 || path.Base(path.Dir(dst)) == "bin" {
		mode = 0o755
	}
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     dst,
		Mode:     mode,
		Size:     info.Size(),
		ModTime:  mtime,
	}); err != nil {
		return err
	}
	_, err = io.Copy(tw, in)
	return err
}
