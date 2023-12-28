package builder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var buildKitBuilders = map[string]*BuildKitBuilder{}

func buildKitBuilder(ctx *context.Context, builderConfig config.BuildKitBuilder) (*BuildKitBuilder, error) {
	builder, ok := buildKitBuilders[builderConfig.Name]
	if ok {
		return builder, nil
	}
	stdOut, strErr, err := runCommandWithOutput(ctx, "", "docker", "buildx", "inspect", builderConfig.Name)
	if err != nil {
		if strings.Contains(string(strErr), "no builder") {
			return nil, fmt.Errorf("builder %s does not exist", builderConfig.Name)
		} else {
			return nil, fmt.Errorf("failed to identify builder: %w", err)
		}
	}
	builder = &BuildKitBuilder{
		BuilderName: builderConfig.Name,
	}

	driver := "docker"
	executor := ""
	scanner := bufio.NewScanner(bytes.NewReader(stdOut))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Driver:") {
			driver = strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "Driver:"))
		}
		if strings.Contains(scanner.Text(), "org.mobyproject.buildkit.worker.executor:") {
			parts := strings.Split(scanner.Text(), ":")
			if len(parts) == 2 {
				executor = strings.TrimSpace(parts[1])
			}
		}
	}
	log.Debugf("Instantiating buildkit builder '%s' with driver %s and executor '%s'", builder.BuilderName, driver, executor)

	switch driver {
	case "docker":
		builder.storeImageNatively = true
		builder.supportsMultiPlatformsBuild = false
	case "kubernetes":
		builder.storeImageNatively = false
		builder.supportsMultiPlatformsBuild = true
	default:
		builder.storeImageNatively = false
		builder.supportsMultiPlatformsBuild = true
	}

	switch executor {
	case "containerd":
		builder.supportsMultiPlatformsBuild = true
	}

	buildKitBuilders[builder.BuilderName] = builder
	return builder, nil
}

type BuildKitBuilder struct {
	BuilderName                 string
	storeImageNatively          bool
	supportsMultiPlatformsBuild bool
}

func (BuildKitBuilder) SkipBuildIfPublish() bool {
	return true
}

func (b BuildKitBuilder) Build(ctx *context.Context, params ImageBuildParameters, importImages bool, pushImages bool, logger *log.Entry) error {
	needLoad := importImages && !b.storeImageNatively
	if !pushImages || needLoad {
		if needLoad || !b.supportsMultiPlatformsBuild {
			// If the builder does not support multi-arch builds (e.g. standard docker) or we need to load the results,
			// we trigger multiple parallel builds
			// ToDo: for now this loop is not parallel as we don't pass the syncGroup here
			for _, platform := range params.Platforms {
				logger := logger.WithField("platfrom", fmt.Sprintf("%s/%s", platform.Goos, platform.Goarch))
				logger.Info("building...")
				err := b.buildImages(ctx, params, []config.ContainerPlatform{platform}, needLoad)
				if err != nil {
					return fmt.Errorf("failed to build image for platform %v: %w", platform, err)
				}
				logger.Debug("built")
			}
		} else {
			logger.Info("building...")
			err := b.buildImages(ctx, params, params.Platforms, needLoad)
			if err != nil {
				return fmt.Errorf("failed to build image: %w", err)
			}
			logger.Debug("built")
		}
	}
	if pushImages {
		logger.Info("pushing...")
		err := b.pushImages(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to push image: %w", err)
		}
		logger.Debug("pushed")
		return nil
	}

	return nil
}

func (b BuildKitBuilder) pushImages(ctx *context.Context, params ImageBuildParameters) error {
	flags := []string{}
	flags = append(flags, params.BuildFlags...)
	flags = append(flags, params.PushFlags...)
	if !b.supportsMultiPlatformsBuild {
		if len(params.Platforms) > 1 {
			// ToDo
			return errors.New("builder does not support multi-image builds")
		}
		digest, err := push(ctx, params.BuildPath, params.Images, flags, b.BuilderName, params.Platforms)
		if err != nil {
			return err
		}
		platform := params.Platforms[0]
		for _, img := range params.Images {
			art := &artifact.Artifact{
				Type:   artifact.DockerImage,
				Name:   img,
				Path:   img,
				Goarch: platform.Goarch,
				Goos:   platform.Goos,
				Extra: map[string]interface{}{
					artifact.ExtraDigest: digest,
				},
			}
			if params.ID != "" {
				art.Extra[artifact.ExtraID] = params.ID
			}
			ctx.Artifacts.Add(art)
		}
		return nil
	}

	digest, err := push(ctx, params.BuildPath, params.Images, flags, b.BuilderName, params.Platforms)
	if err != nil {
		return err
	}
	for _, img := range params.Images {
		art := &artifact.Artifact{
			Type: artifact.DockerManifest,
			Name: img,
			Path: img,
			Extra: map[string]interface{}{
				artifact.ExtraDigest: digest,
			},
		}
		if params.ID != "" {
			art.Extra[artifact.ExtraID] = params.ID
		}
		ctx.Artifacts.Add(art)
	}
	return nil
}

func (b BuildKitBuilder) buildImages(ctx *context.Context, params ImageBuildParameters, platforms []config.ContainerPlatform, loadImages bool) error {
	digest, err := build(ctx, params.BuildPath, params.Images, params.BuildFlags, b.BuilderName, platforms, loadImages)
	if err != nil {
		return err
	}
	if digest == "" {
		return nil
	}
	if !b.supportsMultiPlatformsBuild {
		// Case of non-multiplatform builders
		for _, img := range params.Images {
			art := &artifact.Artifact{
				Type:   artifact.PublishableDockerImage,
				Name:   img,
				Path:   img,
				Goarch: platforms[0].Goarch,
				Goos:   platforms[0].Goos,
				Extra: map[string]interface{}{
					artifact.ExtraDigest: digest,
				},
			}
			if params.ID != "" {
				art.Extra[artifact.ExtraID] = params.ID
			}
			ctx.Artifacts.Add(art)
		}
	} else {
		for _, img := range params.Images {
			art := &artifact.Artifact{
				Type: artifact.DockerManifest,
				Name: img,
				Path: img,
				Extra: map[string]interface{}{
					artifact.ExtraDigest: digest,
				},
			}
			if params.ID != "" {
				art.Extra[artifact.ExtraID] = params.ID
			}
			ctx.Artifacts.Add(art)
		}
	}
	return nil
}

func parseMetadata(root string) (metadataFile, error) {
	// Retrieve the manifest digest from the metadata file produced during the build
	content, err := os.OpenFile(path.Join(root, "metadata.json"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return metadataFile{}, fmt.Errorf("failed to open metadata file %s: %w", path.Join(root, "metadata.json"), err)
	}
	defer content.Close()
	output, err := io.ReadAll(content)
	if err != nil {
		return metadataFile{}, fmt.Errorf("failed to read metadata file: %w", err)
	}
	metadata := metadataFile{}
	if err := json.Unmarshal(output, &metadata); err != nil {
		return metadataFile{}, fmt.Errorf("failed to unmarshall metadata file: %w", err)
	}
	return metadata, nil
}

func build(ctx *context.Context, root string, images, buildFlags []string, builder string, platforms []config.ContainerPlatform, loadImages bool) (string, error) {
	command := buildCommand(root, images, buildFlags, builder, platforms, loadImages)
	err := runCommand(ctx, root, "docker", command...)
	if err != nil {
		return "", fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	metadata, err := parseMetadata(root)
	if err != nil {
		return "", err
	}
	return metadata.ContainerImageDigest, nil
}

type metadataFile struct {
	ImageName            string `json:"image.name"` // DEPRECATED: inconsistent support in BuildKit.
	ContainerImageDigest string `json:"containerimage.digest"`
}

func push(ctx *context.Context, root string, images, flags []string, builder string, platforms []config.ContainerPlatform) (string, error) {
	command := pushCommand(root, images, flags, builder, platforms)
	err := runCommand(ctx, root, "docker", command...)
	if err != nil {
		return "", fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	metadata, err := parseMetadata(root)
	if err != nil {
		return "", err
	}
	return metadata.ContainerImageDigest, nil
}

func baseCommand(root string, images, flags []string, platforms []config.ContainerPlatform, builderName string) []string {
	cmd := []string{"buildx"}
	if builderName != "" {
		cmd = append(cmd, "--builder", builderName)
	}
	cmd = append(cmd, "build", ".")
	platformNames := []string{}
	for _, platform := range platforms {
		platformNames = append(platformNames, fmt.Sprintf("%s/%s", platform.Goos, platform.Goarch))
	}
	cmd = append(cmd, "--platform", strings.Join(platformNames, ","))
	for _, image := range images {
		cmd = append(cmd, "-t", image)
	}
	cmd = append(cmd, "--metadata-file", root+"/metadata.json")
	cmd = append(cmd, flags...)
	return cmd
}

func buildCommand(root string, images, buildFlags []string, builder string, platforms []config.ContainerPlatform, loadImages bool) []string {
	cmd := baseCommand(root, images, buildFlags, platforms, builder)
	if loadImages {
		cmd = append(cmd, "--load")
	}
	return cmd
}

func pushCommand(root string, images, buildAndPushFlags []string, builder string, platforms []config.ContainerPlatform) []string {
	cmd := baseCommand(root, images, buildAndPushFlags, platforms, builder)
	cmd = append(cmd, "--push")
	return cmd
}
