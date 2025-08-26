// Package makeself implements the Pipe interface providing makeself
// self-extracting archive support.
package makeself

import (
	"bytes"
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
		ids.Inc(cfg.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, cfg := range ctx.Config.Makeselfs {
		return doRun(ctx, cfg)
	}
	return g.Wait()
}

func doRun(ctx *context.Context, cfg config.MakeselfPackage) error {
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
			artifact.ByType(artifact.Header),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		),
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}
	if len(cfg.Goos) > 0 {
		gf := make([]artifact.Filter, len(cfg.Goos))
		for i, goos := range cfg.Goos {
			gf[i] = artifact.ByGoos(goos)
		}
		filters = append(filters, artifact.Or(gf...))
	}
	if len(cfg.Goarch) > 0 {
		gf := make([]artifact.Filter, len(cfg.Goarch))
		for i, goarch := range cfg.Goarch {
			gf[i] = artifact.ByGoarch(goarch)
		}
		filters = append(filters, artifact.Or(gf...))
	}

	groupedBinaries := ctx.Artifacts.
		Filter(artifact.And(filters...)).
		GroupByPlatform()
	if len(groupedBinaries) == 0 {
		return fmt.Errorf("no binaries found for builds %v with goos %v goarch %v", cfg.IDs, cfg.Goos, cfg.Goarch)
	}

	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for plat, binaries := range groupedBinaries {
		g.Go(func() error {
			return create(ctx, cfg, plat, binaries)
		})
	}
	return g.Wait()
}

// https://ibiblio.org/pub/linux/LSM-TEMPLATE.html
const lsmTemplate = `Begin4
Title: {{ .Title }}
Version: {{ .Version }}
{{ with .Description }}Description: {{ . }}{{ end }}
{{ with .Keywords }} Keywords: {{ . }}{{ end }}
{{- with .Maintainer }}
Author: {{ . }}
Maintained-by: {{ . }}
{{- end }}
{{ with .Homepage }}Primary-site: {{ . }}{{ end }}
Platforms: {{ .Platform }}
{{ with .License }}Copying-policy: {{ . }}{{ end }}
End`

func create(ctx *context.Context, cfg config.MakeselfPackage, plat string, binaries []*artifact.Artifact) error {
	tpl := tmpl.New(ctx)
	if len(binaries) > 0 {
		tpl = tpl.WithArtifact(binaries[0])
	}

	disable, err := tpl.Bool(cfg.Disable)
	if err != nil {
		return err
	}
	if disable {
		return pipe.Skip("disabled")
	}

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

	lsm, err := tpl.WithExtraFields(tmpl.Fields{
		"Title":       name,
		"Description": description,
		"Keywords":    strings.Join(keywords, ", "),
		"Maintainer":  maintainer,
		"Homepage":    homepage,
		"License":     license,
		"Platform":    plat,
	}).Apply(lsmTemplate)
	if err != nil {
		return err
	}

	dir := filepath.Join(ctx.Config.Dist, "makeself", cfg.ID, plat)
	packagePath := filepath.Join(dir, filename)

	log := log.WithField("package", filename).WithField("dir", dir)
	log.Info("creating makeself package")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	for _, binary := range binaries {
		dst := filepath.Join(dir, filepath.Base(binary.Name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", binary.Name, err)
		}
		if err := gio.Copy(binary.Path, dst); err != nil {
			return fmt.Errorf("failed to copy binary %s: %w", binary.Name, err)
		}
	}

	files, err := archivefiles.Eval(tpl, cfg.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}
	for _, f := range files {
		dst := filepath.Join(dir, f.Destination)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", f.Destination, err)
		}
		if err := gio.Copy(f.Source, dst); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", f.Source, err)
		}
	}
	if err := gio.Copy(script, filepath.Join(dir, "setup.sh")); err != nil {
		return fmt.Errorf("failed to copy binary %s: %w", script, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.lsm"), []byte(lsm), 0o644); err != nil {
		return fmt.Errorf("failed to write LSM file: %w", err)
	}

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
	arg = append(arg, ".", filename)
	arg = append(arg, name)
	arg = append(arg, "./setup.sh")

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

	// Create artifact
	art := &artifact.Artifact{
		Type: artifact.MakeselfPackage,
		Name: filename,
		Path: packagePath,
		Extra: map[string]any{
			artifact.ExtraID:     cfg.ID,
			artifact.ExtraFormat: "makeself",
			artifact.ExtraExt:    filepath.Ext(filename),
		},
	}

	if len(binaries) > 0 {
		binary := binaries[0]
		art.Goos = binary.Goos
		art.Goarch = binary.Goarch
		art.Goamd64 = binary.Goamd64
		art.Go386 = binary.Go386
		art.Goarm = binary.Goarm
		art.Goarm64 = binary.Goarm64
		art.Gomips = binary.Gomips
		art.Goppc64 = binary.Goppc64
		art.Goriscv64 = binary.Goriscv64
		art.Target = binary.Target
		if rep, ok := binaries[0].Extra[artifact.ExtraReplaces]; ok {
			art.Extra[artifact.ExtraReplaces] = rep
		}
	}

	ctx.Artifacts.Add(art)
	return nil
}
