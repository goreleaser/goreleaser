package reportsizes

import (
	"os"

	"github.com/caarlos0/log"
	"github.com/docker/go-units"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.ReportSizes }
func (Pipe) String() string                 { return "size reports" }

func (Pipe) Run(ctx *context.Context) error {
	return ctx.Artifacts.Filter(artifact.Or(
		artifact.ByType(artifact.Binary),
		artifact.ByType(artifact.UniversalBinary),
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.PublishableSnapcraft),
		artifact.ByType(artifact.LinuxPackage),
		artifact.ByType(artifact.CArchive),
		artifact.ByType(artifact.CShared),
		artifact.ByType(artifact.Header),
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
