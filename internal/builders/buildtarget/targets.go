// Package buildtarget can generate a list of targets based on a matrix of
// goos, goarch, goarm, goamd64, gomips and go version.
package buildtarget

import (
	"fmt"
	"regexp"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

type target struct {
	os, arch, amd64, go386, arm, arm64, mips, ppc64, riscv64 string
}

func (t target) String() string {
	if extra := t.amd64 + t.go386 + t.arm + t.arm64 + t.mips + t.ppc64 + t.riscv64; extra != "" {
		return fmt.Sprintf("%s_%s_%s", t.os, t.arch, extra)
	}
	return fmt.Sprintf("%s_%s", t.os, t.arch)
}

// List compiles the list of targets for the given builds.
func List(build config.Build) ([]string, error) {
	//nolint:prealloc
	var targets []target
	//nolint:prealloc
	var result []string
	for _, target := range allBuildTargets(build) {
		if !contains(target.os, validGoos) {
			return result, fmt.Errorf("invalid goos: %s", target.os)
		}
		if !contains(target.arch, validGoarch) {
			return result, fmt.Errorf("invalid goarch: %s", target.arch)
		}
		if target.amd64 != "" && !contains(target.amd64, validGoamd64) {
			return result, fmt.Errorf("invalid goamd64: %s", target.amd64)
		}
		if target.go386 != "" && !contains(target.go386, validGo386) {
			return result, fmt.Errorf("invalid go386: %s", target.go386)
		}
		if target.arm != "" && !contains(target.arm, validGoarm) {
			return result, fmt.Errorf("invalid goarm: %s", target.arm)
		}
		if target.arm64 != "" && !validGoarm64.MatchString(target.arm64) {
			return result, fmt.Errorf("invalid goarm64: %s", target.arm64)
		}
		if target.mips != "" && !contains(target.mips, validGomips) {
			return result, fmt.Errorf("invalid gomips: %s", target.mips)
		}
		if target.ppc64 != "" && !contains(target.ppc64, validGoppc64) {
			return result, fmt.Errorf("invalid goppc64: %s", target.ppc64)
		}
		if target.riscv64 != "" && !contains(target.riscv64, validGoriscv64) {
			return result, fmt.Errorf("invalid goriscv64: %s", target.riscv64)
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
			//nolint:gocritic
			switch goarch {
			case "386":
				for _, go386 := range build.Go386 {
					targets = append(targets, target{
						os:    goos,
						arch:  goarch,
						go386: go386,
					})
				}
			case "amd64":
				for _, goamd64 := range build.Goamd64 {
					targets = append(targets, target{
						os:    goos,
						arch:  goarch,
						amd64: goamd64,
					})
				}
			case "arm64":
				for _, goarm64 := range build.Goarm64 {
					targets = append(targets, target{
						os:    goos,
						arch:  goarch,
						arm64: goarm64,
					})
				}
			case "arm":
				for _, goarm := range build.Goarm {
					targets = append(targets, target{
						os:   goos,
						arch: goarch,
						arm:  goarm,
					})
				}
			case "mips", "mipsle", "mips64", "mips64le":
				for _, gomips := range build.Gomips {
					targets = append(targets, target{
						os:   goos,
						arch: goarch,
						mips: gomips,
					})
				}
			case "ppc64":
				for _, goppc64 := range build.Goppc64 {
					targets = append(targets, target{
						os:    goos,
						arch:  goarch,
						ppc64: goppc64,
					})
				}
			case "riscv64":
				for _, goriscv64 := range build.Goriscv64 {
					targets = append(targets, target{
						os:      goos,
						arch:    goarch,
						riscv64: goriscv64,
					})
				}
			default:
				targets = append(targets, target{
					os:   goos,
					arch: goarch,
				})
			}
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
		if ig.Goamd64 != "" && ig.Goamd64 != target.amd64 {
			continue
		}
		if ig.Go386 != "" && ig.Go386 != target.go386 {
			continue
		}
		if ig.Goarm != "" && ig.Goarm != target.arm {
			continue
		}
		if ig.Goarm64 != "" && ig.Goarm64 != target.arm64 {
			continue
		}
		if ig.Gomips != "" && ig.Gomips != target.mips {
			continue
		}
		if ig.Goppc64 != "" && ig.Goppc64 != target.ppc64 {
			continue
		}
		if ig.Goriscv64 != "" && ig.Goriscv64 != target.riscv64 {
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

// lists from https://go.dev/doc/install/source#environment
//
//nolint:gochecknoglobals
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

	validGoamd64   = []string{"v1", "v2", "v3", "v4"}
	validGo386     = []string{"sse2", "softfloat"}
	validGoarm     = []string{"5", "6", "7"}
	validGoarm64   = regexp.MustCompile(`(v8\.[0-9]|v9\.[0-5])((,lse|,crypto)?)+`)
	validGomips    = []string{"hardfloat", "softfloat"}
	validGoppc64   = []string{"power8", "power9", "power10"}
	validGoriscv64 = []string{"rva20u64", "rva22u64"}
)
