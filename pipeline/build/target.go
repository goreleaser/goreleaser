package build

import (
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/internal/buildtarget"
)

func buildTargets(build config.Build) (targets []buildtarget.Target) {
	for _, target := range allBuildTargets(build) {
		if !valid(target) {
			log.WithField("target", target.String()).
				Warn("skipped invalid build")
			continue
		}
		if ignored(build, target) {
			log.WithField("target", target.String()).
				Warn("skipped ignored build")
			continue
		}
		targets = append(targets, target)
	}
	return
}

func allBuildTargets(build config.Build) (targets []buildtarget.Target) {
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			if goarch == "arm" {
				for _, goarm := range build.Goarm {
					targets = append(targets, buildtarget.New(goos, goarch, goarm))
				}
				continue
			}
			targets = append(targets, buildtarget.New(goos, goarch, ""))
		}
	}
	return
}

func ignored(build config.Build, target buildtarget.Target) bool {
	for _, ig := range build.Ignore {
		var ignored = buildtarget.New(ig.Goos, ig.Goarch, ig.Goarm)
		if ignored == target {
			return true
		}
	}
	return false
}

func valid(target buildtarget.Target) bool {
	var s = target.OS + target.Arch
	for _, a := range valids {
		if a == s {
			return true
		}
	}
	return false
}
