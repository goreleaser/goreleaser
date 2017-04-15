// Package checksums provides a Pipe that creates .checksums files for
// each artifact.
package checksums

import (
	"fmt"
	"io/ioutil"
	"log"
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
	var g errgroup.Group
	for _, artifact := range ctx.Artifacts {
		artifact := artifact
		g.Go(func() error {
			return checksums(ctx, artifact)
		})
	}
	return g.Wait()
}

func checksums(ctx *context.Context, name string) error {
	log.Println("Checksumming", name)
	var artifact = filepath.Join(ctx.Config.Dist, name)
	sha, err := checksum.SHA256(artifact)
	if err != nil {
		return err
	}
	var file = filepath.Join(
		ctx.Config.Dist,
		fmt.Sprintf("%v.%v", name, "checksums"),
	)
	var content = fmt.Sprintf("%v\t%v\n", sha, name)
	if err := ioutil.WriteFile(file, []byte(content), 0644); err != nil {
		return err
	}
	ctx.AddArtifact(file)
	return nil
}
