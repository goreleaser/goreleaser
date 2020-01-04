package golang

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

type target struct {
	os, arch, arm, mips string
}

func (t target) String() string {
	if t.arm != "" {
		return fmt.Sprintf("%s_%s_%s", t.os, t.arch, t.arm)
	}
	if t.mips != "" {
		return fmt.Sprintf("%s_%s_%s", t.os, t.arch, t.mips)
	}
	return fmt.Sprintf("%s_%s", t.os, t.arch)
}

func matrix(build config.Build) (result []string) {
	// nolint:prealloc
	var targets []target
	for _, target := range allBuildTargets(build) {
		if !valid(target) {
			log.WithField("target", target).
				Debug("skipped invalid build")
			continue
		}
		if ignored(build, target) {
			log.WithField("target", target).
				Debug("skipped ignored build")
			continue
		}
		targets = append(targets, target)
	}
	for _, target := range targets {
		result = append(result, target.String())
	}
	return
}

func allBuildTargets(build config.Build) (targets []target) {
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			if goarch == "arm" {
				for _, goarm := range build.Goarm {
					targets = append(targets, target{
						os:   goos,
						arch: goarch,
						arm:  goarm,
					})
				}
				continue
			}
			if strings.HasPrefix(goarch, "mips") {
				for _, gomips := range build.Gomips {
					targets = append(targets, target{
						os:   goos,
						arch: goarch,
						mips: gomips,
					})
				}
				continue
			}
			targets = append(targets, target{
				os:   goos,
				arch: goarch,
			})
		}
	}
	return
}

// TODO: this could be improved by using a map
// https://github.com/goreleaser/goreleaser/pull/522#discussion_r164245014
func ignored(build config.Build, target target) bool {
	for _, ig := range build.Ignore {
		if ig.Goos != "" && ig.Goos != target.os {
			continue
		}
		if ig.Goarch != "" && ig.Goarch != target.arch {
			continue
		}
		if ig.Goarm != "" && ig.Goarm != target.arm {
			continue
		}
		if ig.Gomips != "" && ig.Gomips != target.mips {
			continue
		}
		return true
	}
	return false
}

func valid(target target) bool {
	var s = target.os + target.arch
	for _, a := range validTargets {
		if a == s {
			return true
		}
	}
	return false
}

// list from https://golang.org/doc/install/source#environment
// nolint: gochecknoglobals
var validTargets = []string{
	"aixppc64",
	"android386",
	"androidamd64",
	"androidarm",
	"androidarm64",
	"darwin386",
	"darwinamd64",
	// "darwinarm", - requires admin rights and other ios stuff
	// "darwinarm64", - requires admin rights and other ios stuff
	"dragonflyamd64",
	"freebsd386",
	"freebsdamd64",
	"freebsdarm",
	"illumosamd64",
	"jswasm",
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
	"plan9arm",
	"solarisamd64",
	"windows386",
	"windowsamd64",
}
