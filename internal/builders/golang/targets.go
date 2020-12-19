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

func matrix(build config.Build) ([]string, error) {
	// nolint:prealloc
	var targets []target
	// nolint:prealloc
	var result []string
	for _, target := range allBuildTargets(build) {
		if !contains(target.os, validGoos) {
			return result, fmt.Errorf("invalid goos: %s", target.os)
		}
		if !contains(target.arch, validGoarch) {
			return result, fmt.Errorf("invalid goarch: %s", target.arch)
		}
		if target.arm != "" && !contains(target.arm, validGoarm) {
			return result, fmt.Errorf("invalid goarm: %s", target.arm)
		}
		if target.mips != "" && !contains(target.mips, validGomips) {
			return result, fmt.Errorf("invalid gomips: %s", target.mips)
		}
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
	return result, nil
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

// TODO: this could be improved by using a map.
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
	return contains(target.os+target.arch, validTargets)
}

func contains(s string, ss []string) bool {
	for _, z := range ss {
		if z == s {
			return true
		}
	}
	return false
}

// lists from https://golang.org/doc/install/source#environment
// nolint: gochecknoglobals
var (
	validTargets = []string{
		"aixppc64",
		"android386",
		"androidamd64",
		"androidarm",
		"androidarm64",
		// "darwin386", - deprecated on latest go 1.15+
		"darwinamd64",
		// "darwinarm", - requires admin rights and other ios stuff
		"darwinarm64",
		"dragonflyamd64",
		"freebsd386",
		"freebsdamd64",
		"freebsdarm",
		"freebsdarm64", // not on the official list for some reason, yet its supported on go 1.14+
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
		"openbsdarm64",
		"plan9386",
		"plan9amd64",
		"plan9arm",
		"solarisamd64",
		"windows386",
		"windowsamd64",
	}

	validGoos = []string{
		"aix",
		"android",
		"darwin",
		"dragonfly",
		"freebsd",
		"illumos",
		"js",
		"linux",
		"netbsd",
		"openbsd",
		"plan9",
		"solaris",
		"windows",
	}

	validGoarch = []string{
		"386",
		"amd64",
		"arm",
		"arm64",
		"mips",
		"mips64",
		"mips64le",
		"mipsle",
		"ppc64",
		"ppc64le",
		"s390x",
		"wasm",
	}

	validGoarm  = []string{"5", "6", "7"}
	validGomips = []string{"hardfloat", "softfloat"}
)
