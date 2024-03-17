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

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	artifactChecksumExtra = "Checksum"
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
	cs := &ctx.Config.Checksum
	if cs.Algorithm == "" {
		cs.Algorithm = "sha256"
	}
	if cs.NameTemplate == "" {
		if cs.Split {
			cs.NameTemplate = "{{ .ArtifactName }}.{{ .Algorithm }}"
		} else {
			cs.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
		}
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Checksum.Split {
		return splitChecksum(ctx)
	}

	return singleChecksum(ctx)
}

func splitChecksum(ctx *context.Context) error {
	artifactList, err := buildArtifactList(ctx)
	if err != nil {
		return err
	}

	for _, art := range artifactList {
		filename, err := tmpl.New(ctx).
			WithArtifact(art).
			WithExtraFields(tmpl.Fields{
				"Algorithm": ctx.Config.Checksum.Algorithm,
			}).
			Apply(ctx.Config.Checksum.NameTemplate)
		if err != nil {
			return fmt.Errorf("checksum: name template: %w", err)
		}
		filepath := filepath.Join(ctx.Config.Dist, filename)
		if err := refreshOne(ctx, *art, filepath); err != nil {
			return fmt.Errorf("checksum: %s: %w", art.Path, err)
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.Checksum,
			Path: filepath,
			Name: filename,
			Extra: map[string]interface{}{
				artifact.ExtraChecksumOf: art.Path,
				artifact.ExtraRefresh: func() error {
					log.WithField("file", filename).Info("refreshing checksums")
					return refreshOne(ctx, *art, filepath)
				},
			},
		})
	}
	return nil
}

func singleChecksum(ctx *context.Context) error {
	filename, err := tmpl.New(ctx).Apply(ctx.Config.Checksum.NameTemplate)
	if err != nil {
		return err
	}
	filepath := filepath.Join(ctx.Config.Dist, filename)
	if err := refreshAll(ctx, filepath); err != nil {
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
				return refreshAll(ctx, filepath)
			},
		},
	})
	return nil
}

func refreshOne(ctx *context.Context, art artifact.Artifact, path string) error {
	check, err := art.Checksum(ctx.Config.Checksum.Algorithm)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(check), 0o644)
}

func refreshAll(ctx *context.Context, filepath string) error {
	lock.Lock()
	defer lock.Unlock()

	artifactList, err := buildArtifactList(ctx)
	if err != nil {
		return err
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
	sort.Sort(ByFilename(sumLines))
	_, err = file.WriteString(strings.Join(sumLines, ""))
	return err
}

func buildArtifactList(ctx *context.Context) ([]*artifact.Artifact, error) {
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
		return nil, err
	}

	for name, path := range extraFiles {
		artifactList = append(artifactList, &artifact.Artifact{
			Name: name,
			Path: path,
			Type: artifact.UploadableFile,
		})
	}

	if len(artifactList) == 0 {
		return nil, errNoArtifacts
	}
	return artifactList, nil
}

func checksums(algorithm string, a *artifact.Artifact) (string, error) {
	log.WithField("file", a.Name).Debug("checksumming")
	sha, err := a.Checksum(algorithm)
	if err != nil {
		return "", err
	}

	if a.Extra == nil {
		a.Extra = make(artifact.Extras)
	}
	a.Extra[artifactChecksumExtra] = fmt.Sprintf("%s:%s", algorithm, sha)

	return fmt.Sprintf("%v  %v\n", sha, a.Name), nil
}

// ByFilename implements sort.Interface for []string based on
// the filename of a checksum line ("{checksum}  {filename}\n")
type ByFilename []string

func (s ByFilename) Len() int      { return len(s) }
func (s ByFilename) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByFilename) Less(i, j int) bool {
	return strings.Split(s[i], "  ")[1] < strings.Split(s[j], "  ")[1]
}
