package golang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/pkg/errors"
)

var Default = &Builder{}

func init() {
	build.Register("go", Default)
}

type Builder struct {
}

func (*Builder) Build(ctx *context.Context, cfg config.Build, options build.Options) error {
	if err := checkMain(ctx, cfg); err != nil {
		return err
	}
	cmd := []string{"go", "build"}
	if cfg.Flags != "" {
		cmd = append(cmd, strings.Fields(cfg.Flags)...)
	}
	flags, err := ldflags(ctx, cfg)
	if err != nil {
		return err
	}
	cmd = append(cmd, "-ldflags="+flags, "-o", options.Path, cfg.Main)
	var target = newBuildTarget(options.Target)
	var env = append(cfg.Env, target.Env()...)
	if err := build.Run(ctx, cmd, env); err != nil {
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
			"Binary": cfg.Binary,
			"Ext":    options.Ext,
		},
	})
	return nil
}

func ldflags(ctx *context.Context, build config.Build) (string, error) {
	var data = struct {
		Commit  string
		Tag     string
		Version string
		Date    string
		Env     map[string]string
	}{
		Commit:  ctx.Git.Commit,
		Tag:     ctx.Git.CurrentTag,
		Version: ctx.Version,
		Date:    time.Now().UTC().Format(time.RFC3339),
		Env:     ctx.Env,
	}
	var out bytes.Buffer
	t, err := template.New("ldflags").
		Option("missingkey=error").
		Parse(build.Ldflags)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

type buildTarget struct {
	os, arch, arm string
}

func newBuildTarget(s string) buildTarget {
	var t = buildTarget{}
	parts := strings.Split(s, "_")
	t.os = parts[0]
	t.arch = parts[1]
	if len(parts) == 3 {
		t.arm = parts[2]
	}
	return t
}

func (b buildTarget) Env() []string {
	return []string{
		"GOOS=" + b.os,
		"GOARCH=" + b.arch,
		"GOARM=" + b.arm,
	}
}

func checkMain(ctx *context.Context, build config.Build) error {
	var main = build.Main
	if main == "" {
		main = "."
	}
	stat, ferr := os.Stat(main)
	if os.IsNotExist(ferr) {
		return errors.Wrapf(ferr, "could not open %s", main)
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
	file, err := parser.ParseFile(token.NewFileSet(), build.Main, nil, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to parse file: %s", build.Main)
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
