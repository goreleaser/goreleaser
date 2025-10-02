// Package reportsizes reports the sizes of the artifacts.
package reportsizes

import (
	"os"

	"github.com/caarlos0/log"
	"github.com/docker/go-units"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

type Pipe struct{}

func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.ReportSizes }
func (Pipe) String() string                 { return "size reports" }

func (Pipe) Run(ctx *context.Context) error {
	return ctx.Artifacts.Filter(artifact.ByTypes(
		artifact.Binary,
		artifact.UniversalBinary,
		artifact.UploadableArchive,
		artifact.Makeself,
		artifact.PublishableSnapcraft,
		artifact.LinuxPackage,
		artifact.CArchive,
		artifact.CShared,
		artifact.Header,
	)).Visit(func(a *artifact.Artifact) error {
		stat, err := os.Stat(a.Path)
		if err != nil {
			return err
		}
		a.Extra[artifact.ExtraSize] = stat.Size()
		log.WithField("path", a.Path).
			Info(units.BytesSize(float64(stat.Size())))
		return nil
	})
}
