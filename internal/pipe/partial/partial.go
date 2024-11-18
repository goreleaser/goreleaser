package partial

import (
	"cmp"
	"os"
	"runtime"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/experimental"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string                 { return "partial" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Partial }

func (Pipe) Run(ctx *context.Context) error {
	ctx.PartialTarget = getFilter()
	return nil
}

func getFilter() string {
	goos := cmp.Or(os.Getenv("GGOOS"), os.Getenv("GOOS"), runtime.GOOS)
	goarch := cmp.Or(os.Getenv("GGOARCH"), os.Getenv("GOARCH"), runtime.GOARCH)
	target := goos + "_" + goarch

	if strings.HasSuffix(target, "_amd64") {
		goamd64 := cmp.Or(os.Getenv("GGOAMD64"), os.Getenv("GOAMD64"), "v1")
		target = target + "_" + goamd64
	}
	if strings.HasSuffix(target, "_arm") {
		goarm := cmp.Or(os.Getenv("GGOARM"), os.Getenv("GOARM"), experimental.DefaultGOARM())
		target = target + "_" + goarm
	}
	if strings.HasSuffix(target, "_arm64") {
		goarm := cmp.Or(os.Getenv("GGOARM64"), os.Getenv("GOARM64"), "v8.0")
		target = target + "_" + goarm
	}
	if strings.HasSuffix(target, "_386") {
		goarm := cmp.Or(os.Getenv("GGO386"), os.Getenv("GO386"), "sse2")
		target = target + "_" + goarm
	}
	if strings.HasSuffix(target, "_ppc64") {
		goarm := cmp.Or(os.Getenv("GGOPPC64"), os.Getenv("GOPPC64"), "power8")
		target = target + "_" + goarm
	}
	if strings.HasSuffix(target, "_riscv64") {
		goarm := cmp.Or(os.Getenv("GGORISCV64"), os.Getenv("GORISCV64"), "rva20u64")
		target = target + "_" + goarm
	}
	if strings.HasSuffix(target, "_mips") ||
		strings.HasSuffix(target, "_mips64") ||
		strings.HasSuffix(target, "_mipsle") ||
		strings.HasSuffix(target, "_mips64le") {
		gomips := cmp.Or(os.Getenv("GGOMIPS"), os.Getenv("GOMIPS"), "hardfloat")
		target = target + "_" + gomips
	}
	return target
}
