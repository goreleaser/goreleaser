package docker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

func init() {
	registerManifester(useDocker, dockerManifester{})

	registerImager(useDocker, dockerImager{})
	registerImager(useBuildx, dockerImager{
		buildx: true,
	})
}

const maxRetries = 10

type dockerManifester struct{}

func (m dockerManifester) Create(ctx *context.Context, manifest string, images, flags []string) error {
	return m.tryCreate(ctx, manifest, images, flags, 0)
}

func (m dockerManifester) tryCreate(ctx *context.Context, manifest string, images, flags []string, try int) error {
	_ = runCommand(ctx, ".", "docker", "manifest", "rm", manifest)

	args := []string{"manifest", "create", manifest}
	args = append(args, images...)
	args = append(args, flags...)

	if err := runCommand(ctx, ".", "docker", args...); err != nil {
		if strings.Contains(err.Error(), "manifest verification failed for digest") && try < maxRetries {
			// this error happens every so often for some reason... retry
			log.WithField("try", try+1).
				WithField("maxRetries", maxRetries).
				WithField("manifest", manifest).
				WithField("images", images).
				WithField("flags", flags).
				WithError(err).
				Warn("got an error while creating the manifest, will retry")
			return m.tryCreate(ctx, manifest, images, flags, try+1)
		}

		return fmt.Errorf("failed to create %s: %w", manifest, err)
	}
	return nil
}

func (m dockerManifester) Push(ctx *context.Context, manifest string, flags []string) (string, error) {
	args := []string{"manifest", "push", manifest}
	args = append(args, flags...)
	bts, err := runCommandWithOutput(ctx, ".", "docker", args...)
	if err != nil {
		return "", fmt.Errorf("failed to push %s: %w", manifest, err)
	}
	digest := dockerDigestPattern.FindString(string(bts))
	if digest == "" {
		return "", fmt.Errorf("failed to find docker digest in docker push output: %s", string(bts))
	}
	return digest, nil
}

type dockerImager struct {
	buildx bool
}

var dockerDigestPattern = regexp.MustCompile("sha256:[a-z0-9]{64}")

func (i dockerImager) Push(ctx *context.Context, image string, _ []string) (string, error) {
	bts, err := runCommandWithOutput(ctx, ".", "docker", "push", image)
	if err != nil {
		return "", fmt.Errorf("failed to push %s: %w", image, err)
	}
	digest := dockerDigestPattern.FindString(string(bts))
	if digest == "" {
		return "", fmt.Errorf("failed to find docker digest in docker push output: %s", string(bts))
	}
	return digest, nil
}

func (i dockerImager) Build(ctx *context.Context, root string, images, flags []string) error {
	if err := runCommand(ctx, root, "docker", i.buildCommand(images, flags)...); err != nil {
		return fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	return nil
}

func (i dockerImager) buildCommand(images, flags []string) []string {
	base := []string{"build", "."}
	if i.buildx {
		base = []string{"buildx", "build", ".", "--load"}
	}
	for _, image := range images {
		base = append(base, "-t", image)
	}
	base = append(base, flags...)
	return base
}
