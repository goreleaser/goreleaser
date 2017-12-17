// Package checksums provides a Pipe that creates .checksums files for
// each artifact.
package checksums

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

// Pipe for checksums
type Pipe struct{}

func (Pipe) String() string {
	return "calculating checksums"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Checksum.NameTemplate == "" {
		ctx.Config.Checksum.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	filename, err := filenameFor(ctx)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, filename),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0444,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Errorf("failed to close %s", file.Name())
		}
		ctx.Artifacts.Add(artifact.Artifact{
			Type: artifact.Checksum,
			Path: file.Name(),
			Name: filename,
		})
	}()
	// TODO: parallelism should be considered here as well.
	var g errgroup.Group
	var artifacts []artifact.Artifact
	for _, t := range []artifact.Type{
		artifact.UploadableArchive,
		artifact.UploadableBinary,
	} {
		artifacts = append(artifacts, ctx.Artifacts.Filter(artifact.ByType(t)).List()...)
	}
	for _, artifact := range artifacts {
		artifact := artifact
		g.Go(func() error {
			return checksums(ctx, file, artifact)
		})
	}
	return g.Wait()
}

func checksums(ctx *context.Context, file *os.File, artifact artifact.Artifact) error {
	log.WithField("file", artifact.Name).Info("checksumming")
	sha, err := checksum.SHA256(artifact.Path)
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%v  %v\n", sha, artifact.Name))
	return err
}
