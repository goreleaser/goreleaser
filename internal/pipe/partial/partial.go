package partial

import (
	"os"
	"runtime"

	"github.com/goreleaser/goreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string                 { return "partial" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Partial }

func (Pipe) Run(ctx *context.Context) error {
	ctx.PartialTarget = getFilter()
	return nil
}

func getFilter() string {
	goos := firstNonEmpty(os.Getenv("GGOOS"), os.Getenv("GOOS"), runtime.GOOS)
	goarch := firstNonEmpty(os.Getenv("GGOARCH"), os.Getenv("GOARCH"), runtime.GOARCH)
	return goos + "_" + goarch
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
