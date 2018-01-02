// Package snapcraft implements the Pipe interface providing Snapcraft bindings.
package snapcraft

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/apex/log"
	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v2"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/filenametemplate"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/pipeline"
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
		return pipeline.Skip("no summary nor description were provided")
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

	var g errgroup.Group
	for platform, binaries := range ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("linux"),
			artifact.ByType(artifact.Binary),
		),
	).GroupByPlatform() {
		arch := linux.Arch(platform)
		binaries := binaries
		g.Go(func() error {
			return create(ctx, arch, binaries)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, arch string, binaries []artifact.Artifact) error {
	var log = log.WithField("arch", arch)
	folder, err := filenametemplate.Apply(
		ctx.Config.Snapcraft.NameTemplate,
		filenametemplate.NewFields(ctx, ctx.Config.Snapcraft.Replacements, binaries...),
	)
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
	if ctx.Config.Snapcraft.Name != "" {
		metadata.Name = ctx.Config.Snapcraft.Name
	} else {
		metadata.Name = ctx.Config.ProjectName
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
		}
		metadata.Apps[binary.Name] = appMetadata

		destBinaryPath := filepath.Join(primeDir, filepath.Base(binary.Path))
		if err = os.Link(binary.Path, destBinaryPath); err != nil {
			return err
		}
	}
	out, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(file, out, 0644); err != nil {
		return err
	}

	var snap = filepath.Join(ctx.Config.Dist, folder+".snap")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "snapcraft", "snap", primeDir, "--output", snap)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate snap package: %s", string(out))
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.LinuxPackage,
		Name:   folder + ".snap",
		Path:   snap,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
	})
	return nil
}
