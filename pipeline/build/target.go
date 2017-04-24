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
			if !valid(goos, goarch) {
				log.Printf("Skipped build for %v/%v\n", goos, goarch)
				continue
			}
			if goarch == "arm" {
				for _, goarm := range ctx.Config.Build.Goarm {
					targets = append(targets, buildTarget{goos, goarch, goarm})
				}
				continue
			}
			targets = append(targets, buildTarget{goos, goarch, ""})
		}
	}
	return
}
