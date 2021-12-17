// Package extrafiles provides a Pipe that adds extra pre-existing files
package extrafiles

import (
	"fmt"
	"os"

	"github.com/goreleaser/goreleaser/internal/artifact"
	intextrafiles "github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for extra files.
type Pipe struct{}

// String returns the description of the pipe.
func (Pipe) String() string                 { return "extra files" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.ExtraFiles) == 0 }

// Add extra file artifacts.
func (Pipe) Run(ctx *context.Context) error {
	extraFiles, err := intextrafiles.Find(ctx, ctx.Config.ExtraFiles)
	if err != nil {
		return err
	}

	for name, path := range extraFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("failed to add extra file %s: %w", name, err)
		}

		ctx.Artifacts.Add(&artifact.Artifact{
			Name: name,
			Path: path,
			Type: artifact.UploadableFile,
		})
	}

	return nil
}
