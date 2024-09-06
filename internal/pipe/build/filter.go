package build

import (
	"fmt"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

func filter(ctx *context.Context, targets []string) []string {
	if !ctx.Partial {
		return targets
	}

	target := ctx.PartialTarget
	log.WithField("match", fmt.Sprintf("target=%s", target)).Infof("partial build")

	var result []string
	for _, t := range targets {
		if t != target {
			continue
		}
		result = append(result, t)
		break
	}
	return result
}
