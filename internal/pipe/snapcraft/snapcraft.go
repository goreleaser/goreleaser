// Package snapcraft implements the Pipe interface providing Snapcraft bindings.
package snapcraft

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	yaml "gopkg.in/yaml.v2"
)

// ErrNoSnapcraft is shown when snapcraft cannot be found in $PATH
var ErrNoSnapcraft = errors.New("snapcraft not present in $PATH")

// ErrNoDescription is shown when no description provided
var ErrNoDescription = errors.New("no description provided for snapcraft")

// ErrNoSummary is shown when no summary provided
var ErrNoSummary = errors.New("no summary provided for snapcraft")

// Metadata to generate the snap package
type Metadata struct {
	Name          string
	Version       string
	Summary       string
	Description   string
	Grade         string `yaml:",omitempty"`
	Confinement   string `yaml:",omitempty"`
	Architectures []string
	Apps          map[string]AppMetadata
}

// AppMetadata for the binaries that will be in the snap package
type AppMetadata struct {
	Command string
	Plugs   []string `yaml:",omitempty"`
	Daemon  string   `yaml:",omitempty"`
}

const defaultNameTemplate = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

// Pipe for snapcraft packaging
type Pipe struct{}

func (Pipe) String() string {
	return "creating Linux packages with snapcraft"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	var snap = &ctx.Config.Snapcraft
	if snap.NameTemplate == "" {
		snap.NameTemplate = defaultNameTemplate
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Snapcraft.Summary == "" && ctx.Config.Snapcraft.Description == "" {
		return pipe.Skip("no summary nor description were provided")
	}
	if ctx.Config.Snapcraft.Summary == "" {
		return ErrNoSummary
	}
	if ctx.Config.Snapcraft.Description == "" {
		return ErrNoDescription
	}
	_, err := exec.LookPath("snapcraft")
	if err != nil {
		return ErrNoSnapcraft
	}

	var g = semerrgroup.New(ctx.Parallelism)
	for platform, binaries := range ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("linux"),
			artifact.ByType(artifact.Binary),
		),
	).GroupByPlatform() {
		arch := linux.Arch(platform)
		if arch == "armel" {
			log.WithField("arch", arch).Warn("ignored unsupported arch")
			continue
		}
		binaries := binaries
		g.Go(func() error {
			return create(ctx, arch, binaries)
		})
	}
	return g.Wait()
}

// Publish packages
func (Pipe) Publish(ctx *context.Context) error {
	snaps := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	var g = semerrgroup.New(ctx.Parallelism)
	for _, snap := range snaps {
		snap := snap
		g.Go(func() error {
			return push(ctx, snap)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, arch string, binaries []artifact.Artifact) error {
	var log = log.WithField("arch", arch)
	folder, err := tmpl.New(ctx).
		WithArtifact(binaries[0], ctx.Config.Snapcraft.Replacements).
		Apply(ctx.Config.Snapcraft.NameTemplate)
	if err != nil {
		return err
	}
	// prime is the directory that then will be compressed to make the .snap package.
	var folderDir = filepath.Join(ctx.Config.Dist, folder)
	var primeDir = filepath.Join(folderDir, "prime")
	var metaDir = filepath.Join(primeDir, "meta")
	// #nosec
	if err = os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	var file = filepath.Join(primeDir, "meta", "snap.yaml")
	log.WithField("file", file).Debug("creating snap metadata")

	var metadata = &Metadata{
		Version:       ctx.Version,
		Summary:       ctx.Config.Snapcraft.Summary,
		Description:   ctx.Config.Snapcraft.Description,
		Grade:         ctx.Config.Snapcraft.Grade,
		Confinement:   ctx.Config.Snapcraft.Confinement,
		Architectures: []string{arch},
		Apps:          make(map[string]AppMetadata),
	}

	metadata.Name = ctx.Config.ProjectName
	if ctx.Config.Snapcraft.Name != "" {
		metadata.Name = ctx.Config.Snapcraft.Name
	}

	for _, binary := range binaries {
		log.WithField("path", binary.Path).
			WithField("name", binary.Name).
			Debug("passed binary to snapcraft")
		appMetadata := AppMetadata{
			Command: binary.Name,
		}
		if configAppMetadata, ok := ctx.Config.Snapcraft.Apps[binary.Name]; ok {
			appMetadata.Plugs = configAppMetadata.Plugs
			appMetadata.Daemon = configAppMetadata.Daemon
			appMetadata.Command = strings.Join([]string{
				appMetadata.Command,
				configAppMetadata.Args,
			}, " ")
		}
		metadata.Apps[binary.Name] = appMetadata

		destBinaryPath := filepath.Join(primeDir, filepath.Base(binary.Path))
		if err = os.Link(binary.Path, destBinaryPath); err != nil {
			return err
		}
	}

	if _, ok := metadata.Apps[metadata.Name]; !ok {
		metadata.Apps[metadata.Name] = metadata.Apps[binaries[0].Name]
	}

	out, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(file, out, 0644); err != nil {
		return err
	}

	var snap = filepath.Join(ctx.Config.Dist, folder+".snap")
	log.WithField("snap", snap).Info("creating")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "snapcraft", "pack", primeDir, "--output", snap)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate snap package: %s", string(out))
	}
	if !ctx.Config.Snapcraft.Publish {
		return nil
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.PublishableSnapcraft,
		Name:   folder + ".snap",
		Path:   snap,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
	})
	return nil
}

func push(ctx *context.Context, snap artifact.Artifact) error {
	var cmd = exec.CommandContext(ctx, "snapcraft", "push", "--release=stable", snap.Path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push %s package: %s", snap.Path, string(out))
	}
	snap.Type = artifact.Snapcraft
	ctx.Artifacts.Add(snap)
	return nil
}
