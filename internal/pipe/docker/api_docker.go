package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/goreleaser/goreleaser/pkg/context"
)

func init() {
	registerManifester(useDocker, dockerManifester{})

	registerImager(useDocker, dockerImager{})
	registerImager(useBuildx, dockerImager{
		buildx: true,
	})
	registerImager(useDepot, depotImager{})
}

type dockerManifester struct{}

func (m dockerManifester) Create(ctx *context.Context, manifest string, images, flags []string) error {
	_ = runCommand(ctx, ".", "docker", "manifest", "rm", manifest)

	args := []string{"manifest", "create", manifest}
	args = append(args, images...)
	args = append(args, flags...)

	if err := runCommand(ctx, ".", "docker", args...); err != nil {
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

func (i dockerImager) Build(ctx *context.Context, config buildConfig) (string, error) {
	args := i.buildCommand(config.Images, config.Flags)
	if err := runCommand(ctx, config.RootDir, "docker", args...); err != nil {
		return "", fmt.Errorf("failed to build %s: %w", config.Images[0], err)
	}
	return "", nil
}

func (i dockerImager) buildCommand(images, flags []string) []string {
	base := []string{"build", "."}
	if i.buildx {
		base = []string{"buildx", "--builder", "default", "build", ".", "--load"}
	}
	for _, image := range images {
		base = append(base, "-t", image)
	}
	base = append(base, flags...)
	return base
}

type depotImager struct {
}

// Push is a no-op for depot as the build also pushes the image.
func (i depotImager) Push(_ *context.Context, _ string, _ []string) (string, error) {
	return "", nil
}

func (i depotImager) Build(ctx *context.Context, config buildConfig) (string, error) {
	flags := depotFlags(config)

	if err := runCommand(ctx, config.RootDir, "depot", flags...); err != nil {
		return "", fmt.Errorf("failed to build %s: %w", config.Images[0], err)
	}

	digest, err := os.ReadFile(filepath.Join(config.RootDir, "image-digest.txt"))
	if err != nil {
		return "", fmt.Errorf("unable to read image digest: %w", err)
	}

	return string(digest), nil
}

func depotFlags(config buildConfig) []string {
	flags := append([]string{"build", "."}, config.Flags...)
	flags = append(flags, "--platform", string(config.Platform))
	for _, image := range config.Images {
		flags = append(flags, "-t", image)
	}
	flags = append(flags, "--push")
	flags = append(flags, "--iidfile=image-digest.txt")
	if config.DepotProject != "" {
		flags = append(flags, "--project", config.DepotProject)
	}
	return flags
}
