package docker

import (
	"cmp"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/avast/retry-go/v4"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// ManifestPipe is an implementation for the docker manifest feature,
// allowing to publish multi-arch docker images.
type ManifestPipe struct{}

func (ManifestPipe) String() string { return "docker manifests" }

func (ManifestPipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.DockerManifests) == 0 || skips.Any(ctx, skips.Docker)
}

func (ManifestPipe) Dependencies(ctx *context.Context) []string {
	var cmds []string
	for _, s := range ctx.Config.DockerManifests {
		switch s.Use {
		case useDocker, useBuildx:
			cmds = append(cmds, "docker")
			// TODO: check buildx
		}
	}
	return cmds
}

// Default sets the pipe defaults.
func (ManifestPipe) Default(ctx *context.Context) error {
	var warnOnce sync.Once
	ids := ids.New("docker_manifests")
	for i := range ctx.Config.DockerManifests {
		// TODO: properly deprecate this.
		warnOnce.Do(func() {
			log.Warn(
				logext.Keyword("dockers") +
					logext.Warning(" and ") +
					logext.Keyword("docker_manifests") +
					logext.Warning(" are being phased out and will eventually be replaced by ") +
					logext.Keyword("dockers_v2") +
					logext.Warning(", check ") +
					logext.URL("https://goreleaser.com/deprecations#dockers") +
					logext.Warning(" for more info"),
			)
		})
		manifest := &ctx.Config.DockerManifests[i]
		if manifest.ID != "" {
			ids.Inc(manifest.ID)
		}
		if manifest.Use == "" {
			manifest.Use = useDocker
		}
		manifest.Retry.Attempts = cmp.Or(manifest.Retry.Attempts, 10)
		manifest.Retry.Delay = cmp.Or(manifest.Retry.Delay, 10*time.Second)
		manifest.Retry.MaxDelay = cmp.Or(manifest.Retry.MaxDelay, 5*time.Minute)
		if err := validateManifester(manifest.Use); err != nil {
			return err
		}
	}
	return ids.Validate()
}

// Publish the docker manifests.
func (ManifestPipe) Publish(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(1))
	for _, manifest := range ctx.Config.DockerManifests {
		g.Go(func() error {
			skip, err := tmpl.New(ctx).Apply(manifest.SkipPush)
			if err != nil {
				return err
			}
			if strings.TrimSpace(skip) == "true" {
				return pipe.Skip("docker_manifest.skip_push is set")
			}

			if strings.TrimSpace(skip) == "auto" && ctx.Semver.Prerelease != "" {
				return pipe.Skip("prerelease detected with 'auto' push, skipping docker manifest")
			}

			name, err := manifestName(ctx, manifest)
			if err != nil {
				return err
			}

			images, err := manifestImages(ctx, manifest)
			if err != nil {
				return err
			}

			manifester := manifesters[manifest.Use]
			if err := retry.Do(
				func() error {
					log.WithField("manifest", name).
						WithField("images", images).
						Info("creating manifest")
					return manifester.Create(ctx, name, images, manifest.CreateFlags)
				},
				retry.RetryIf(isRetriableManifestCreate),
				retry.Attempts(manifest.Retry.Attempts),
				retry.Delay(manifest.Retry.Delay),
				retry.MaxDelay(manifest.Retry.MaxDelay),
				retry.LastErrorOnly(true),
			); err != nil {
				return err
			}
			art := &artifact.Artifact{
				Type:  artifact.DockerManifest,
				Name:  name,
				Path:  name,
				Extra: map[string]any{},
			}
			if manifest.ID != "" {
				art.Extra[artifact.ExtraID] = manifest.ID
			}

			digest, err := retry.DoWithData(
				func() (string, error) {
					log.WithField("manifest", name).Info("pushing manifest")
					return manifester.Push(ctx, name, manifest.PushFlags)
				},
				retry.RetryIf(isRetriablePush),
				retry.Attempts(manifest.Retry.Attempts),
				retry.Delay(manifest.Retry.Delay),
				retry.MaxDelay(manifest.Retry.MaxDelay),
				retry.LastErrorOnly(true),
			)
			if err != nil {
				return err
			}

			log.WithField("image", name).
				WithField("digest", digest).
				Info("artifact pushed")

			art.Extra[artifact.ExtraDigest] = digest
			ctx.Artifacts.Add(art)
			return nil
		})
	}
	return g.Wait()
}

func validateManifester(use string) error {
	if _, ok := manifesters[use]; ok {
		return nil
	}
	return fmt.Errorf("docker manifest: invalid use: %s, valid options are %v", use, slices.Sorted(maps.Keys(manifesters)))
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
	artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List()
	imgs := make([]string, 0, len(manifest.ImageTemplates))
	for _, img := range manifest.ImageTemplates {
		str, err := tmpl.New(ctx).Apply(img)
		if err != nil {
			return []string{}, err
		}

		if str != "" {
			imgs = append(imgs, withDigest(str, artifacts))
		}
	}
	if strings.TrimSpace(strings.Join(manifest.ImageTemplates, "")) == "" {
		return imgs, pipe.Skip("manifest has no images")
	}
	return imgs, nil
}

func withDigest(name string, images []*artifact.Artifact) string {
	for _, art := range images {
		if art.Name == name {
			if digest := artifact.ExtraOr(*art, artifact.ExtraDigest, ""); digest != "" {
				return name + "@" + digest
			}
			log.Warnf("unknown digest for %q, using insecure mode", name)
			return name
		}
	}

	suggestion := ""
	suggestionDistance := math.MaxInt
	for _, img := range images {
		if d := levenshtein.ComputeDistance(name, img.Name); d < suggestionDistance {
			suggestion = name
			suggestionDistance = d
		}
	}

	log.Warnf("could not find %q, did you mean %q?", name, suggestion)
	return name
}

func isRetriableManifestCreate(err error) bool {
	return strings.Contains(err.Error(), "manifest verification failed for digest")
}
