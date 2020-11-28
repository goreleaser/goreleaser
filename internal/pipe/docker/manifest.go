package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
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
	var g = semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, manifest := range ctx.Config.DockerManifests {
		manifest:=manifest
		g.Go(func() error {
			man, err:= tmpl.New(ctx).Apply(manifest.ManifestTemplate)
			if err != nil{
				return err
			}
			var imgs []string
			for _, img := range manifest.ImageTemplates {
				str, err := tmpl.New(ctx).Apply(img)
				if err != nil {
					return err
				}
				imgs = append(imgs, str)
			}
			if strings.TrimSpace(man) == "" {
				return pipe.Skip("manifest name is empty")
			}
			if strings.TrimSpace(strings.Join(manifest.ImageTemplates, "")) == "" {
				return pipe.Skip("manifest has no images")
			}
			if err := dockerManifestCreate(ctx, man, imgs, manifest.CreateFlags); err != nil {
				return err
			}
			return dockerManifestPush(ctx, man, manifest.PushFlags)
		})
	}
	return g.Wait()
}

func dockerManifestCreate(ctx *context.Context, manifest string, images, flags []string) error {
	log.WithField("manifest", manifest).Info("creating docker manifest")
	var args = []string{"manifest", "create", manifest}
	for _, img := range images {
		args = append(args, "--amend", img)
	}
	args = append(args, flags...)
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", args...)
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest output: \n%s", string(out))
	return nil
}

func dockerManifestPush(ctx *context.Context, manifest string, flags []string) error {
	log.WithField("manifest", manifest).Info("pushing docker manifest")
	var args = []string{"manifest", "push", manifest}
	args = append(args, flags...)
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", args...)
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest output: \n%s", string(out))
	return nil
}
