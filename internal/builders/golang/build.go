package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/builders/buildtarget"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	api "github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Default builder instance.
//
//nolint:gochecknoglobals
var Default = &Builder{}

//nolint:gochecknoinits
func init() {
	api.Register("go", Default)
}

// Builder is golang builder.
type Builder struct{}

// WithDefaults sets the defaults for a golang build and returns it.
func (*Builder) WithDefaults(build config.Build) (config.Build, error) {
	if build.GoBinary == "" {
		build.GoBinary = "go"
	}
	if build.Command == "" {
		build.Command = "build"
	}
	if build.Dir == "" {
		build.Dir = "."
	}
	if build.Main == "" {
		build.Main = "."
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
		if len(build.Goarm) == 0 {
			build.Goarm = []string{"6"}
		}
		if len(build.Gomips) == 0 {
			build.Gomips = []string{"hardfloat"}
		}
		if len(build.Goamd64) == 0 {
			build.Goamd64 = []string{"v1"}
		}
		targets, err := buildtarget.List(build)
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
					targets[t] = true
				}
				continue
			}
			if strings.HasSuffix(target, "_amd64") {
				targets[target+"_v1"] = true
				continue
			}
			if strings.HasSuffix(target, "_arm") {
				targets[target+"_6"] = true
				continue
			}
			if strings.HasSuffix(target, "_mips") ||
				strings.HasSuffix(target, "_mips64") ||
				strings.HasSuffix(target, "_mipsle") ||
				strings.HasSuffix(target, "_mips64le") {
				targets[target+"_hardfloat"] = true
				continue
			}
			targets[target] = true
		}
		build.Targets = keys(targets)
	}
	return build, nil
}

func warnIfTargetsAndOtherOptionTogether(build config.Build) bool {
	if len(build.Targets) == 0 {
		return false
	}

	res := false
	for k, v := range map[string]int{
		"goos":    len(build.Goos),
		"goarch":  len(build.Goarch),
		"goarm":   len(build.Goarm),
		"gomips":  len(build.Gomips),
		"goamd64": len(build.Goamd64),
		"ignore":  len(build.Ignore),
	} {
		if v == 0 {
			continue
		}
		log.Warnf(logext.Keyword("builds."+k) + " is ignored when " + logext.Keyword("builds.targets") + " is set")
		res = true
	}
	return res
}

func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

const (
	go118FirstClassTargetsName    = "go_118_first_class"
	goStableFirstClassTargetsName = "go_first_class"
)

// go tool dist list -json | jq -r '.[] | select(.FirstClass) | [.GOOS, .GOARCH] | @tsv'
var go118FirstClassTargets = []string{
	"darwin_amd64_v1",
	"darwin_arm64",
	"linux_386",
	"linux_amd64_v1",
	"linux_arm_6",
	"linux_arm64",
	"windows_386",
	"windows_amd64_v1",
}

// Build builds a golang build.
func (*Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	if err := checkMain(build); err != nil {
		return err
	}

	a := &artifact.Artifact{
		Type:    artifact.Binary,
		Path:    options.Path,
		Name:    options.Name,
		Goos:    options.Goos,
		Goarch:  options.Goarch,
		Goamd64: options.Goamd64,
		Goarm:   options.Goarm,
		Gomips:  options.Gomips,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:    options.Ext,
			artifact.ExtraID:     build.ID,
		},
	}

	if build.Buildmode == "c-archive" {
		a.Type = artifact.CArchive
		ctx.Artifacts.Add(getHeaderArtifactForLibrary(build, options))
	}
	if build.Buildmode == "c-shared" {
		a.Type = artifact.CShared
		ctx.Artifacts.Add(getHeaderArtifactForLibrary(build, options))
	}

	details, err := withOverrides(ctx, build, options)
	if err != nil {
		return err
	}

	env := []string{}
	// used for unit testing only
	testEnvs := []string{}
	env = append(env, ctx.Env.Strings()...)
	for _, e := range details.Env {
		ee, err := tmpl.New(ctx).WithEnvS(env).WithArtifact(a).Apply(e)
		if err != nil {
			return err
		}
		log.Debugf("env %q evaluated to %q", e, ee)
		if ee != "" {
			env = append(env, ee)
			if strings.HasPrefix(e, "TEST_") {
				testEnvs = append(testEnvs, ee)
			}
		}
	}
	env = append(
		env,
		"GOOS="+options.Goos,
		"GOARCH="+options.Goarch,
		"GOARM="+options.Goarm,
		"GOMIPS="+options.Gomips,
		"GOMIPS64="+options.Gomips,
		"GOAMD64="+options.Goamd64,
	)

	if len(testEnvs) > 0 {
		a.Extra["testEnvs"] = testEnvs
	}

	cmd, err := buildGoBuildLine(ctx, build, details, options, a, env)
	if err != nil {
		return err
	}

	if err := run(ctx, cmd, env, build.Dir); err != nil {
		return fmt.Errorf("failed to build for %s: %w", options.Target, err)
	}

	modTimestamp, err := tmpl.New(ctx).WithEnvS(env).WithArtifact(a).Apply(build.ModTimestamp)
	if err != nil {
		return err
	}
	if err := gio.Chtimes(options.Path, modTimestamp); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}

func withOverrides(ctx *context.Context, build config.Build, options api.Options) (config.BuildDetails, error) {
	optsTarget := options.Goos + options.Goarch + options.Goarm + options.Gomips + options.Goamd64
	for _, o := range build.BuildDetailsOverrides {
		overrideTarget, err := tmpl.New(ctx).Apply(o.Goos + o.Goarch + o.Gomips + o.Goarm + o.Goamd64)
		if err != nil {
			return build.BuildDetails, err
		}

		if optsTarget == overrideTarget {
			dets := config.BuildDetails{
				Buildmode: build.BuildDetails.Buildmode,
				Ldflags:   build.BuildDetails.Ldflags,
				Tags:      build.BuildDetails.Tags,
				Flags:     build.BuildDetails.Flags,
				Asmflags:  build.BuildDetails.Asmflags,
				Gcflags:   build.BuildDetails.Gcflags,
			}
			if err := mergo.Merge(&dets, o.BuildDetails, mergo.WithOverride); err != nil {
				return build.BuildDetails, err
			}

			dets.Env = context.ToEnv(append(build.Env, o.BuildDetails.Env...)).Strings()
			log.WithField("details", dets).Infof("overridden build details for %s", optsTarget)
			return dets, nil
		}
	}

	return build.BuildDetails, nil
}

func buildGoBuildLine(
	ctx *context.Context,
	build config.Build,
	details config.BuildDetails,
	options api.Options,
	artifact *artifact.Artifact,
	env []string,
) ([]string, error) {
	gobin, err := tmpl.New(ctx).WithBuildOptions(options).Apply(build.GoBinary)
	if err != nil {
		return nil, err
	}
	cmd := []string{gobin, build.Command}

	// tags, ldflags, and buildmode, should only appear once, warning only to avoid a breaking change
	validateUniqueFlags(details)

	flags, err := processFlags(ctx, artifact, env, details.Flags, "")
	if err != nil {
		return cmd, err
	}
	cmd = append(cmd, flags...)
	if build.Command == "test" && !slices.Contains(flags, "-c") {
		cmd = append(cmd, "-c")
	}

	asmflags, err := processFlags(ctx, artifact, env, details.Asmflags, "-asmflags=")
	if err != nil {
		return cmd, err
	}
	cmd = append(cmd, asmflags...)

	gcflags, err := processFlags(ctx, artifact, env, details.Gcflags, "-gcflags=")
	if err != nil {
		return cmd, err
	}
	cmd = append(cmd, gcflags...)

	// tags is not a repeatable flag
	if len(details.Tags) > 0 {
		tags, err := processFlags(ctx, artifact, env, details.Tags, "")
		if err != nil {
			return cmd, err
		}
		cmd = append(cmd, "-tags="+strings.Join(tags, ","))
	}

	// ldflags is not a repeatable flag
	if len(details.Ldflags) > 0 {
		// flag prefix is skipped because ldflags need to output a single string
		ldflags, err := processFlags(ctx, artifact, env, details.Ldflags, "")
		if err != nil {
			return cmd, err
		}
		// ldflags need to be single string in order to apply correctly
		cmd = append(cmd, "-ldflags="+strings.Join(ldflags, " "))
	}

	if details.Buildmode != "" {
		cmd = append(cmd, "-buildmode="+details.Buildmode)
	}

	cmd = append(cmd, "-o", options.Path, build.Main)
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

func processFlags(ctx *context.Context, a *artifact.Artifact, env, flags []string, flagPrefix string) ([]string, error) {
	processed := make([]string, 0, len(flags))
	for _, rawFlag := range flags {
		flag, err := processFlag(ctx, a, env, rawFlag)
		if err != nil {
			return nil, err
		}
		processed = append(processed, flagPrefix+flag)
	}
	return processed, nil
}

func processFlag(ctx *context.Context, a *artifact.Artifact, env []string, rawFlag string) (string, error) {
	return tmpl.New(ctx).WithEnvS(env).WithArtifact(a).Apply(rawFlag)
}

func run(ctx *context.Context, command, env []string, dir string) error {
	/* #nosec */
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env
	cmd.Dir = dir
	log.Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	if s := buildOutput(out); s != "" {
		log.WithField("cmd", command).Info(s)
	}
	return nil
}

func buildOutput(out []byte) string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, "go: downloading") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func checkMain(build config.Build) error {
	if build.NoMainCheck {
		return nil
	}
	main := build.Main
	if build.UnproxiedMain != "" {
		main = build.UnproxiedMain
	}
	dir := build.Dir
	if build.UnproxiedDir != "" {
		dir = build.UnproxiedDir
	}

	if main == "" {
		main = "."
	}
	if dir != "" {
		main = filepath.Join(dir, main)
	}
	stat, ferr := os.Stat(main)
	if ferr != nil {
		return fmt.Errorf("couldn't find main file: %w", ferr)
	}
	if stat.IsDir() {
		packs, err := parser.ParseDir(token.NewFileSet(), main, nil, 0)
		if err != nil {
			return fmt.Errorf("failed to parse dir: %s: %w", main, err)
		}
		for _, pack := range packs {
			for _, file := range pack.Files {
				if hasMain(file) {
					return nil
				}
			}
		}
		return errNoMain{build.Binary}
	}
	file, err := parser.ParseFile(token.NewFileSet(), main, nil, 0)
	if err != nil {
		return fmt.Errorf("failed to parse file: %s: %w", main, err)
	}
	if hasMain(file) {
		return nil
	}
	return errNoMain{build.Binary}
}

type errNoMain struct {
	bin string
}

func (e errNoMain) Error() string {
	return fmt.Sprintf("build for %s does not contain a main function\nLearn more at https://goreleaser.com/errors/no-main\n", e.bin)
}

func hasMain(file *ast.File) bool {
	for _, decl := range file.Decls {
		fn, isFn := decl.(*ast.FuncDecl)
		if !isFn {
			continue
		}
		if fn.Name.Name == "main" && fn.Recv == nil {
			return true
		}
	}
	return false
}

func getHeaderArtifactForLibrary(build config.Build, options api.Options) *artifact.Artifact {
	fullPathWithoutExt := strings.TrimSuffix(options.Path, options.Ext)
	basePath := filepath.Base(fullPathWithoutExt)
	fullPath := fullPathWithoutExt + ".h"
	headerName := basePath + ".h"

	return &artifact.Artifact{
		Type:    artifact.Header,
		Path:    fullPath,
		Name:    headerName,
		Goos:    options.Goos,
		Goarch:  options.Goarch,
		Goamd64: options.Goamd64,
		Goarm:   options.Goarm,
		Gomips:  options.Gomips,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: headerName,
			artifact.ExtraExt:    ".h",
			artifact.ExtraID:     build.ID,
		},
	}
}
