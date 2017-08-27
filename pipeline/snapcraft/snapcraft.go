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
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/pipeline"
	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v2"
)

// ErrNoSnapcraft is shown when snapcraft cannot be found in $PATH
var ErrNoSnapcraft = errors.New("snapcraft not present in $PATH")

// ErrNoDescription is shown when no description provided
var ErrNoDescription = errors.New("no description provided for snapcraft")

// ErrNoSummary is shown when no summary provided
var ErrNoSummary = errors.New("no summary provided for snapcraft")

// SnapcraftMetadata to generate the snap package
type SnapcraftMetadata struct {
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

// Pipe for snapcraft packaging
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating Linux packages with snapcraft"
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
	for platform, groups := range ctx.Binaries {
		if !strings.Contains(platform, "linux") {
			log.WithField("platform", platform).Debug("skipped non-linux builds for snapcraft")
			continue
		}
		arch := linux.Arch(platform)
		for folder, binaries := range groups {
			g.Go(func() error {
				return create(ctx, folder, arch, binaries)
			})
		}
	}
	return g.Wait()
}

func create(ctx *context.Context, folder, arch string, binaries []context.Binary) error {
	var log = log.WithField("arch", arch)
	// prime is the directory that then will be compressed to make the .snap package.
	var folderDir = filepath.Join(ctx.Config.Dist, folder)
	var primeDir = filepath.Join(folderDir, "prime")
	var metaDir = filepath.Join(primeDir, "meta")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	var file = filepath.Join(primeDir, "meta", "snap.yaml")
	log.WithField("file", file).Debug("creating snap metadata")

	var metadata = &SnapcraftMetadata{
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
		if err := os.Link(binary.Path, destBinaryPath); err != nil {
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
	var cmd = exec.Command("snapcraft", "snap", primeDir, "--output", snap)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate snap package: %s", string(out))
	}
	ctx.AddArtifact(snap)
	return nil
}
