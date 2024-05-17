package upx

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/caarlos0/log"
	"github.com/docker/go-units"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string { return "upx" }
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.UPXs {
		upx := &ctx.Config.UPXs[i]
		if upx.Binary == "" {
			upx.Binary = "upx"
		}
	}
	return nil
}
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.UPXs) == 0 }
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, upx := range ctx.Config.UPXs {
		enabled, err := tmpl.New(ctx).Bool(upx.Enabled)
		if err != nil {
			return err
		}
		if !enabled {
			return pipe.Skip("upx is not enabled")
		}
		if _, err := exec.LookPath(upx.Binary); err != nil {
			return pipe.Skipf("%s not found in PATH", upx.Binary)
		}
		for _, bin := range findBinaries(ctx, upx) {
			g.Go(func() error {
				sizeBefore := sizeOf(bin.Path)
				args := []string{
					"--quiet",
				}
				switch upx.Compress {
				case "best":
					args = append(args, "--best")
				case "":
				default:
					args = append(args, "-"+upx.Compress)
				}
				if upx.LZMA {
					args = append(args, "--lzma")
				}
				if upx.Brute {
					args = append(args, "--brute")
				}
				args = append(args, bin.Path)
				out, err := exec.CommandContext(ctx, "upx", args...).CombinedOutput()
				if err != nil {
					for _, ke := range knownExceptions {
						if strings.Contains(string(out), ke) {
							log.WithField("binary", bin.Path).
								WithField("exception", ke).
								Warn("could not pack")
							return nil
						}
					}
					return fmt.Errorf("could not pack %s: %w: %s", bin.Path, err, string(out))
				}

				sizeAfter := sizeOf(bin.Path)

				log.
					WithField("before", units.HumanSize(float64(sizeBefore))).
					WithField("after", units.HumanSize(float64(sizeAfter))).
					WithField("ratio", fmt.Sprintf("%d%%", (sizeAfter*100)/sizeBefore)).
					WithField("binary", bin.Path).
					Info("packed")

				return nil
			})
		}
	}
	return g.Wait()
}

var knownExceptions = []string{
	"CantPackException",
	"AlreadyPackedException",
	"NotCompressibleException",
}

func findBinaries(ctx *context.Context, upx config.UPX) []*artifact.Artifact {
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
		),
	}
	if f := orBy(artifact.ByGoos, upx.Goos); f != nil {
		filters = append(filters, f)
	}
	if f := orBy(artifact.ByGoarch, upx.Goarch); f != nil {
		filters = append(filters, f)
	}
	if f := orBy(artifact.ByGoarm, upx.Goarm); f != nil {
		filters = append(filters, f)
	}
	if f := orBy(artifact.ByGoamd64, upx.Goamd64); f != nil {
		filters = append(filters, f)
	}
	if len(upx.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(upx.IDs...))
	}
	return ctx.Artifacts.Filter(artifact.And(filters...)).List()
}

func orBy(fn func(string) artifact.Filter, items []string) artifact.Filter {
	var result []artifact.Filter
	for _, f := range items {
		result = append(result, fn(f))
	}
	if len(result) == 0 {
		return nil
	}
	return artifact.Or(result...)
}

func sizeOf(name string) int64 {
	st, err := os.Stat(name)
	if err != nil {
		return 0
	}
	return st.Size()
}
