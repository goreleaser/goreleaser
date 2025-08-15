package golang

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

func formatBuildTarget(o config.BuildDetailsOverride) string {
	return formatTarget(Target{
		Goos:      o.Goos,
		Goarch:    o.Goarch,
		Goamd64:   o.Goamd64,
		Go386:     o.Go386,
		Goarm:     o.Goarm,
		Goarm64:   o.Goarm64,
		Gomips:    o.Gomips,
		Goppc64:   o.Goppc64,
		Goriscv64: o.Goriscv64,
	})
}

func formatTarget(t Target) string {
	target := t.Goos + "_" + t.Goarch
	if extra := t.Goamd64 + t.Go386 + t.Goarm + t.Goarm64 + t.Gomips + t.Goppc64 + t.Goriscv64; extra != "" {
		target += "_" + extra
	}
	return target
}

// Target is a Go build target.
type Target struct {
	Target    string
	Goos      string
	Goarch    string
	Goamd64   string
	Go386     string
	Goarm     string
	Goarm64   string
	Gomips    string
	Goppc64   string
	Goriscv64 string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:      t.Goos,
		tmpl.KeyArch:    t.Goarch,
		tmpl.KeyAmd64:   t.Goamd64,
		tmpl.Key386:     t.Go386,
		tmpl.KeyArm:     t.Goarm,
		tmpl.KeyArm64:   t.Goarm64,
		tmpl.KeyMips:    t.Gomips,
		tmpl.KeyPpc64:   t.Goppc64,
		tmpl.KeyRiscv64: t.Goriscv64,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

func (t Target) env() []string {
	return []string{
		"GOOS=" + t.Goos,
		"GOARCH=" + t.Goarch,
		"GOAMD64=" + t.Goamd64,
		"GO386=" + t.Go386,
		"GOARM=" + t.Goarm,
		"GOARM64=" + t.Goarm64,
		"GOMIPS=" + t.Gomips,
		"GOMIPS64=" + t.Gomips,
		"GOPPC64=" + t.Goppc64,
		"GORISCV64=" + t.Goriscv64,
	}
}

func listTargets(build config.Build) ([]string, error) {
	//nolint:prealloc
	var targets []Target
	//nolint:prealloc
	var result []string
	for _, target := range allBuildTargets(build) {
		if !slices.Contains(validGoos, target.Goos) {
			return result, fmt.Errorf("invalid goos: %s", target.Goos)
		}
		if !slices.Contains(validGoarch, target.Goarch) {
			return result, fmt.Errorf("invalid goarch: %s", target.Goarch)
		}
		if target.Goamd64 != "" && !slices.Contains(validGoamd64, target.Goamd64) {
			return result, fmt.Errorf("invalid goamd64: %s", target.Goamd64)
		}
		if target.Go386 != "" && !slices.Contains(validGo386, target.Go386) {
			return result, fmt.Errorf("invalid go386: %s", target.Go386)
		}
		if target.Goarm != "" && !slices.Contains(validGoarm, target.Goarm) {
			return result, fmt.Errorf("invalid goarm: %s", target.Goarm)
		}
		if target.Goarm64 != "" && !validGoarm64.MatchString(target.Goarm64) {
			return result, fmt.Errorf("invalid goarm64: %s", target.Goarm64)
		}
		if target.Gomips != "" && !slices.Contains(validGomips, target.Gomips) {
			return result, fmt.Errorf("invalid gomips: %s", target.Gomips)
		}
		if target.Goppc64 != "" && !slices.Contains(validGoppc64, target.Goppc64) {
			return result, fmt.Errorf("invalid goppc64: %s", target.Goppc64)
		}
		if target.Goriscv64 != "" && !slices.Contains(validGoriscv64, target.Goriscv64) {
			return result, fmt.Errorf("invalid goriscv64: %s", target.Goriscv64)
		}
		if !valid(target) {
			log.WithField("target", target).Debug("skipped invalid build")
			continue
		}
		if ignored(build, target) {
			log.WithField("target", target).Debug("skipped ignored build")
			continue
		}
		warnUnstable(target)
		targets = append(targets, target)
	}
	for _, target := range targets {
		result = append(result, target.String())
	}
	return result, nil
}

func allBuildTargets(build config.Build) []Target {
	var targets []Target
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			switch goarch {
			case "386":
				for _, go386 := range build.Go386 {
					targets = append(targets, Target{
						Goos:   goos,
						Goarch: goarch,
						Go386:  go386,
					})
				}
			case "amd64":
				for _, goamd64 := range build.Goamd64 {
					targets = append(targets, Target{
						Goos:    goos,
						Goarch:  goarch,
						Goamd64: goamd64,
					})
				}
			case "arm64":
				for _, goarm64 := range build.Goarm64 {
					targets = append(targets, Target{
						Goos:    goos,
						Goarch:  goarch,
						Goarm64: goarm64,
					})
				}
			case "arm":
				for _, goarm := range build.Goarm {
					targets = append(targets, Target{
						Goos:   goos,
						Goarch: goarch,
						Goarm:  goarm,
					})
				}
			case "mips", "mipsle", "mips64", "mips64le":
				for _, gomips := range build.Gomips {
					targets = append(targets, Target{
						Goos:   goos,
						Goarch: goarch,
						Gomips: gomips,
					})
				}
			case "ppc64", "ppc64le":
				for _, goppc64 := range build.Goppc64 {
					targets = append(targets, Target{
						Goos:    goos,
						Goarch:  goarch,
						Goppc64: goppc64,
					})
				}
			case "riscv64":
				for _, goriscv64 := range build.Goriscv64 {
					targets = append(targets, Target{
						Goos:      goos,
						Goarch:    goarch,
						Goriscv64: goriscv64,
					})
				}
			default:
				targets = append(targets, Target{
					Goos:   goos,
					Goarch: goarch,
				})
			}
		}
	}
	for i := range targets {
		targets[i].Target = formatTarget(targets[i])
	}
	return targets
}

func ignored(build config.Build, target Target) bool {
	for _, ig := range build.Ignore {
		if ig.Goos != "" && ig.Goos != target.Goos {
			continue
		}
		if ig.Goarch != "" && ig.Goarch != target.Goarch {
			continue
		}
		if ig.Goamd64 != "" && ig.Goamd64 != target.Goamd64 {
			continue
		}
		if ig.Go386 != "" && ig.Go386 != target.Go386 {
			continue
		}
		if ig.Goarm != "" && ig.Goarm != target.Goarm {
			continue
		}
		if ig.Goarm64 != "" && ig.Goarm64 != target.Goarm64 {
			continue
		}
		if ig.Gomips != "" && ig.Gomips != target.Gomips {
			continue
		}
		if ig.Goppc64 != "" && ig.Goppc64 != target.Goppc64 {
			continue
		}
		if ig.Goriscv64 != "" && ig.Goriscv64 != target.Goriscv64 {
			continue
		}
		return true
	}
	return false
}

func valid(target Target) bool {
	t := target.Goos + target.Goarch
	return slices.Contains(validTargets, t) ||
		slices.Contains(experimentalTargets, t) ||
		slices.Contains(brokenTargets, t)
}

func warnUnstable(target Target) {
	t := target.Goos + target.Goarch
	if slices.Contains(experimentalTargets, t) {
		log.WithField("target", target).Warn("experimental target, use at your own risk")
	}
	if slices.Contains(brokenTargets, t) {
		log.WithField("target", target).Error("broken target, use at your own risk")
	}
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
		"freebsdarm64",
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
		"netbsdarm64",
		"openbsd386",
		"openbsdamd64",
		"openbsdarm",
		"openbsdarm64",
		"plan9386",
		"plan9amd64",
		"plan9arm",
		"solarisamd64",
		"solarissparc",
		"solarissparc64",
		"windowsarm64",
		"windows386",
		"windowsamd64",
	}

	experimentalTargets = []string{
		"openbsdriscv64", // https://golang.google.cn/doc/go1.23#openbsd
	}

	brokenTargets = []string{
		"windowsarm", // broken: https://golang.google.cn/doc/go1.24#windows , https://golang.google.cn/doc/go1.25#windows
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
		"sparc",
		"sparc64",
	}

	validGoamd64   = []string{"v1", "v2", "v3", "v4"}
	validGo386     = []string{"sse2", "softfloat"}
	validGoarm     = []string{"5", "6", "7"}
	validGoarm64   = regexp.MustCompile(`(v8\.[0-9]|v9\.[0-5])((,lse|,crypto)?)+`)
	validGomips    = []string{"hardfloat", "softfloat"}
	validGoppc64   = []string{"power8", "power9", "power10"}
	validGoriscv64 = []string{"rva20u64", "rva22u64", "rva23u64"}
)
