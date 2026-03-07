// Package dockerdigest provides a pipe to generate a file with docker image
// digests.
package dockerdigest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe for docker digests.
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
		artifact.ByTypes(
			artifact.DockerImageV2,
			artifact.DockerImage,
			artifact.DockerManifest,
		),
	).List()
	slices.SortFunc(images, func(a, b *artifact.Artifact) int {
		return strings.Compare(a.Name, b.Name)
	})

	var data bytes.Buffer
	for _, img := range images {
		digest := artifact.ExtraOr(*img, artifact.ExtraDigest, "")
		if idx := strings.IndexRune(digest, ':'); idx != -1 {
			digest = digest[idx+1:]
		}
		_, _ = fmt.Fprintf(&data, "%s  %s\n", digest, img.Name)
	}

	filename, err := tmpl.New(ctx).Apply(ctx.Config.DockerDigest.NameTemplate)
	if err != nil {
		return err
	}
	filename = filepath.Join(ctx.Config.Dist, filename)
	if err := os.WriteFile(filename, data.Bytes(), 0o644); err != nil {
		return fmt.Errorf("could not write image digest: %w", err)
	}

	log.WithField("path", filename).Info("written digest file")
	return nil
}
