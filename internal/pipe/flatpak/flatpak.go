// Package flatpak implements the Pipe interface providing Flatpak bindings.
//
//nolint:tagliatelle
package flatpak

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/redact"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

var (
	ErrNoFlatpakBuilder = errors.New("flatpak-builder not present in $PATH")
	ErrNoAppID          = errors.New("no app_id provided for flatpak")
	ErrNoRuntime        = errors.New("no runtime provided for flatpak")
	ErrNoRuntimeVersion = errors.New("no runtime_version provided for flatpak")
	ErrNoSDK            = errors.New("no sdk provided for flatpak")
)

const defaultNameTemplate = `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`

// Manifest is the Flatpak manifest written to disk.
type Manifest struct {
	ID             string           `json:"id"`
	Runtime        string           `json:"runtime"`
	RuntimeVersion string           `json:"runtime-version"`
	SDK            string           `json:"sdk"`
	Command        string           `json:"command"`
	FinishArgs     []string         `json:"finish-args,omitempty"`
	Modules        []ManifestModule `json:"modules"`
}

// ManifestModule is a module entry in the Flatpak manifest.
type ManifestModule struct {
	Name          string           `json:"name"`
	BuildSystem   string           `json:"buildsystem"`
	BuildCommands []string         `json:"build-commands"`
	Sources       []ManifestSource `json:"sources"`
}

// ManifestSource is a source entry for a Flatpak module.
type ManifestSource struct {
	Type         string `json:"type"`
	Path         string `json:"path"`
	DestFilename string `json:"dest-filename,omitempty"`
}

// Pipe for Flatpak packaging.
type Pipe struct{}

func (Pipe) String() string                         { return "flatpak packages" }
func (Pipe) ContinueOnError() bool                  { return true }
func (Pipe) Dependencies(*context.Context) []string { return []string{"flatpak-builder", "flatpak"} }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Flatpak) || len(ctx.Config.Flatpaks) == 0
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("flatpaks")
	for i := range ctx.Config.Flatpaks {
		fp := &ctx.Config.Flatpaks[i]
		if fp.NameTemplate == "" {
			fp.NameTemplate = defaultNameTemplate
		}
		if fp.AppID == "" {
			return ErrNoAppID
		}
		if fp.Runtime == "" {
			return ErrNoRuntime
		}
		if fp.RuntimeVersion == "" {
			return ErrNoRuntimeVersion
		}
		if fp.SDK == "" {
			return ErrNoSDK
		}
		ids.Inc(fp.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, fp := range ctx.Config.Flatpaks {
		if err := doRun(ctx, fp); err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, fp config.Flatpak) error {
	disable, err := tmpl.New(ctx).Bool(fp.Disable)
	if err != nil {
		return err
	}
	if disable {
		return pipe.Skip("configuration is disabled")
	}
	if _, err := exec.LookPath("flatpak-builder"); err != nil {
		return ErrNoFlatpakBuilder
	}

	g := semerrgroup.NewBlockingFirst(semerrgroup.New(ctx.Parallelism))
	filters := []artifact.Filter{
		artifact.ByGoos("linux"),
		artifact.ByType(artifact.Binary),
		artifact.ByGoarches("amd64", "arm64"),
	}
	if len(fp.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(fp.IDs...))
	}
	for _, binaries := range ctx.Artifacts.Filter(
		artifact.And(filters...),
	).GroupByPlatform() {
		arch := archToFlatpak[binaries[0].Goarch]
		g.Go(func() error {
			return create(ctx, fp, arch, binaries)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, fp config.Flatpak, arch string, binaries []*artifact.Artifact) error {
	folder, err := tmpl.New(ctx).WithArtifact(binaries[0]).Apply(fp.NameTemplate)
	if err != nil {
		return err
	}

	flatpakName := folder + ".flatpak"
	log := log.WithField("arch", arch).WithField("flatpak", flatpakName)
	command := fp.Command
	if command == "" {
		command = filepath.Base(binaries[0].Name)
	}

	workDir := filepath.Join(ctx.Config.Dist, "flatpak", folder, arch)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("failed to create flatpak work directory: %w", err)
	}

	manifest := &Manifest{
		ID:             fp.AppID,
		Runtime:        fp.Runtime,
		RuntimeVersion: fp.RuntimeVersion,
		SDK:            fp.SDK,
		Command:        command,
		FinishArgs:     fp.FinishArgs,
	}

	installCmds := make([]string, 0, len(binaries))
	sources := make([]ManifestSource, 0, len(binaries))
	for _, binary := range binaries {
		binaryName := filepath.Base(binary.Path)
		dest := filepath.Join(workDir, binaryName)
		log.WithField("src", binary.Path).WithField("dst", dest).Debug("copying binary")
		if err := gio.CopyWithMode(binary.Path, dest, 0o555); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}
		sources = append(sources, ManifestSource{
			Type:         "file",
			Path:         binaryName,
			DestFilename: binaryName,
		})
		installCmds = append(installCmds, fmt.Sprintf("install -Dm755 %s /app/bin/%s", binaryName, binaryName))
	}

	manifest.Modules = []ManifestModule{
		{
			Name:          fp.AppID,
			BuildSystem:   "simple",
			BuildCommands: installCmds,
			Sources:       sources,
		},
	}

	manifestName := fp.AppID + ".json"
	manifestFile := filepath.Join(workDir, manifestName)
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal flatpak manifest: %w", err)
	}

	log.WithField("file", manifestFile).Debug("writing flatpak manifest")
	if err := os.WriteFile(manifestFile, manifestBytes, 0o644); err != nil { //nolint:gosec
		return err
	}

	if err := runCmd(
		ctx,
		workDir,
		"failed to build flatpak",
		"flatpak-builder",
		"--force-clean",
		"--arch="+arch,
		"--default-branch="+ctx.Version,
		"--repo=repo",
		"build",
		manifestName,
	); err != nil {
		return err
	}

	log.Info("creating bundle")
	if err := runCmd(
		ctx,
		workDir,
		"failed to create flatpak bundle",
		"flatpak",
		"build-bundle",
		"--arch="+arch,
		"repo",
		flatpakName,
		fp.AppID,
		ctx.Version,
	); err != nil {
		return err
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Type:      artifact.Flatpak,
		Name:      flatpakName,
		Path:      filepath.Join(workDir, flatpakName),
		Goos:      binaries[0].Goos,
		Goarch:    binaries[0].Goarch,
		Goamd64:   binaries[0].Goamd64,
		Go386:     binaries[0].Go386,
		Goarm:     binaries[0].Goarm,
		Goarm64:   binaries[0].Goarm64,
		Gomips:    binaries[0].Gomips,
		Goppc64:   binaries[0].Goppc64,
		Goriscv64: binaries[0].Goriscv64,
		Target:    binaries[0].Target,
		Extra: map[string]any{
			artifact.ExtraID:     fp.ID,
			artifact.ExtraFormat: "flatpak",
		},
	})
	return nil
}

func runCmd(ctx *context.Context, dir, errMsg, bin string, args ...string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = redact.Writer(io.MultiWriter(logext.NewWriter(), w), cmd.Env)
	cmd.Stdout = redact.Writer(io.MultiWriter(logext.NewWriter(), w), cmd.Env)
	if err := cmd.Run(); err != nil {
		return gerrors.Wrap(
			err,
			gerrors.WithMessage(errMsg),
			gerrors.WithDetails("args", strings.Join(cmd.Args, " ")),
			gerrors.WithOutput(b.String()),
		)
	}
	return nil
}

var archToFlatpak = map[string]string{
	"amd64": "x86_64",
	"arm64": "aarch64",
}
