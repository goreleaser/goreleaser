// Package snapcraft implements the Pipe interface providing Snapcraft bindings.
package snapcraft

import (
	"errors"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v2"
)

// ErrNoSnapcraft is shown when snapcraft cannot be found in $PATH
var ErrNoSnapcraft = errors.New("snapcraft not present in $PATH")

// SnapcraftMetadata to generate the snap package
type SnapcraftMetadata struct {
	Name          string
	Version       string
	Summary       string
	Description   string
	Grade         string `yaml:",omitempty"`
	Confinement   string `yaml:",omitempty"`
	Architectures []string
	Apps          map[string]AppsMetadata
	Parts         map[string]PartsMetadata
}

// AppsMetadata for the binaries that will be in the snap package
type AppsMetadata struct {
	Command string
	//	Plugs []string
	//	Daemon string
}

// PartsMetadata for the binaries that will be in the snap package
type PartsMetadata struct {
	Source string
	Plugin string
	Prime  []string
}

// Pipe for snapcraft packaging
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating Linux packages with snapcraft"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Snapcraft.Summary == "" {
		log.Info("no snapcraft summary defined, skipping")
		return nil
	}
	if ctx.Config.Snapcraft.Summary == "" {
		log.Info("no snapcraft description defined, skipping")
		return nil
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
		arch := archFor(platform)
		for folder, binaries := range groups {
			g.Go(func() error {
				return create(ctx, folder, arch, binaries)
			})
		}
	}
	return g.Wait()
}

func archFor(key string) string {
	switch {
	case strings.Contains(key, "amd64"):
		return "amd64"
	case strings.Contains(key, "386"):
		return "i386"
	case strings.Contains(key, "arm64"):
		return "arm64"
	case strings.Contains(key, "arm6"):
		return "armhf"
	}
	return key
}

func create(ctx *context.Context, folder, arch string, binaries []context.Binary) error {
	var path = filepath.Join(ctx.Config.Dist, folder)
	var file = filepath.Join(path, "snapcraft.yaml")
	log.WithField("file", file).Info("creating snapcraft metadata")

	var metadata = &SnapcraftMetadata{
		Name:          ctx.Config.ProjectName,
		Version:       ctx.Version,
		Summary:       ctx.Config.Snapcraft.Summary,
		Description:   ctx.Config.Snapcraft.Description,
		Grade:         ctx.Config.Snapcraft.Grade,
		Confinement:   ctx.Config.Snapcraft.Confinement,
		Architectures: []string{arch},
		Apps:          make(map[string]AppsMetadata),
		Parts:         make(map[string]PartsMetadata),
	}

	metadata.Parts[ctx.Config.ProjectName] = PartsMetadata{
		Source: ".",
		Plugin: "dump",
	}
	for _, binary := range binaries {
		log.WithField("path", binary.Path).
			WithField("name", binary.Name).
			Info("passed binary to snapcraft")
		metadata.Apps[binary.Name] = AppsMetadata{Command: binary.Name}
		prime := metadata.Parts[ctx.Config.ProjectName].Prime
		prime = append(prime, binary.Path)
	}
	out, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(file, out, 0644); err != nil {
		return err
	}

	snap := metadata.Name + "_" + metadata.Version + "_" + arch + ".snap"
	cmd := exec.Command("snapcraft", "snap", "--output", snap)
	cmd.Dir = path
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	ctx.AddArtifact(filepath.Join(path, snap))
	return nil
}
