// Package checksums provides a Pipe that creates .checksums files for
// each artifact.
package checksums

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for checksums
type Pipe struct{}

func (Pipe) String() string {
	return "calculating checksums"
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
		ctx.AddArtifact(file.Name())
	}()
	var g errgroup.Group
	for _, artifact := range ctx.Artifacts {
		artifact := artifact
		g.Go(func() error {
			return checksums(ctx, file, artifact)
		})
	}
	return g.Wait()
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Checksum.NameTemplate == "" {
		ctx.Config.Checksum.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
	}
	return nil
}

func checksums(ctx *context.Context, file *os.File, name string) error {
	log.WithField("file", name).Info("checksumming")
	var artifact = filepath.Join(ctx.Config.Dist, name)
	sha, err := checksum.SHA256(artifact)
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%v  %v\n", sha, name))
	return err
}
