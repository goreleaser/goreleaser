package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ManifestPipe is beta implementation of for the docker manifest feature,
// allowing to publish multi-arch docker images.
type ManifestPipe struct{}

func (ManifestPipe) String() string {
	return "docker manifests"
}

// Publish the docker manifests.
func (ManifestPipe) Publish(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	g := semerrgroup.NewSkipAware(semerrgroup.New(1))
	for _, manifest := range ctx.Config.DockerManifests {
		manifest := manifest
		g.Go(func() error {
			name, err := manifestName(ctx, manifest)
			if err != nil {
				return err
			}
			if err := dockerManifestRm(ctx, name); err != nil {
				return err
			}
			images, err := manifestImages(ctx, manifest)
			if err != nil {
				return err
			}
			if err := dockerManifestCreate(ctx, name, images, manifest.CreateFlags); err != nil {
				return err
			}
			ctx.Artifacts.Add(&artifact.Artifact{
				Type: artifact.DockerManifest,
				Name: name,
				Path: name,
			})
			return dockerManifestPush(ctx, name, manifest.PushFlags)
		})
	}
	return g.Wait()
}

func manifestName(ctx *context.Context, manifest config.DockerManifest) (string, error) {
	name, err := tmpl.New(ctx).Apply(manifest.NameTemplate)
	if err != nil {
		return name, err
	}
	if strings.TrimSpace(name) == "" {
		return name, pipe.Skip("manifest name is empty")
	}
	return name, nil
}

func manifestImages(ctx *context.Context, manifest config.DockerManifest) ([]string, error) {
	imgs := make([]string, 0, len(manifest.ImageTemplates))
	for _, img := range manifest.ImageTemplates {
		str, err := tmpl.New(ctx).Apply(img)
		if err != nil {
			return []string{}, err
		}
		imgs = append(imgs, str)
	}
	if strings.TrimSpace(strings.Join(manifest.ImageTemplates, "")) == "" {
		return imgs, pipe.Skip("manifest has no images")
	}
	return imgs, nil
}

func dockerManifestRm(ctx *context.Context, manifest string) error {
	log.WithField("manifest", manifest).Info("removing local docker manifest")
	/* #nosec */
	cmd := exec.CommandContext(ctx, "docker", "manifest", "rm", manifest)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasPrefix(string(out), "No such manifest: ") {
			// ignore "no such manifest" error, is the state we want in the end...
			return nil
		}
		return fmt.Errorf("failed to remove local docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest rm output: \n%s", string(out))
	return nil
}

func dockerManifestCreate(ctx *context.Context, manifest string, images, flags []string) error {
	log.WithField("manifest", manifest).WithField("images", images).Info("creating docker manifest")
	args := []string{"manifest", "create", manifest}
	args = append(args, images...)
	args = append(args, flags...)
	/* #nosec */
	cmd := exec.CommandContext(ctx, "docker", args...)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest output: \n%s", string(out))
	return nil
}

func dockerManifestPush(ctx *context.Context, manifest string, flags []string) error {
	log.WithField("manifest", manifest).Info("pushing docker manifest")
	args := []string{"manifest", "push", manifest}
	args = append(args, flags...)
	/* #nosec */
	cmd := exec.CommandContext(ctx, "docker", args...)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest output: \n%s", string(out))
	return nil
}
