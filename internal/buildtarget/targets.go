package buildtarget

import (
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
)

// All returns all valid build targets for a given build
func All(build config.Build) (targets []Target) {
	for _, target := range allBuildTargets(build) {
		if !valid(target) {
			log.WithField("target", target.PrettyString()).
				Debug("skipped invalid build")
			continue
		}
		if ignored(build, target) {
			log.WithField("target", target.PrettyString()).
				Debug("skipped ignored build")
			continue
		}
		targets = append(targets, target)
	}
	return
}

func allBuildTargets(build config.Build) (targets []Target) {
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			if goarch == "arm" {
				for _, goarm := range build.Goarm {
					targets = append(targets, New(goos, goarch, goarm))
				}
				continue
			}
			targets = append(targets, New(goos, goarch, ""))
		}
	}
	return
}

func ignored(build config.Build, target Target) bool {
	for _, ig := range build.Ignore {
		if ig.Goos != "" && ig.Goos != target.OS {
			continue
		}
		if ig.Goarch != "" && ig.Goarch != target.Arch {
			continue
		}
		if ig.Goarm != "" && ig.Goarm != target.Arm {
			continue
		}
		return true
	}
	return false
}

func valid(target Target) bool {
	var s = target.OS + target.Arch
	for _, a := range validTargets {
		if a == s {
			return true
		}
	}
	return false
}

// list from https://golang.org/doc/install/source#environment
var validTargets = []string{
	"androidarm",
	"darwin386",
	"darwinamd64",
	// "darwinarm", - requires admin rights and other ios stuff
	// "darwinarm64", - requires admin rights and other ios stuff
	"dragonflyamd64",
	"freebsd386",
	"freebsdamd64",
	"freebsdarm",
	"linux386",
	"linuxamd64",
	"linuxarm",
	"linuxarm64",
	"linuxppc64",
	"linuxppc64le",
	"linuxmips",
	"linuxmipsle",
	"linuxmips64",
	"linuxmips64le",
	"linuxs390x",
	"netbsd386",
	"netbsdamd64",
	"netbsdarm",
	"openbsd386",
	"openbsdamd64",
	"openbsdarm",
	"plan9386",
	"plan9amd64",
	"solarisamd64",
	"windows386",
	"windowsamd64",
}
