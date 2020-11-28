package docker

import (
	"fmt"
	"os/exec"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/pipe"
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
	var tmplt = tmpl.New(ctx)
	for _, manifest := range ctx.Config.DockerManifests {
		var imgs []string
		for _, img := range manifest.ImageTemplates {
			str, err := tmplt.Apply(img)
			if err != nil {
				return err
			}
			imgs = append(imgs, str)
		}
		man, err:= tmplt.Apply(manifest.ManifestTemplate)
		if err != nil{
			return err
		}
		if err := dockerManifestCreate(ctx, man, imgs); err != nil {
			return err
		}
		return dockerManifestPush(ctx, man)
	}
	return nil
}

func dockerManifestCreate(ctx *context.Context, manifest string, images []string) error {
	log.WithField("manifest", manifest).Info("creating docker manifest")
	var args = []string{"manifest", "create", manifest}
	for _, img := range images {
		args = append(args, "--amend", img)
	}
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

func dockerManifestPush(ctx *context.Context, manifest string) error {
	log.WithField("manifest", manifest).Info("pushing docker manifest")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "manifest", "push", manifest)
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest output: \n%s", string(out))
	return nil
}
