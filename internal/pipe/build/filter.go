package build

import (
	"fmt"

	"github.com/caarlos0/log"
	builders "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

func filter(ctx *context.Context, build config.Build) []string {
	if !ctx.Partial {
		return build.Targets
	}
	target := ctx.PartialTarget
	fixer, ok := builders.For(build.Builder).(builders.TargetFixer)
	if ok {
		target = fixer.FixTarget(target)
	}
	log.WithField("match", fmt.Sprintf("target=%s", target)).Infof("partial build")

	var result []string
	for _, t := range build.Targets {
		if t != target {
			continue
		}
		result = append(result, t)
		break
	}
	return result
}
