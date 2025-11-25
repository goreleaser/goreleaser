// Package partial handles the partial builds.
package partial

import (
	"cmp"
	"errors"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/caarlos0/log"
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
			break
		}
		ctx.PartialTarget = findRuntime(b.Targets)
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
		var sb strings.Builder
		for _, key := range keys {
			if env := os.Getenv(key); env != "" {
				sb.WriteString("_" + env)
				break
			}
		}
		target += sb.String()
	}
	return target
}

// best effort figuring out the translation tables here, we support only the most common ones...
var goosToOthers = map[string][]string{
	"darwin":  {"macos", "darwin"},
	"linux":   {"linux"},
	"windows": {"windows"},
}

var goarchToOthers = map[string][]string{
	"arm64": {"aarch64"},
	"amd64": {"x86_64"},
	"386":   {"i686", "i586", "i386"},
}

func findRuntime(targets []string) string {
	goos := cmp.Or(os.Getenv("GGOOS"), os.Getenv("GOOS"), runtime.GOOS)
	goarch := cmp.Or(os.Getenv("GGOARCH"), os.Getenv("GOARCH"), runtime.GOARCH)

	oses := goosToOthers[goos]
	arches := goarchToOthers[goarch]

	for _, target := range targets {
		parts := strings.Split(target, "-")
		if hasAny(oses, parts) && hasAny(arches, parts) {
			log.Infof("using %s based on runtime", target)
			return target
		}
	}

	// not found
	return ""
}

func hasAny(s1, s2 []string) bool {
	for _, s := range s1 {
		if slices.Contains(s2, s) {
			return true
		}
	}
	return false
}
