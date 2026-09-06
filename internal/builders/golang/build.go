package golang

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"dario.cat/mergo"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
	gomain "github.com/goreleaser/goreleaser/v2/internal/builders/golang/gomain"
	"github.com/goreleaser/goreleaser/v2/internal/elf"
	"github.com/goreleaser/goreleaser/v2/internal/experimental"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultGoamd64   = "v1"
	defaultGo386     = "sse2"
	defaultGoarm64   = "v8.0"
	defaultGomips    = "hardfloat"
	defaultGoppc64   = "power8"
	defaultGoriscv64 = "rva20u64"
)

// Default builder instance.
//
//nolint:gochecknoglobals
var Default = &Builder{}

// type constraints
var (
	_ api.Builder          = &Builder{}
	_ api.DependingBuilder = &Builder{}
	_ api.TargetFixer      = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("go", Default)
}

// Builder is golang builder.
type Builder struct{}

// Dependencies implements build.DependingBuilder.
func (b *Builder) Dependencies() []string {
	return []string{"go"}
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	target = fixTarget(target)
	parts := strings.Split(target, "_")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	goos := parts[0]
	goarch := parts[1]

	t := Target{
		Target: target,
		Goos:   goos,
		Goarch: goarch,
	}

	if len(parts) > 2 {
		extra := parts[2]
		switch goarch {
		case "amd64":
			t.Goamd64 = extra
		case "arm64":
			t.Goarm64 = extra
		case "386":
			t.Go386 = extra
		case "arm":
			version, abi, err := splitGoarm(extra)
			if err != nil {
				return nil, err
			}
			t.Goarm = version
			t.Abi = abi
		case "mips", "mipsle", "mips64", "mips64le":
			t.Gomips = extra
		case "ppc64", "ppc64le":
			t.Goppc64 = extra
		case "riscv", "riscv64":
			t.Goriscv64 = extra
		}
	}

	return t, nil
}

// WithDefaults sets the defaults for a golang build and returns it.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	if build.Tool == "" {
		build.Tool = "go"
	}
	if build.Command == "" {
		build.Command = "build"
	}
	if build.Dir == "" {
		build.Dir = "."
	}
	if len(build.Ldflags) == 0 {
		build.Ldflags = []string{"-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser"}
	}

	_ = warnIfTargetsAndOtherOptionTogether(build)
	if len(build.Targets) == 0 {
		if len(build.Goos) == 0 {
			build.Goos = []string{"linux", "darwin", "windows"}
		}
		if len(build.Goarch) == 0 {
			build.Goarch = []string{"amd64", "arm64", "386"}
		}
		if len(build.Goamd64) == 0 {
			build.Goamd64 = []string{defaultGoamd64}
		}
		if len(build.Go386) == 0 {
			build.Go386 = []string{defaultGo386}
		}
		if len(build.Goarm) == 0 {
			build.Goarm = []string{experimental.DefaultGOARM()}
		}
		if len(build.Goarm64) == 0 {
			build.Goarm64 = []string{defaultGoarm64}
		}
		if len(build.Gomips) == 0 {
			build.Gomips = []string{defaultGomips}
		}
		if len(build.Goppc64) == 0 {
			build.Goppc64 = []string{defaultGoppc64}
		}
		if len(build.Goriscv64) == 0 {
			build.Goriscv64 = []string{defaultGoriscv64}
		}

		targets, err := listTargets(build)
		if err != nil {
			return build, err
		}
		build.Targets = targets
	} else {
		targets := map[string]bool{}
		for _, target := range build.Targets {
			if target == go118FirstClassTargetsName ||
				target == goStableFirstClassTargetsName {
				for _, t := range go118FirstClassTargets {
					targets[fixTarget(t)] = true
				}
				continue
			}
			targets[fixTarget(target)] = true
		}
		build.Targets = slices.Sorted(maps.Keys(targets))

		// Explicit targets skip the matrix, so validate their GOARM and run the
		// same one-ABI-per-version conflict check the matrix path runs.
		parsed := make([]Target, 0, len(build.Targets))
		for _, target := range build.Targets {
			t, err := b.Parse(target)
			if err != nil {
				return build, err
			}
			parsed = append(parsed, t.(Target))
		}
		if err := checkGoarmConflict(parsed); err != nil {
			return build, err
		}
	}

	for _, o := range build.BuildDetailsOverrides {
		if o.Goos == "" || o.Goarch == "" {
			log.Warn("overrides must set, at least, both 'goos' and 'goarch'")
			break
		}
	}
	return build, nil
}

// FixTarget implements build.TargetFixer.
func (b *Builder) FixTarget(target string) string {
	return fixTarget(target)
}

func fixTarget(target string) string {
	if strings.HasSuffix(target, "_amd64") {
		return target + "_" + defaultGoamd64
	}
	if strings.HasSuffix(target, "_386") {
		return target + "_" + defaultGo386
	}
	if strings.HasSuffix(target, "_arm") {
		return target + "_" + experimental.DefaultGOARM()
	}
	if strings.HasSuffix(target, "_arm64") {
		return target + "_" + defaultGoarm64
	}
	if strings.HasSuffix(target, "_mips") ||
		strings.HasSuffix(target, "_mips64") ||
		strings.HasSuffix(target, "_mipsle") ||
		strings.HasSuffix(target, "_mips64le") {
		return target + "_" + defaultGomips
	}
	if strings.HasSuffix(target, "_ppc64") ||
		strings.HasSuffix(target, "_ppc64le") {
		return target + "_" + defaultGoppc64
	}
	if strings.HasSuffix(target, "_riscv64") {
		return target + "_" + defaultGoriscv64
	}
	return target
}

func warnIfTargetsAndOtherOptionTogether(build config.Build) bool {
	if len(build.Targets) == 0 {
		return false
	}

	res := false
	for k, v := range map[string]int{
		"goos":      len(build.Goos),
		"goarch":    len(build.Goarch),
		"go386":     len(build.Go386),
		"goamd64":   len(build.Goamd64),
		"goarm":     len(build.Goarm),
		"goarm64":   len(build.Goarm64),
		"gomips":    len(build.Gomips),
		"goppc64":   len(build.Goppc64),
		"goriscv64": len(build.Goriscv64),
		"ignore":    len(build.Ignore),
	} {
		if v == 0 {
			continue
		}
		log.Warnf(logext.Keyword("builds."+k) + " is ignored when " + logext.Keyword("builds.targets") + " is set")
		res = true
	}
	return res
}

const (
	go118FirstClassTargetsName    = "go_118_first_class"
	goStableFirstClassTargetsName = "go_first_class"
)

// go tool dist list -json | jq -r '.[] | select(.FirstClass) | [.GOOS, .GOARCH] | @tsv'
var go118FirstClassTargets = []string{
	"darwin_amd64",
	"darwin_arm64",
	"linux_386",
	"linux_amd64",
	"linux_arm",
	"linux_arm64",
	"windows_386",
	"windows_amd64",
}

// Build builds a golang build.
func (*Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	t := options.Target.(Target)
	details, err := withOverrides(ctx, build, t)
	if err != nil {
		return err
	}

	env, testEnvs, err := buildEnv(ctx, details, options, getBinaryArtifact(t, build, options.Name, options.Path, options.Ext))
	if err != nil {
		return err
	}

	mains, allbinaries, err := checkBuild(ctx, build, details, options, env)
	if err != nil {
		return err
	}

	if len(testEnvs) > 0 {
		for i := range allbinaries {
			allbinaries[i].Extra["testEnvs"] = testEnvs
		}
	}

	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(allbinaries[0])

	cmd, err := buildGoBuildLine(ctx, build, details, options, allbinaries[0], mains, env)
	if err != nil {
		return err
	}

	if err := base.Exec(ctx, cmd, env, build.Dir, base.WithLogFilter(buildOutput)); err != nil {
		return err
	}
	if err := ensureEllipsisOutputs(mains, allbinaries, options.Ext); err != nil {
		return err
	}

	for _, a := range allbinaries {
		if err := base.ChTimes(build, tpl.WithArtifact(a), a); err != nil {
			return err
		}
		if a.Type == artifact.CShared || a.Type == artifact.CArchive {
			fullPathWithoutExt := strings.TrimSuffix(a.Path, options.Ext)
			if ha := getHeaderArtifactForLibrary(build, t, fullPathWithoutExt); ha != nil {
				if err := base.ChTimes(build, tpl.WithArtifact(ha), ha); err != nil {
					return err
				}
				ctx.Artifacts.Add(ha)
			}
		}
		if elf.IsDynamicallyLinked(a.Path) {
			a.Extra[artifact.ExtranDynLink] = true
		}
		ctx.Artifacts.Add(a)
	}
	return nil
}

// ensureEllipsisOutputs renames the files `go build` actually wrote to the
// paths GoReleaser registered.
//
// With an ellipsis main, `go build` gets an output directory instead of an
// output file, and names each output after its package, ignoring the extension
// GoReleaser derives from the buildmode.
func ensureEllipsisOutputs(mains map[string]string, binaries []*artifact.Artifact, ext string) error {
	if mains == nil || ext == "" {
		return nil
	}
	for _, a := range binaries {
		_, err := os.Stat(a.Path)
		if err == nil {
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", a.Name, err)
		}

		actual, err := findBuildOutput(a.Path, ext)
		if err != nil {
			return err
		}
		if err := os.Rename(actual, a.Path); err != nil {
			return fmt.Errorf("rename %s: %w", a.Name, err)
		}
	}
	return nil
}

// findBuildOutput looks for the file `go build` wrote in place of the expected
// path, e.g. `foo` for `foo.wasm`, or `foo.a` for `foo.lib`.
func findBuildOutput(expected, ext string) (string, error) {
	name := filepath.Base(expected)
	prefix := strings.TrimSuffix(expected, ext)
	matches, err := filepath.Glob(prefix + ".*")
	if err != nil {
		return "", fmt.Errorf("find build output for %s: %w", name, err)
	}
	// the header is generated alongside c-archive/c-shared libraries.
	candidates := slices.DeleteFunc(matches, func(m string) bool {
		return filepath.Ext(m) == ".h"
	})
	if _, err := os.Stat(prefix); err == nil {
		candidates = append(candidates, prefix)
	}
	if len(candidates) != 1 {
		return "", fmt.Errorf("could not find the build output for %s", name)
	}
	return candidates[0], nil
}

func buildEnv(ctx *context.Context, details config.BuildDetails, options api.Options, a *artifact.Artifact) ([]string, []string, error) {
	env := ctx.Env.Strings()
	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(a)

	tenv, err := base.TemplateEnv(details.Env, tpl)
	if err != nil {
		return nil, nil, err
	}

	var testEnvs []string
	for _, e := range tenv {
		if strings.HasPrefix(e, "TEST_") {
			testEnvs = append(testEnvs, e)
		}
	}
	env = append(env, tenv...)
	env = append(env, options.Target.(Target).env()...)
	if v := os.Getenv("GOCACHEPROG"); v != "" {
		env = append(env, "GOCACHEPROG="+v)
	}
	return env, testEnvs, nil
}

func buildOutput(out []byte) string {
	var lines []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" || strings.HasPrefix(line, "go: downloading") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func withOverrides(ctx *context.Context, build config.Build, target Target) (config.BuildDetails, error) {
	optsTarget := target.Target
	for _, o := range build.BuildDetailsOverrides {
		overrideTarget, err := tmpl.New(ctx).Apply(formatBuildTarget(o))
		if err != nil {
			return build.BuildDetails, err
		}
		overrideTarget = fixTarget(overrideTarget)

		if optsTarget == overrideTarget {
			dets := config.BuildDetails{
				Buildmode: build.Buildmode,
				Ldflags:   build.Ldflags,
				Tags:      build.Tags,
				Flags:     build.Flags,
				Asmflags:  build.Asmflags,
				Gcflags:   build.Gcflags,
			}
			if err := mergo.Merge(&dets, o.BuildDetails, mergo.WithOverride); err != nil {
				return build.BuildDetails, err
			}

			dets.Env = mergeEnv(build.Env, o.Env)
			log.WithField("details", dets).Infof("overridden build details for %s", optsTarget)
			return dets, nil
		}
		log.Debugf("targets don't match: %s != %s", optsTarget, overrideTarget)
	}

	return build.BuildDetails, nil
}

// mergeEnv merges the override entries into the defaults, keeping insertion
// order so that entries can reference the ones defined before them.
//
// The last definition of a key sets both its value and its position, so an
// override that redefines a key in terms of a variable it also introduces
// still comes out after it.
func mergeEnv(defaults, overrides []string) []string {
	all := append(slices.Clone(defaults), overrides...)
	values := make(map[string]string, len(all))
	keys := make([]string, 0, len(all))
	for _, env := range all {
		key, value, ok := strings.Cut(env, "=")
		if !ok || key == "" {
			continue
		}
		// a redefined key drops its old position.
		keys = slices.DeleteFunc(keys, func(k string) bool { return k == key })
		keys = append(keys, key)
		values[key] = value
	}

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func buildGoBuildLine(
	ctx *context.Context,
	build config.Build,
	details config.BuildDetails,
	options api.Options,
	artifact *artifact.Artifact,
	mains map[string]string,
	env []string,
) ([]string, error) {
	gobin, err := tmpl.New(ctx).WithBuildOptions(options).Apply(build.Tool)
	if err != nil {
		return nil, err
	}
	cmd := []string{gobin, build.Command}

	// tags, ldflags, and buildmode, should only appear once, warning only to avoid a breaking change
	validateUniqueFlags(details)

	tpl := tmpl.New(ctx).WithEnvS(env).WithArtifact(artifact)
	flags, err := tpl.Slice(details.Flags, tmpl.NonEmpty())
	if err != nil {
		return cmd, err
	}
	cmd = append(cmd, flags...)
	if build.Command == "test" && !slices.Contains(flags, "-c") {
		cmd = append(cmd, "-c")
	}

	asmflags, err := tpl.Slice(details.Asmflags, tmpl.NonEmpty(), tmpl.WithPrefix("-asmflags="))
	if err != nil {
		return cmd, err
	}
	cmd = append(cmd, asmflags...)

	gcflags, err := tpl.Slice(details.Gcflags, tmpl.NonEmpty(), tmpl.WithPrefix("-gcflags="))
	if err != nil {
		return cmd, err
	}
	cmd = append(cmd, gcflags...)

	// tags is not a repeatable flag
	if len(details.Tags) > 0 {
		tags, err := tpl.Slice(details.Tags, tmpl.NonEmpty())
		if err != nil {
			return cmd, err
		}
		cmd = append(cmd, "-tags="+strings.Join(tags, ","))
	}

	// ldflags is not a repeatable flag
	if len(details.Ldflags) > 0 {
		// flag prefix is skipped because ldflags need to output a single string
		ldflags, err := tpl.Slice(details.Ldflags, tmpl.NonEmpty())
		if err != nil {
			return cmd, err
		}
		// ldflags need to be single string in order to apply correctly
		cmd = append(cmd, "-ldflags="+strings.Join(ldflags, " "))
	}

	if details.Buildmode != "" {
		cmd = append(cmd, "-buildmode="+details.Buildmode)
	}

	if mains == nil {
		// NOTE: build.Main will never be empty here
		cmd = append(cmd, "-o", options.Path, build.Main)
	} else {
		cmd = append(cmd, "-o", filepath.Dir(options.Path))
		cmd = append(cmd, slices.Sorted(maps.Values(mains))...)
	}
	return cmd, nil
}

func validateUniqueFlags(details config.BuildDetails) {
	for _, flag := range details.Flags {
		if strings.HasPrefix(flag, "-tags") && len(details.Tags) > 0 {
			log.WithField("flag", flag).WithField("tags", details.Tags).Warn("tags is defined twice")
		}
		if strings.HasPrefix(flag, "-ldflags") && len(details.Ldflags) > 0 {
			log.WithField("flag", flag).WithField("ldflags", details.Ldflags).Warn("ldflags is defined twice")
		}
		if strings.HasPrefix(flag, "-buildmode") && details.Buildmode != "" {
			log.WithField("flag", flag).WithField("buildmode", details.Buildmode).Warn("buildmode is defined twice")
		}
	}
}

func artifactType(t Target, buildmode string) artifact.Type {
	switch buildmode {
	case "c-archive":
		return artifact.CArchive
	case "c-shared":
		if !strings.Contains(t.Target, "wasm") {
			return artifact.CShared
		}
	}
	return artifact.Binary
}

func getHeaderArtifactForLibrary(build config.Build, t Target, fullPathWithoutExt string) *artifact.Artifact {
	basePath := filepath.Base(fullPathWithoutExt)
	fullPath := fullPathWithoutExt + ".h"
	headerName := basePath + ".h"

	if _, err := os.Stat(fullPath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	a := &artifact.Artifact{
		Type:      artifact.Header,
		Path:      fullPath,
		Name:      headerName,
		Goos:      t.Goos,
		Goarch:    t.Goarch,
		Goamd64:   t.Goamd64,
		Go386:     t.Go386,
		Goarm:     t.Goarm,
		Goarm64:   t.Goarm64,
		Gomips:    t.Gomips,
		Goppc64:   t.Goppc64,
		Goriscv64: t.Goriscv64,
		Target:    t.Target,
		Extra: map[string]any{
			artifact.ExtraBinary: headerName,
			artifact.ExtraExt:    ".h",
			artifact.ExtraID:     build.ID,
		},
	}
	if t.Abi != "" {
		a.Extra[keyAbi] = t.Abi
	}
	return a
}

func getBinaryArtifact(
	t Target,
	build config.Build,
	name, path, ext string,
) *artifact.Artifact {
	a := &artifact.Artifact{
		Type:      artifactType(t, build.Buildmode),
		Path:      path,
		Name:      name,
		Goos:      t.Goos,
		Goarch:    t.Goarch,
		Goamd64:   t.Goamd64,
		Go386:     t.Go386,
		Goarm:     t.Goarm,
		Goarm64:   t.Goarm64,
		Gomips:    t.Gomips,
		Goppc64:   t.Goppc64,
		Goriscv64: t.Goriscv64,
		Target:    t.Target,
		Extra: map[string]any{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(name), ext),
			artifact.ExtraExt:     ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "go",
		},
	}
	if t.Abi != "" {
		a.Extra[keyAbi] = t.Abi
	}
	return a
}

func checkBuild(
	ctx *context.Context,
	build config.Build,
	details config.BuildDetails,
	options api.Options,
	env []string,
) (map[string]string, []*artifact.Artifact, error) {
	main := cmp.Or(build.UnproxiedMain, build.Main, ".")
	dir := cmp.Or(build.UnproxiedDir, build.Dir)

	t := options.Target.(Target)

	if build.NoMainCheck {
		return nil, []*artifact.Artifact{
			getBinaryArtifact(t, build, options.Name, options.Path, options.Ext),
		}, nil
	}

	if strings.HasSuffix(main, "/...") {
		return checkBuildElipsis(ctx, build, details, options, dir, main, env)
	}

	// old behavior
	if dir != "" {
		main = filepath.Join(dir, main)
	}
	if err := gomain.Check(main, build.Binary); err != nil {
		return nil, nil, err
	}

	logBuild(main, build.Binary, t.String())

	return nil, []*artifact.Artifact{
		getBinaryArtifact(t, build, options.Name, options.Path, options.Ext),
	}, nil
}

func checkBuildElipsis(
	ctx *context.Context,
	build config.Build,
	details config.BuildDetails,
	options api.Options,
	dir, main string,
	env []string,
) (map[string]string, []*artifact.Artifact, error) {
	logFindingMains(build, main)

	var binaries []*artifact.Artifact
	buildFlags, err := buildSelectionFlags(ctx, details, getBinaryArtifact(options.Target.(Target), build, options.Name, options.Path, options.Ext), env)
	if err != nil {
		return nil, nil, err
	}
	mains, err := gomain.All(dir, env, buildFlags, main)
	if err != nil {
		return nil, nil, err
	}

	// we should try and find all `func main`'s:
	if len(mains) > 1 && build.Binary != "" && !build.InternalDefaults.Binary {
		return nil, nil, errors.New("'main' contains an ellipsis path (e.g. './...') and 'binary' is also set: either set 'main' to a specific package, or unset 'binary' to auto-detect all mains and binary names")
	}

	if len(mains) > 1 && !build.InternalDefaults.ID {
		return nil, nil, errors.New("'main' contains an ellipsis path (e.g. './...') and resolves to more than one main package, and 'id' is set: either set 'main' to a specific package, or unset 'id'")
	}

	var bins []string
	var pkgs []string
	t := options.Target.(Target)

	for _, bin := range slices.SortedFunc(maps.Keys(mains), func(a, b string) int {
		return strings.Compare(mains[a], mains[b])
	}) {
		name := bin + options.Ext
		path := filepath.Join(filepath.Dir(options.Path), name)
		if build.UnproxiedMain != "" {
			mains[bin] = toProxiedImportPath(build, mains[bin])
		}
		bins = append(bins, name)
		pkgs = append(pkgs, mains[bin])
		a := getBinaryArtifact(t, build, name, path, options.Ext)
		// the set of main packages is target dependent, as build constraints
		// may exclude some of them, so the ID cannot depend on how many were
		// found for this target.
		if build.InternalDefaults.ID {
			a.Extra[artifact.ExtraID] = bin
		}
		binaries = append(binaries, a)
	}

	logBuild(pkgs, bins, t.String())
	return mains, binaries, nil
}

func buildSelectionFlags(
	ctx *context.Context,
	details config.BuildDetails,
	a *artifact.Artifact,
	env []string,
) ([]string, error) {
	tpl := tmpl.New(ctx).WithEnvS(env).WithArtifact(a)
	flags, err := tpl.Slice(details.Flags, tmpl.NonEmpty())
	if err != nil {
		return nil, err
	}
	flags = dropListIncompatibleFlags(flags)

	if len(details.Tags) > 0 {
		tags, err := tpl.Slice(details.Tags, tmpl.NonEmpty())
		if err != nil {
			return nil, err
		}
		flags = append(flags, "-tags="+strings.Join(tags, ","))
	}
	return flags, nil
}

// dropListIncompatibleFlags removes the flags `go list` rejects, as they only
// make sense when actually building. They cannot change package selection.
func dropListIncompatibleFlags(flags []string) []string {
	result := make([]string, 0, len(flags))
	for i := 0; i < len(flags); i++ {
		name, _, hasValue := strings.Cut(flags[i], "=")
		switch name {
		case "-c":
			continue
		case "-o":
			if !hasValue {
				i++ // the value is a separate argument.
			}
			continue
		}
		result = append(result, flags[i])
	}
	return result
}

func toProxiedImportPath(build config.Build, rel string) string {
	modulePath := strings.TrimSuffix(build.Main, "/...")
	if suffix := strings.TrimPrefix(strings.TrimSuffix(build.UnproxiedMain, "/..."), "."); suffix != "" {
		modulePath = strings.TrimSuffix(build.Main, suffix+"/...")
	}

	if rel == "." {
		return modulePath
	}

	return path.Join(modulePath, strings.TrimPrefix(rel, "./"))
}

var ellipsisLog = sync.Map{}

func logFindingMains(build config.Build, main string) {
	if _, loaded := ellipsisLog.LoadOrStore(build.ID, true); !loaded {
		log.
			WithField("id", build.ID).
			WithField("path", main).
			Info("finding all " + logext.Keyword("func main()"))
	}
}

func logBuild[T string | []string](paths T, binaries T, target string) {
	log.
		WithField("paths", paths).
		WithField("binaries", binaries).
		WithField("target", target).
		Info("building")
}
