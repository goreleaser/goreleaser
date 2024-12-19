package partial

import (
	"cmp"
	"errors"
	"os"
	"runtime"
	"strings"

	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string                 { return "partial" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Partial }

func (Pipe) Run(ctx *context.Context) error {
	if t := os.Getenv("TARGET"); t != "" {
		ctx.PartialTarget = t
		return nil
	}

	for _, b := range ctx.Config.Builds {
		if b.Builder == "go" {
			ctx.PartialTarget = getGoEnvFilter()
		}
	}

	if ctx.PartialTarget == "" {
		return errors.New("could not setup the target filter, maybe set TARGET=[something]")
	}
	return nil
}

var archExtraEnvs = map[string][]string{
	"386":      {"GGO386", "GO386"},
	"amd64":    {"GGOAMD64", "GOAMD64"},
	"arm":      {"GGOARM", "GOARM"},
	"arm64":    {"GGOARM64", "GOARM64"},
	"mips":     {"GGOMIPS", "GOMIPS"},
	"mips64":   {"GGOMIPS", "GOMIPS"},
	"mips64le": {"GGOMIPS", "GOMIPS"},
	"mipsle":   {"GGOMIPS", "GOMIPS"},
	"ppc64":    {"GGOPPC64", "GOPPC64"},
	"riscv64":  {"GGORISCV64", "GORISCV64"},
}

func getGoEnvFilter() string {
	goos := cmp.Or(os.Getenv("GGOOS"), os.Getenv("GOOS"), runtime.GOOS)
	goarch := cmp.Or(os.Getenv("GGOARCH"), os.Getenv("GOARCH"), runtime.GOARCH)

	target := goos + "_" + goarch
	for suffix, keys := range archExtraEnvs {
		if !strings.HasSuffix(target, "_"+suffix) {
			continue
		}
		for _, key := range keys {
			if env := os.Getenv(key); env != "" {
				target += "_" + env
				break
			}
		}
	}
	return target
}
