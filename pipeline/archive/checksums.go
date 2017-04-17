package archive

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/sha256sum"
	"golang.org/x/sync/errgroup"
)

// ArchiveChecksums is a pipe for generating checksums for Archives
type ArchiveChecksums struct{}

// Description for ArchiveChecksums
func (ArchiveChecksums) Description() string {
	return "Archive SHA256 checksummer"
}

// Run creates checksums of all archives in context
func (ArchiveChecksums) Run(ctx *context.Context) error {
	var g errgroup.Group

	for _, archive := range ctx.Archives {
		archive := archive
		g.Go(func() error {
			return createChecksum(ctx, archive)
		})
	}

	return g.Wait()
}

func createChecksum(ctx *context.Context, archive string) error {
	folder := filepath.Join(
		ctx.Config.Dist,
		archive)

	fileName := archive + "." + ctx.Config.Archive.Format
	filePath := folder + "." + ctx.Config.Archive.Format
	sum, err := sha256sum.For(filePath)
	if err != nil {
		return err
	}

	checksumName := fmt.Sprintf("%v.%v", fileName, "sha256sum")
	log.Printf("Creating checksum: %v\n", checksumName)

	err = ioutil.WriteFile(filepath.Join(
		ctx.Config.Dist,
		checksumName),
		[]byte(sum),
		os.ModePerm)
	if err != nil {
		return err
	}

	ctx.AddArtifact(checksumName)
	return nil
}
