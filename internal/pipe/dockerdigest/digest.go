package dockerdigest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe for checksums.
type Pipe struct{}

func (Pipe) String() string { return "docker digests" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	if skips.Any(ctx, skips.Docker) {
		return true, nil
	}
	return tmpl.New(ctx).Bool(ctx.Config.DockerDigest.Disable)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	d := &ctx.Config.DockerDigest
	if d.NameTemplate == "" {
		d.NameTemplate = "digests.txt"
	}
	return nil
}

// Publish will create the digests file.
// It doesn't actually publish anything, but it's implemented as a publisher as
// it needs to run in the publishing phase, after docker images are pushed.
func (Pipe) Publish(ctx *context.Context) error {
	images := ctx.Artifacts.Filter(
		artifact.Or(
			// artifact.ByType(artifact.DockerImageV2),
			artifact.ByType(artifact.DockerImage),
			artifact.ByType(artifact.DockerManifest),
		),
	).List()

	filename, err := tmpl.New(ctx).Apply(ctx.Config.DockerDigest.NameTemplate)
	if err != nil {
		return err
	}
	slices.SortFunc(images, func(a, b *artifact.Artifact) int {
		return strings.Compare(a.Name, b.Name)
	})

	filepath := filepath.Join(ctx.Config.Dist, filename)
	file, err := os.OpenFile(
		filepath,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("could not write image digest: %w", err)
	}
	defer file.Close()

	for _, img := range images {
		digest := artifact.ExtraOr(*img, artifact.ExtraDigest, "")
		if idx := strings.IndexRune(digest, ':'); idx != -1 {
			digest = digest[idx+1:]
		}
		if _, err := fmt.Fprintf(file, "%s %s\n", digest, img.Name); err != nil {
			return fmt.Errorf("could not write image digest: %w", err)
		}

	}

	return nil
}
