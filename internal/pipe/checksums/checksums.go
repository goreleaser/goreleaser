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

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for checksums.
type Pipe struct{}

func (Pipe) String() string                 { return "calculating checksums" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.Config.Checksum.Disable }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	c := &ctx.Config.Checksum
	if c.NameTemplate == "" {
		if c.Split {
			c.NameTemplate = "{{ .ArtifactName }}.{{ .Algorithm }}"
		} else {
			c.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
		}
	}
	if c.Algorithm == "" {
		c.Algorithm = "sha256"
	}
	if len(c.IDs) > 0 {
		deprecate.Notice(ctx, "checksum.ids")
	}
	if len(c.ExtraFiles) > 0 {
		deprecate.Notice(ctx, "checksum.extra_files")
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	ctx.Artifacts.SetChecksummer(getChecksummer(ctx))
	return nil
}

func getChecksummer(ctx *context.Context) artifact.Checksummer {
	if ctx.Config.Checksum.Split {
		return splitChecksums(ctx)
	}
	return singleChecksum(ctx)
}

func splitChecksums(ctx *context.Context) artifact.Checksummer {
	return func(items []*artifact.Artifact) ([]*artifact.Artifact, error) {
		var checks []*artifact.Artifact
		var lock sync.Mutex

		g := semerrgroup.New(ctx.Parallelism)
		for _, art := range items {
			art := art
			g.Go(func() error {
				sum, err := art.Checksum(ctx.Config.Checksum.Algorithm)
				if err != nil {
					if errors.Is(err, artifact.ErrNotChecksummable) {
						return nil
					}
					return err
				}
				filename, err := tmpl.New(ctx).
					WithArtifact(art).
					WithExtraFields(tmpl.Fields{
						"Algorithm": ctx.Config.Checksum.Algorithm,
					}).
					Apply(ctx.Config.Checksum.NameTemplate)
				if err != nil {
					return err
				}
				filepath := filepath.Join(ctx.Config.Dist, filename)
				if err := os.WriteFile(filepath, []byte(sum), 0o644); err != nil {
					return err
				}
				lock.Lock()
				defer lock.Unlock()
				checks = append(checks, &artifact.Artifact{
					Type: artifact.Checksum,
					Path: filepath,
					Name: filename,
					Extra: artifact.Extras{
						artifact.ExtraChecksumOf: art.Name,
					},
				})
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return checks, nil
	}
}

func singleChecksum(ctx *context.Context) artifact.Checksummer {
	return func(items []*artifact.Artifact) ([]*artifact.Artifact, error) {
		filename, err := tmpl.New(ctx).Apply(ctx.Config.Checksum.NameTemplate)
		if err != nil {
			return nil, err
		}
		filepath := filepath.Join(ctx.Config.Dist, filename)

		g := semerrgroup.New(ctx.Parallelism)

		var sumLines []string
		var lock sync.Mutex

		for _, art := range items {
			art := art
			g.Go(func() error {
				sum, err := art.Checksum(ctx.Config.Checksum.Algorithm)
				if err != nil {
					if errors.Is(err, artifact.ErrNotChecksummable) {
						return nil
					}
					return err
				}
				lock.Lock()
				defer lock.Unlock()
				sumLines = append(sumLines, fmt.Sprintf("%v  %v", sum, art.Name))
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(
			filepath,
			os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0o644,
		)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// sort to ensure the signature is deterministic downstream
		sort.Sort(ByFilename(sumLines))

		if _, err := file.WriteString(strings.Join(sumLines, "\n")); err != nil {
			return nil, err
		}

		return []*artifact.Artifact{
			{
				Type: artifact.Checksum,
				Path: filepath,
				Name: filename,
			},
		}, nil
	}
}

// ByFilename implements sort.Interface for []string based on
// the filename of a checksum line ("{checksum}  {filename}\n")
type ByFilename []string

func (s ByFilename) Len() int      { return len(s) }
func (s ByFilename) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByFilename) Less(i, j int) bool {
	return strings.Split(s[i], "  ")[1] < strings.Split(s[j], "  ")[1]
}
