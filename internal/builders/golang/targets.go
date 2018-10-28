package golang

import (
	"fmt"

	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type target struct {
	os, arch, arm string
}

func (t target) Env() []string {
	return []string{
		"GOOS=" + t.os,
		"GOARCH=" + t.arch,
		"GOARM=" + t.arm,
	}
}

func (t target) String() string {
	if t.arm != "" {
		return fmt.Sprintf("%s_%s_%s", t.os, t.arch, t.arm)
	}
	return fmt.Sprintf("%s_%s", t.os, t.arch)
}

func parseTarget(s string) (target, error) {
	var t = target{}
	parts := strings.Split(s, "_")
	if len(parts) < 2 {
		return t, fmt.Errorf("%s is not a valid build target", s)
	}
	t.os = parts[0]
	t.arch = parts[1]
	if len(parts) == 3 {
		t.arm = parts[2]
	}
	return t, nil
}

func matrix(build config.Build) (result []string) {
	// nolint:prealloc
	for _, target := range allBuildTargets(build) {
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
func (t target) ignored(ctx *context.Context, ignoredBuilds []config.IgnoredBuild) bool {
	for _, ig := range ignoredBuilds {
		goos, err := processTemplate(ctx, ig.Goos)

		if err != nil {
			log.WithError(err).Error("Could not process goos template")
			return true
		}

		if goos != "" && goos != t.os {
			continue
		}

		goarch, err := processTemplate(ctx, ig.Goarch)

		if err != nil {
			log.WithError(err).Error("Could not process goarch template")
			return true
		}

		if goarch != "" && goarch != t.arch {
			continue
		}

		goarm, err := processTemplate(ctx, ig.Goarm)

		if err != nil {
			log.WithError(err).Error("Could not process goarm template")
			return true
		}

		if goarm != "" && goarm != t.arm {
			continue
		}
		return true
	}
	return false
}

func (t target) valid() bool {
	var s = t.os + t.arch
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
