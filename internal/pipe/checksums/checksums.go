// Package checksums provides a Pipe that creates .checksums files for
// each artifact.
package checksums

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var (
	errNoArtifacts = errors.New("there are no artifacts to sign")
	lock           sync.Mutex
)

// Pipe for checksums.
type Pipe struct{}

func (Pipe) String() string                 { return "calculating checksums" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.Config.Checksum.Disable }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Checksum.NameTemplate == "" {
		ctx.Config.Checksum.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
	}
	if ctx.Config.Checksum.Algorithm == "" {
		ctx.Config.Checksum.Algorithm = "sha256"
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	filename, err := tmpl.New(ctx).Apply(ctx.Config.Checksum.NameTemplate)
	if err != nil {
		return err
	}
	filepath := filepath.Join(ctx.Config.Dist, filename)
	if err := refresh(ctx, filepath); err != nil {
		if errors.Is(err, errNoArtifacts) {
			return nil
		}
		return err
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Checksum,
		Path: filepath,
		Name: filename,
		Extra: map[string]interface{}{
			artifact.ExtraRefresh: func() error {
				log.WithField("file", filename).Info("refreshing checksums")
				return refresh(ctx, filepath)
			},
		},
	})
	return nil
}

func refresh(ctx *context.Context, filepath string) error {
	lock.Lock()
	defer lock.Unlock()
	filter := artifact.Or(
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableBinary),
		artifact.ByType(artifact.UploadableSourceArchive),
		artifact.ByType(artifact.LinuxPackage),
		artifact.ByType(artifact.SBOM),
	)
	if len(ctx.Config.Checksum.IDs) > 0 {
		filter = artifact.And(filter, artifact.ByIDs(ctx.Config.Checksum.IDs...))
	}

	artifactList := ctx.Artifacts.Filter(filter).List()

	extraFiles, err := extrafiles.Find(ctx, ctx.Config.Checksum.ExtraFiles)
	if err != nil {
		return err
	}

	for name, path := range extraFiles {
		artifactList = append(artifactList, &artifact.Artifact{
			Name: name,
			Path: path,
			Type: artifact.UploadableFile,
		})
	}

	if len(artifactList) == 0 {
		return errNoArtifacts
	}

	g := semerrgroup.New(ctx.Parallelism)
	sumLines := make([]string, len(artifactList))
	for i, artifact := range artifactList {
		i := i
		artifact := artifact
		g.Go(func() error {
			sumLine, err := checksums(ctx.Config.Checksum.Algorithm, artifact)
			if err != nil {
				return err
			}
			sumLines[i] = sumLine
			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(
		filepath,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return err
	}
	defer file.Close()

	// sort to ensure the signature is deterministic downstream
	sort.Strings(sumLines)
	_, err = file.WriteString(strings.Join(sumLines, ""))
	return err
}

func checksums(algorithm string, artifact *artifact.Artifact) (string, error) {
	log.WithField("file", artifact.Name).Debug("checksumming")
	sha, err := artifact.Checksum(algorithm)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v  %v\n", sha, artifact.Name), nil
}
