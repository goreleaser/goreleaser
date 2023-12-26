package buildkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// nolint: unparam
func runCommand(ctx *context.Context, dir, binary string, args ...string) error {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)

	log.
		WithField("cmd", append([]string{binary}, args[0])).
		WithField("cwd", dir).
		WithField("args", args[1:]).Debug("running")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, b.String())
	}
	return nil
}

func runCommandWithOutput(ctx *context.Context, dir, binary string, args ...string) ([]byte, error) {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)

	var b bytes.Buffer
	w := gio.Safe(&b)

	log.
		WithField("cmd", append([]string{binary}, args[0])).
		WithField("cwd", dir).
		WithField("args", args[1:]).
		Debug("running")
	out, err := cmd.CombinedOutput()
	if out != nil {
		// regardless of command success, always print stdout for backward-compatibility with runCommand()
		_, _ = io.MultiWriter(logext.NewWriter(), w).Write(out)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, b.String())
	}

	return out, nil
}

var dockerBuildDigestPattern = regexp.MustCompile("writing image (sha256:[a-z0-9]{64})")

func build(ctx *context.Context, root string, images, buildFlags []string, builder string, platforms []config.DockerPlatform, loadImages bool) (string, error) {
	command := buildCommand(root, images, buildFlags, builder, platforms, loadImages)
	out, err := runCommandWithOutput(ctx, root, "docker", command...)
	if err != nil {
		return "", fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	if !loadImages {
		return "", nil
	}
	digest := dockerBuildDigestPattern.FindStringSubmatch(string(out))
	if len(digest) < 2 {
		return "", fmt.Errorf("failed to find docker digest in docker build output: %s", string(out))
	}
	return digest[1], nil
}

type metadataFile struct {
	ImageName            string `json:"image.name"` // DEPRECATED: inconsistent support in BuildKit.
	ContainerImageDigest string `json:"containerimage.digest"`
}

func push(ctx *context.Context, root string, images, flags []string, builder string, platforms []config.DockerPlatform) (string, error) {
	command := pushCommand(root, images, flags, builder, platforms)
	err := runCommand(ctx, root, "docker", command...)
	if err != nil {
		return "", fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	// Retrieve the manifest digest from the metadata file produced during the build
	content, err := os.OpenFile(root+"/metadata.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to find docker digest in docker build output: %w", err)
	}
	defer content.Close()
	output, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to find docker digest in docker build output: %w", err)
	}
	metadata := metadataFile{}
	if err := json.Unmarshal(output, &metadata); err != nil {
		return "", fmt.Errorf("failed to find docker digest in docker build output: %w", err)
	}
	return metadata.ContainerImageDigest, nil
}

func baseCommand(root string, images, flags []string, platforms []config.DockerPlatform, builderName string) []string {
	cmd := []string{"buildx"}
	if builderName != "" {
		cmd = append(cmd, "--builder", builderName)
	}
	cmd = append(cmd, "build", ".")
	platformNames := []string{}
	for _, platform := range platforms {
		platformNames = append(platformNames, fmt.Sprintf("%s/%s", platform.Os, platform.Arch))
	}
	cmd = append(cmd, "--platform", strings.Join(platformNames, ","))
	for _, image := range images {
		cmd = append(cmd, "-t", image)
	}
	cmd = append(cmd, flags...)
	return cmd
}

func buildCommand(root string, images, buildFlags []string, builder string, platforms []config.DockerPlatform, loadImages bool) []string {
	cmd := baseCommand(root, images, buildFlags, platforms, builder)
	if loadImages {
		cmd = append(cmd, "--load")
	}
	return cmd
}

func pushCommand(root string, images, buildAndPushFlags []string, builder string, platforms []config.DockerPlatform) []string {
	cmd := baseCommand(root, images, buildAndPushFlags, platforms, builder)
	cmd = append(cmd, "--push")
	cmd = append(cmd, "--metadata-file", root+"/metadata.json")
	return cmd
}
