package build

import (
	"fmt"
	"log"
	"runtime"

	"github.com/goreleaser/goreleaser/context"
)

var runtimeTarget = buildTarget{runtime.GOOS, runtime.GOARCH, ""}

// a build target
type buildTarget struct {
	goos, goarch, goarm string
}

func (t buildTarget) String() string {
	return fmt.Sprintf("%v%v%v", t.goos, t.goarch, t.goarm)
}

func allBuildTargets(ctx *context.Context) (targets []buildTarget) {
	for _, goos := range ctx.Config.Build.Goos {
		for _, goarch := range ctx.Config.Build.Goarch {
			targets = append(targets, armTargets(ctx, goos, goarch)...)
			var target = buildTarget{goos, goarch, ""}
			if shouldBuild(ctx, target) {
				continue
			}
			targets = append(targets, buildTarget{goos, goarch, ""})
		}
	}
	return
}

func armTargets(ctx *context.Context, goos, goarch string) (targets []buildTarget) {
	if goarch != "arm" {
		return
	}
	for _, goarm := range ctx.Config.Build.Goarm {
		var target = buildTarget{goos, goarch, goarm}
		if shouldBuild(ctx, target) {
			continue
		}
		targets = append(targets, target)
	}
	return
}

func shouldBuild(ctx *context.Context, target buildTarget) bool {
	return !isValid(target) || isIgnored(ctx, target)
}

func isIgnored(ctx *context.Context, target buildTarget) bool {
	for _, ignore := range ctx.Config.Build.Ignore {
		var ignoredTarget = buildTarget{
			goos:   ignore.Goos,
			goarch: ignore.Goarch,
			goarm:  ignore.Goarm,
		}
		if ignoredTarget == target {
			log.Printf(
				"Skipped ignored build: %v %v %v\n",
				target.goos,
				target.goarch,
				target.goarm,
			)
			return true
		}
	}
	return false
}

func isValid(target buildTarget) bool {
	for _, valid := range validBuildTargets {
		if valid == target {
			return true
		}
	}
	log.Printf(
		"Skipped invalid build: %v %v %v\n",
		target.goos,
		target.goarch,
		target.goarm,
	)
	return false
}
