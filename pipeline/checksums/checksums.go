// Package checksums provides a Pipe that creates .checksums files for
// each artifact.
package checksums

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for checksums
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Calculating checksums"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	file, err := os.OpenFile(
		filepath.Join(
			ctx.Config.Dist,
			fmt.Sprintf("%v_checksums.txt", ctx.Config.Build.Binary),
		),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
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

func checksums(ctx *context.Context, file *os.File, name string) error {
	log.Println("Checksumming", name)
	var artifact = filepath.Join(ctx.Config.Dist, name)
	sha, err := checksum.SHA256(artifact)
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%v\t%v\n", sha, name))
	return err
}
