package build

import (
	"fmt"
	"runtime"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
)

var runtimeTarget = buildTarget{runtime.GOOS, runtime.GOARCH, ""}

// a build target
type buildTarget struct {
	goos, goarch, goarm string
}

func (t buildTarget) String() string {
	return fmt.Sprintf("%v%v%v", t.goos, t.goarch, t.goarm)
}

func (t buildTarget) PrettyString() string {
	return fmt.Sprintf("%v/%v%v", t.goos, t.goarch, t.goarm)
}

func buildTargets(build config.Build) (targets []buildTarget) {
	for _, target := range allBuildTargets(build) {
		if !valid(target) {
			log.WithField("target", target.PrettyString()).
				Warn("skipped invalid build")
			continue
		}
		if ignored(build, target) {
			log.WithField("target", target.PrettyString()).
				Warn("skipped ignored build")
			continue
		}
		targets = append(targets, target)
	}
	return
}

func allBuildTargets(build config.Build) (targets []buildTarget) {
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			if goarch == "arm" {
				for _, goarm := range build.Goarm {
					targets = append(targets, buildTarget{goos, goarch, goarm})
				}
				continue
			}
			targets = append(targets, buildTarget{goos, goarch, ""})
		}
	}
	return
}

func ignored(build config.Build, target buildTarget) bool {
	for _, ig := range build.Ignore {
		var ignored = buildTarget{ig.Goos, ig.Goarch, ig.Goarm}
		if ignored == target {
			return true
		}
	}
	return false
}

func valid(target buildTarget) bool {
	var s = target.goos + target.goarch
	for _, a := range valids {
		if a == s {
			return true
		}
	}
	return false
}
