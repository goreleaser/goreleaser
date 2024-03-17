// Package buildtarget can generate a list of targets based on a matrix of
// goos, goarch, goarm, goamd64, gomips and go version.
package buildtarget

import (
	"fmt"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

type target struct {
	os, arch, arm, mips, amd64 string
}

func (t target) String() string {
	if extra := t.arm + t.mips + t.amd64; extra != "" {
		return fmt.Sprintf("%s_%s_%s", t.os, t.arch, extra)
	}
	return fmt.Sprintf("%s_%s", t.os, t.arch)
}

// List compiles the list of targets for the given builds.
func List(build config.Build) ([]string, error) {
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
		if target.amd64 != "" && !contains(target.amd64, validGoamd64) {
			return result, fmt.Errorf("invalid goamd64: %s", target.amd64)
		}
		if !valid(target) {
			log.WithField("target", target).Debug("skipped invalid build")
			continue
		}
		if ignored(build, target) {
			log.WithField("target", target).Debug("skipped ignored build")
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
			if strings.HasPrefix(goarch, "amd64") {
				for _, goamd := range build.Goamd64 {
					targets = append(targets, target{
						os:    goos,
						arch:  goarch,
						amd64: goamd,
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
		if ig.Goamd64 != "" && ig.Goamd64 != target.amd64 {
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
		"darwinamd64",
		"darwinarm64",
		"dragonflyamd64",
		"freebsd386",
		"freebsdamd64",
		"freebsdarm",
		"freebsdarm64", // not on the official list for some reason, yet its supported on go 1.14+
		"illumosamd64",
		"iosarm64",
		"jswasm",
		"wasip1wasm",
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
		"linuxriscv64",
		"linuxloong64",
		"netbsd386",
		"netbsdamd64",
		"netbsdarm",
		"netbsdarm64", // not on the official list for some reason, yet its supported on go 1.13+
		"openbsd386",
		"openbsdamd64",
		"openbsdarm",
		"openbsdarm64",
		"plan9386",
		"plan9amd64",
		"plan9arm",
		"solarisamd64",
		"windowsarm",
		"windowsarm64",
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
		"ios",
		"js",
		"linux",
		"netbsd",
		"openbsd",
		"plan9",
		"solaris",
		"windows",
		"wasip1",
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
		"riscv64",
		"loong64",
	}

	validGoarm   = []string{"5", "6", "7"}
	validGomips  = []string{"hardfloat", "softfloat"}
	validGoamd64 = []string{"v1", "v2", "v3", "v4"}
)
