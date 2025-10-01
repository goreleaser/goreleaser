// Package makeself implements the Pipe interface providing makeself
// self-extracting archive support.
package makeself

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/archivefiles"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const defaultNameTemplate = `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}.run`

// Pipe for makeself packaging.
type Pipe struct{}

func (Pipe) String() string { return "makeself packages" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Makeself) || len(ctx.Config.Makeselfs) == 0
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("makeselfs")
	for i := range ctx.Config.Makeselfs {
		cfg := &ctx.Config.Makeselfs[i]
		if cfg.ID == "" {
			cfg.ID = "default"
		}
		if cfg.Filename == "" {
			cfg.Filename = defaultNameTemplate
		}
		if cfg.Name == "" {
			cfg.Name = "{{ .ProjectName }}"
		}
		if len(cfg.Goos) == 0 {
			cfg.Goos = []string{"linux", "darwin"}
		}
		ids.Inc(cfg.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, cfg := range ctx.Config.Makeselfs {
		g.Go(func() error {
			return doRun(ctx, cfg)
		})
	}
	return g.Wait()
}

func doRun(ctx *context.Context, cfg config.Makeself) error {
	disable, err := tmpl.New(ctx).Bool(cfg.Disable)
	if err != nil {
		return err
	}
	if disable {
		return pipe.Skip("disabled")
	}

	groups := getArtifacts(ctx, cfg)
	if len(groups) == 0 {
		return fmt.Errorf("no binaries found for builds %v with goos %v goarch %v", cfg.IDs, cfg.Goos, cfg.Goarch)
	}

	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for plat, binaries := range groups {
		g.Go(func() error {
			return create(ctx, cfg, plat, binaries)
		})
	}
	return g.Wait()
}

// LSM file representation.
//
// See: https://ibiblio.org/pub/linux/LSM-TEMPLATE.html
type LSM struct {
	Title         string
	Version       string
	Description   string
	Keywords      []string
	Author        string
	MaintainedBy  string
	PrimarySite   string
	Platform      string
	CopyingPolicy string
}

// String implements the Stringer interface.
func (l LSM) String() string {
	var sb strings.Builder
	_, _ = sb.WriteString("Begin4\n")
	w := func(name, value string) {
		if value == "" {
			return
		}
		_, _ = fmt.Fprintf(&sb, "%s: %s\n", name, value)
	}

	w("Title", l.Title)
	w("Version", l.Version)
	w("Description", l.Description)
	w("Keywords", strings.Join(l.Keywords, ", "))
	w("Author", l.Author)
	w("Maintained-by", l.MaintainedBy)
	w("Primary-site", l.PrimarySite)
	w("Platforms", l.Platform)
	w("Copying-policy", l.CopyingPolicy)

	_, _ = sb.WriteString("End")
	return sb.String()
}

func create(ctx *context.Context, cfg config.Makeself, plat string, binaries []*artifact.Artifact) error {
	binary := binaries[0]
	tpl := tmpl.New(ctx).
		WithArtifact(binary)

	name := cfg.Name
	filename := cfg.Filename
	description := cfg.Description
	maintainer := cfg.Maintainer
	homepage := cfg.Homepage
	license := cfg.License
	script := cfg.Script
	compression := cfg.Compression
	extraArgs := cfg.ExtraArgs
	keywords := cfg.Keywords

	if err := tpl.ApplyAll(
		&name,
		&filename,
		&description,
		&maintainer,
		&homepage,
		&license,
		&script,
		&compression,
	); err != nil {
		return err
	}
	if err := tpl.ApplySlice(&extraArgs); err != nil {
		return err
	}
	if err := tpl.ApplySlice(&keywords); err != nil {
		return err
	}

	lsm := LSM{
		Title:         name,
		Version:       ctx.Version,
		Description:   description,
		Keywords:      keywords,
		MaintainedBy:  maintainer,
		Author:        maintainer,
		PrimarySite:   homepage,
		CopyingPolicy: license,
		Platform:      plat,
	}.String()

	if script == "" {
		return errors.New("script is required")
	}

	dir, err := setupContext(ctx, cfg, tpl, plat, lsm, script, binaries)
	if err != nil {
		return err
	}

	log := log.WithField("package", filename).WithField("dir", dir)
	log.Info("creating makeself package")

	arg := makeArg(name, filename, compression, "./"+filepath.Base(script), extraArgs)
	cmd := exec.CommandContext(ctx, "makeself", arg...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)
	if err := cmd.Run(); err != nil {
		return gerrors.Wrap(
			err,
			"could not create makeself package",
			"args", strings.Join(cmd.Args, " "),
			"id", cfg.ID,
			"output", b.String(),
		)
	}

	path := filepath.Join(dir, filename)
	ctx.Artifacts.Add(makeArtifact(cfg, binary, filename, path))
	return nil
}

func setupContext(
	ctx *context.Context,
	cfg config.Makeself,
	tpl *tmpl.Template,
	plat, lsm, script string,
	binaries []*artifact.Artifact,
) (string, error) {
	dir := filepath.Join(ctx.Config.Dist, "makeself", cfg.ID, plat)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	for _, binary := range binaries {
		dst := filepath.Join(dir, binary.Name)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", fmt.Errorf("failed to create directory for %s: %w", binary.Name, err)
		}
		if err := gio.Copy(binary.Path, dst); err != nil {
			return "", fmt.Errorf("failed to copy binary %s: %w", binary.Name, err)
		}
	}

	files, err := archivefiles.Eval(tpl, toArchiveFiles(cfg.Files))
	if err != nil {
		return "", fmt.Errorf("failed to find files to archive: %w", err)
	}
	for _, f := range files {
		dst := filepath.Join(dir, f.Destination)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", fmt.Errorf("failed to create directory for %s: %w", f.Destination, err)
		}
		if err := gio.Copy(f.Source, dst); err != nil {
			return "", fmt.Errorf("failed to copy file %s: %w", f.Source, err)
		}
	}
	if err := gio.Copy(script, filepath.Join(dir, filepath.Base(script))); err != nil {
		return "", fmt.Errorf("failed to copy binary %s: %w", script, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.lsm"), []byte(lsm), 0o644); err != nil {
		return "", fmt.Errorf("failed to write LSM file: %w", err)
	}
	return dir, nil
}

func makeArg(name, filename, compression, script string, extraArgs []string) []string {
	arg := []string{"--quiet"} // Always run quietly
	switch compression {
	case "gzip", "bzip2", "xz", "lzo", "compress":
		arg = append(arg, "--"+compression)
	case "none":
		arg = append(arg, "--nocomp")
	default:
		// let makeself choose.
	}

	arg = append(arg, "--lsm", "package.lsm")
	arg = append(arg, extraArgs...)
	return append(arg, ".", filename, name, script)
}

func makeArtifact(cfg config.Makeself, binary *artifact.Artifact, filename, path string) *artifact.Artifact {
	// Create artifact
	art := &artifact.Artifact{
		Type:      artifact.Makeself,
		Name:      filename,
		Path:      path,
		Goos:      binary.Goos,
		Goarch:    binary.Goarch,
		Goamd64:   binary.Goamd64,
		Go386:     binary.Go386,
		Goarm:     binary.Goarm,
		Goarm64:   binary.Goarm64,
		Gomips:    binary.Gomips,
		Goppc64:   binary.Goppc64,
		Goriscv64: binary.Goriscv64,
		Target:    binary.Target,
		Extra: map[string]any{
			artifact.ExtraID:     cfg.ID,
			artifact.ExtraFormat: "makeself",
			artifact.ExtraExt:    filepath.Ext(filename),
		},
	}
	if rep, ok := binary.Extra[artifact.ExtraReplaces]; ok {
		art.Extra[artifact.ExtraReplaces] = rep
	}
	return art
}

func getArtifacts(ctx *context.Context, cfg config.Makeself) map[string][]*artifact.Artifact {
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
			artifact.ByType(artifact.Header),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		),
		artifact.ByIDs(cfg.IDs...),
		artifact.ByGooses(cfg.Goos...),
		artifact.ByGoarches(cfg.Goarch...),
	}
	return ctx.Artifacts.
		Filter(artifact.And(filters...)).
		GroupByPlatform()
}

func toArchiveFiles(in []config.MakeselfFile) []config.File {
	result := make([]config.File, 0, len(in))
	for _, f := range in {
		result = append(result, config.File{
			Source:      f.Source,
			Destination: f.Destination,
			StripParent: f.StripParent,
		})
	}
	return result
}
