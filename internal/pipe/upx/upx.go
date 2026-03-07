// Package upx compresses binaries using upx.
package upx

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/caarlos0/log"
	"github.com/docker/go-units"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string                         { return "upx" }
func (Pipe) Skip(ctx *context.Context) bool         { return len(ctx.Config.UPXs) == 0 }
func (Pipe) Dependencies(*context.Context) []string { return []string{"upx"} }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.UPXs {
		upx := &ctx.Config.UPXs[i]
		if upx.Binary == "" {
			upx.Binary = "upx"
		}
	}
	return nil
}

func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	skips := pipe.SkipMemento{}
	for _, upx := range ctx.Config.UPXs {
		enabled, err := tmpl.New(ctx).Bool(upx.Enabled)
		if err != nil {
			return err
		}
		if !enabled {
			skips.Remember(pipe.Skip("upx is not enabled"))
			continue
		}
		if _, err := exec.LookPath(upx.Binary); err != nil {
			skips.Remember(pipe.Skipf("%s not found in PATH", upx.Binary))
			continue
		}
		for _, bin := range findBinaries(ctx, upx) {
			g.Go(func() error {
				return compressOne(ctx, upx, bin)
			})
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return skips.Evaluate()
}

func compressOne(ctx *context.Context, upx config.UPX, bin *artifact.Artifact) error {
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
	out, err := exec.CommandContext(ctx, upx.Binary, args...).CombinedOutput()
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
}

var knownExceptions = []string{
	"CantPackException",
	"AlreadyPackedException",
	"NotCompressibleException",
	"UnknownExecutableFormatException",
	"IOException",
}

func findBinaries(ctx *context.Context, upx config.UPX) []*artifact.Artifact {
	filters := []artifact.Filter{
		artifact.ByTypes(
			artifact.Binary,
			artifact.UniversalBinary,
		),
		artifact.ByGooses(upx.Goos...),
		artifact.ByGoarches(upx.Goarch...),
		artifact.ByGoarms(upx.Goarm...),
		artifact.ByGoamd64s(upx.Goamd64...),
		artifact.ByIDs(upx.IDs...),
	}
	return ctx.Artifacts.Filter(artifact.And(filters...)).List()
}

func sizeOf(name string) int64 {
	st, err := os.Stat(name)
	if err != nil {
		return 0
	}
	return st.Size()
}
