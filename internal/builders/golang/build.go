package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
	api "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/pkg/errors"
)

// Default builder instance
var Default = &Builder{}

func init() {
	api.Register("go", Default)
}

// Builder is golang builder
type Builder struct{}

// WithDefaults sets the defaults for a golang build and returns it
func (*Builder) WithDefaults(build config.Build) config.Build {
	if build.Main == "" {
		build.Main = "."
	}
	if len(build.Goos) == 0 {
		build.Goos = []string{"linux", "darwin"}
	}
	if len(build.Goarch) == 0 {
		build.Goarch = []string{"amd64", "386"}
	}
	if len(build.Goarm) == 0 {
		build.Goarm = []string{"6"}
	}
	if len(build.Ldflags) == 0 {
		build.Ldflags = []string{"-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"}
	}
	if len(build.Targets) == 0 {
		build.Targets = matrix(build)
	}
	return build
}

// Build builds a golang build
func (*Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	if err := checkMain(build); err != nil {
		return err
	}
	cmd := []string{"go", "build"}

	cmd = append(cmd, build.Flags...)

	asmflags, err := processFlags(ctx, build.Asmflags, "asmflags", "-asmflags=")
	if err != nil {
		return err
	}
	cmd = append(cmd, asmflags...)

	gcflags, err := processFlags(ctx, build.Gcflags, "gcflags", "-gcflags=")
	if err != nil {
		return err
	}
	cmd = append(cmd, gcflags...)

	ldflags, err := processFlags(ctx, build.Ldflags, "ldflags", "-ldflags=")
	if err != nil {
		return err
	}
	cmd = append(cmd, ldflags...)

	cmd = append(cmd, "-o", options.Path, build.Main)

	target, err := newBuildTarget(options.Target)
	if err != nil {
		return err
	}
	var env = append(build.Env, target.Env()...)
	if err := run(ctx, cmd, env); err != nil {
		return errors.Wrapf(err, "failed to build for %s", options.Target)
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   target.os,
		Goarch: target.arch,
		Goarm:  target.arm,
		Extra: map[string]string{
			"Binary": build.Binary,
			"Ext":    options.Ext,
		},
	})
	return nil
}

func processFlags(ctx *context.Context, flags []string, flagName string, flagPrefix string) ([]string, error) {
	processed := make([]string, 0, len(flags))
	for _, rawFlag := range flags {
		flag, err := tmpl.New(ctx).Apply(rawFlag)
		if err != nil {
			return nil, err
		}
		processed = append(processed, flagPrefix+flag)
	}
	return processed, nil
}

func run(ctx *context.Context, command, env []string) error {
	/* #nosec */
	var cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	var log = log.WithField("env", env).WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.WithField("cmd", command).WithField("env", env).Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.New(string(out))
	}
	return nil
}

type buildTarget struct {
	os, arch, arm string
}

func newBuildTarget(s string) (buildTarget, error) {
	var t = buildTarget{}
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

func (b buildTarget) Env() []string {
	return []string{
		"GOOS=" + b.os,
		"GOARCH=" + b.arch,
		"GOARM=" + b.arm,
	}
}

func checkMain(build config.Build) error {
	var main = build.Main
	if main == "" {
		main = "."
	}
	stat, ferr := os.Stat(main)
	if ferr != nil {
		return ferr
	}
	if stat.IsDir() {
		packs, err := parser.ParseDir(token.NewFileSet(), main, nil, 0)
		if err != nil {
			return errors.Wrapf(err, "failed to parse dir: %s", main)
		}
		for _, pack := range packs {
			for _, file := range pack.Files {
				if hasMain(file) {
					return nil
				}
			}
		}
		return fmt.Errorf("build for %s does not contain a main function", build.Binary)
	}
	file, err := parser.ParseFile(token.NewFileSet(), main, nil, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to parse file: %s", main)
	}
	if hasMain(file) {
		return nil
	}
	return fmt.Errorf("build for %s does not contain a main function", build.Binary)
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
