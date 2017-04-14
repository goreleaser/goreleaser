package checksums

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/context"
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
	var checksums = fmt.Sprintf("%v.%v", name, "checksums")
	sha, err := checksum.SHA256(artifact)
	if err != nil {
		return err
	}
	md5, err := checksum.MD5(artifact)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, checksums),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	defer func() { _ = file.Close() }()
	var template = "%v %v\n"
	if _, err = file.WriteString(fmt.Sprintf(template, "md5sum", md5)); err != nil {
		return err
	}
	if _, err = file.WriteString(fmt.Sprintf(template, "sha256sum", sha)); err != nil {
		return err
	}
	ctx.AddArtifact(file.Name())
	return nil
}
