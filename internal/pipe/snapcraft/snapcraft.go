// Package snapcraft implements the Pipe interface providing Snapcraft bindings.
package snapcraft

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"gopkg.in/yaml.v2"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoSnapcraft is shown when snapcraft cannot be found in $PATH.
var ErrNoSnapcraft = errors.New("snapcraft not present in $PATH")

// ErrNoDescription is shown when no description provided.
var ErrNoDescription = errors.New("no description provided for snapcraft")

// ErrNoSummary is shown when no summary provided.
var ErrNoSummary = errors.New("no summary provided for snapcraft")

// Metadata to generate the snap package.
type Metadata struct {
	Name          string
	Version       string
	Summary       string
	Description   string
	Base          string `yaml:",omitempty"`
	License       string `yaml:",omitempty"`
	Grade         string `yaml:",omitempty"`
	Confinement   string `yaml:",omitempty"`
	Architectures []string
	Layout        map[string]LayoutMetadata `yaml:",omitempty"`
	Apps          map[string]AppMetadata
	Plugs         map[string]interface{} `yaml:",omitempty"`
}

// AppMetadata for the binaries that will be in the snap package.
type AppMetadata struct {
	Command          string
	Plugs            []string `yaml:",omitempty"`
	Daemon           string   `yaml:",omitempty"`
	Completer        string   `yaml:",omitempty"`
	RestartCondition string   `yaml:"restart-condition,omitempty"`
}

type LayoutMetadata struct {
	Symlink  string `yaml:",omitempty"`
	Bind     string `yaml:",omitempty"`
	BindFile string `yaml:"bind-file,omitempty"`
	Type     string `yaml:",omitempty"`
}

const defaultNameTemplate = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}"

// Pipe for snapcraft packaging.
type Pipe struct{}

func (Pipe) String() string {
	return "snapcraft packages"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("snapcrafts")
	for i := range ctx.Config.Snapcrafts {
		snap := &ctx.Config.Snapcrafts[i]
		if snap.NameTemplate == "" {
			snap.NameTemplate = defaultNameTemplate
		}
		if len(snap.Builds) == 0 {
			for _, b := range ctx.Config.Builds {
				snap.Builds = append(snap.Builds, b.ID)
			}
		}
		ids.Inc(snap.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, snap := range ctx.Config.Snapcrafts {
		// TODO: deal with pipe.skip?
		if err := doRun(ctx, snap); err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, snap config.Snapcraft) error {
	if snap.Summary == "" && snap.Description == "" {
		return pipe.Skip("no summary nor description were provided")
	}
	if snap.Summary == "" {
		return ErrNoSummary
	}
	if snap.Description == "" {
		return ErrNoDescription
	}
	_, err := exec.LookPath("snapcraft")
	if err != nil {
		return ErrNoSnapcraft
	}

	g := semerrgroup.New(ctx.Parallelism)
	for platform, binaries := range ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("linux"),
			artifact.ByType(artifact.Binary),
			artifact.ByIDs(snap.Builds...),
		),
	).GroupByPlatform() {
		arch := linux.Arch(platform)
		if !isValidArch(arch) {
			log.WithField("arch", arch).Warn("ignored unsupported arch")
			continue
		}
		binaries := binaries
		g.Go(func() error {
			return create(ctx, snap, arch, binaries)
		})
	}
	return g.Wait()
}

func isValidArch(arch string) bool {
	// https://snapcraft.io/docs/architectures
	for _, a := range []string{"s390x", "ppc64el", "arm64", "armhf", "amd64", "i386"} {
		if arch == a {
			return true
		}
	}
	return false
}

// Publish packages.
func (Pipe) Publish(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	snaps := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	g := semerrgroup.New(ctx.Parallelism)
	for _, snap := range snaps {
		snap := snap
		g.Go(func() error {
			return push(ctx, snap)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, snap config.Snapcraft, arch string, binaries []*artifact.Artifact) error {
	log := log.WithField("arch", arch)
	folder, err := tmpl.New(ctx).
		WithArtifact(binaries[0], snap.Replacements).
		Apply(snap.NameTemplate)
	if err != nil {
		return err
	}

	// prime is the directory that then will be compressed to make the .snap package.
	folderDir := filepath.Join(ctx.Config.Dist, folder)
	primeDir := filepath.Join(folderDir, "prime")
	metaDir := filepath.Join(primeDir, "meta")
	// #nosec
	if err = os.MkdirAll(metaDir, 0o755); err != nil {
		return err
	}

	for _, file := range snap.Files {
		if file.Destination == "" {
			file.Destination = file.Source
		}
		if file.Mode == 0 {
			file.Mode = 0o644
		}
		if err := os.MkdirAll(filepath.Join(primeDir, filepath.Dir(file.Destination)), 0o755); err != nil {
			return fmt.Errorf("failed to link extra file '%s': %w", file.Source, err)
		}
		if err := link(file.Source, filepath.Join(primeDir, file.Destination), os.FileMode(file.Mode)); err != nil {
			return fmt.Errorf("failed to link extra file '%s': %w", file.Source, err)
		}
	}

	file := filepath.Join(primeDir, "meta", "snap.yaml")
	log.WithField("file", file).Debug("creating snap metadata")

	metadata := &Metadata{
		Version:       ctx.Version,
		Summary:       snap.Summary,
		Description:   snap.Description,
		Grade:         snap.Grade,
		Confinement:   snap.Confinement,
		Architectures: []string{arch},
		Layout:        map[string]LayoutMetadata{},
		Apps:          map[string]AppMetadata{},
	}

	if snap.Base != "" {
		metadata.Base = snap.Base
	}

	if snap.License != "" {
		metadata.License = snap.License
	}

	metadata.Name = ctx.Config.ProjectName
	if snap.Name != "" {
		metadata.Name = snap.Name
	}

	for targetPath, layout := range snap.Layout {
		metadata.Layout[targetPath] = LayoutMetadata{
			Symlink:  layout.Symlink,
			Bind:     layout.Bind,
			BindFile: layout.BindFile,
			Type:     layout.Type,
		}
	}

	// if the user didn't specify any apps then
	// default to the main binary being the command:
	if len(snap.Apps) == 0 {
		name := snap.Name
		if name == "" {
			name = binaries[0].Name
		}
		metadata.Apps[name] = AppMetadata{
			Command: filepath.Base(binaries[0].Name),
		}
	}

	for _, binary := range binaries {
		// build the binaries and link resources
		destBinaryPath := filepath.Join(primeDir, filepath.Base(binary.Path))
		log.WithField("src", binary.Path).
			WithField("dst", destBinaryPath).
			Debug("linking")

		if err = os.Link(binary.Path, destBinaryPath); err != nil {
			return fmt.Errorf("failed to link binary: %w", err)
		}
		if err := os.Chmod(destBinaryPath, 0o555); err != nil {
			return fmt.Errorf("failed to change binary permissions: %w", err)
		}
	}

	// setup the apps: directive for each binary
	for name, config := range snap.Apps {
		command := name
		if config.Command != "" {
			command = config.Command
		}

		// TODO: test that the correct binary is used in Command
		// See https://github.com/goreleaser/goreleaser/pull/1449
		appMetadata := AppMetadata{
			Command: strings.TrimSpace(strings.Join([]string{
				command,
				config.Args,
			}, " ")),
			Plugs:            config.Plugs,
			Daemon:           config.Daemon,
			RestartCondition: config.RestartCondition,
		}

		if config.Completer != "" {
			destCompleterPath := filepath.Join(primeDir, config.Completer)
			if err := os.MkdirAll(filepath.Dir(destCompleterPath), 0o755); err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}
			log.WithField("src", config.Completer).
				WithField("dst", destCompleterPath).
				Debug("linking")
			if err := os.Link(config.Completer, destCompleterPath); err != nil {
				return fmt.Errorf("failed to link completer: %w", err)
			}
			if err := os.Chmod(destCompleterPath, 0o644); err != nil {
				return fmt.Errorf("failed to change completer permissions: %w", err)
			}
			appMetadata.Completer = config.Completer
		}

		metadata.Apps[name] = appMetadata
		metadata.Plugs = snap.Plugs
	}

	out, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	log.WithField("file", file).Debugf("writing metadata file")
	if err = os.WriteFile(file, out, 0o644); err != nil { //nolint: gosec
		return err
	}

	snapFile := filepath.Join(ctx.Config.Dist, folder+".snap")
	log.WithField("snap", snapFile).Info("creating")
	/* #nosec */
	cmd := exec.CommandContext(ctx, "snapcraft", "pack", primeDir, "--output", snapFile)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate snap package: %s", string(out))
	}
	if !snap.Publish {
		return nil
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.PublishableSnapcraft,
		Name:   folder + ".snap",
		Path:   snapFile,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
	})
	return nil
}

const reviewWaitMsg = `Waiting for previous upload(s) to complete their review process.`

func push(ctx *context.Context, snap *artifact.Artifact) error {
	log := log.WithField("snap", snap.Name)
	log.Info("pushing snap")
	// TODO: customize --release based on snap.Grade?
	/* #nosec */
	cmd := exec.CommandContext(ctx, "snapcraft", "upload", "--release=stable", snap.Path)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), reviewWaitMsg) {
			log.Warn(reviewWaitMsg)
		} else {
			return fmt.Errorf("failed to push %s package: %s", snap.Path, string(out))
		}
	}
	snap.Type = artifact.Snapcraft
	ctx.Artifacts.Add(snap)
	return nil
}

// walks the src, recreating dirs and hard-linking files.
func link(src, dest string, mode os.FileMode) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// We have the following:
		// - src = "a/b"
		// - dest = "dist/linuxamd64/b"
		// - path = "a/b/c.txt"
		// So we join "a/b" with "c.txt" and use it as the destination.
		dst := filepath.Join(dest, strings.Replace(path, src, "", 1))
		log.WithFields(log.Fields{
			"src": path,
			"dst": dst,
		}).Debug("extra file")
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		if err := os.Link(path, dst); err != nil {
			return err
		}
		return os.Chmod(dst, mode)
	})
}
