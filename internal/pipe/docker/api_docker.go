package docker

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

func init() {
	registerManifester(useDocker, dockerManifester{})

	registerImager(useDocker, dockerImager{})
	registerImager(useBuildx, dockerImager{
		buildx: true,
	})
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

var (
	dockerDigestPattern = regexp.MustCompile("sha256:[a-z0-9]{64}")
	driverWarningOnce   sync.Once
)

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
	if i.buildx {
		checkBuildxDriver(ctx)
	}
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

// checkBuildxDriver checks if the buildx driver is docker-container and warns if not.
func checkBuildxDriver(ctx *context.Context) {
	driverWarningOnce.Do(func() {
		driver := getBuildxDriver(ctx)
		if driver != "" && driver != "docker-container" {
			log.Warn(
				logext.Warning("docker buildx is using the ") +
					logext.Keyword(driver) +
					logext.Warning(" driver, which may cause issues with attestations when pushing images. ") +
					logext.Warning("Consider switching to the ") +
					logext.Keyword("docker-container") +
					logext.Warning(" driver. Learn more at ") +
					logext.URL("https://docs.docker.com/go/attestations/"),
			)
		}
	})
}

// getBuildxDriver returns the current buildx driver name.
func getBuildxDriver(ctx *context.Context) string {
	out, err := runCommandWithOutput(ctx, ".", "docker", "buildx", "inspect")
	if err != nil {
		// If we can't inspect, silently continue as buildx might not be available
		return ""
	}

	// Parse the output to find the Driver line
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Driver:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}
