// Package gomod provides go modules utilities, such as template variables and the ability to proxy the module from
// proxy.golang.org.
package gomod

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for env.
type Pipe struct{}

func (Pipe) String() string {
	return "loading go mod information"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.GoMod.GoBinary == "" {
		ctx.Config.GoMod.GoBinary = "go"
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	out, err := exec.CommandContext(ctx, ctx.Config.GoMod.GoBinary, "list", "-m").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get module path: %w: %s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == "command-line-arguments" {
		return pipe.Skip("not a go module")
	}

	ctx.ModulePath = result

	if !ctx.Config.GoMod.Proxy {
		return pipe.Skip("gomod.proxy is disabled")
	}

	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}

	return setupProxy(ctx)
}

func setupProxy(ctx *context.Context) error {
	for i := range ctx.Config.Builds {
		build := &ctx.Config.Builds[i]
		if err := proxyBuild(ctx, build); err != nil {
			return err
		}
	}

	return nil
}

const goModTpl = `
module {{ .BuildID }}

require {{ .ModulePath }} {{ .Tag }}
`

const mainGoTpl = `
// +build main
package main

import _ "{{ .Main }}"
`

func proxyBuild(ctx *context.Context, build *config.Build) error {
	mainPackage := path.Join(ctx.ModulePath, build.Main)
	template := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"Main":    mainPackage,
		"BuildID": build.ID,
	})

	log.Infof("proxying %s@%s to build %s", ctx.ModulePath, ctx.Git.CurrentTag, mainPackage)

	mod, err := template.Apply(goModTpl)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	main, err := template.Apply(mainGoTpl)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	dir := filepath.Join(ctx.Config.Dist, "proxy", build.ID)

	log.Debugf("creating needed files")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0o666); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o666); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := copyGoSum("go.sum", filepath.Join(dir, "go.sum")); err != nil {
		return err
	}

	log.Debugf("tidying")
	cmd := exec.CommandContext(ctx, ctx.Config.GoMod.GoBinary, "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(ctx.Config.GoMod.Env, os.Environ()...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to proxy module: %w: %s", err, string(out))
	}

	build.Main = mainPackage
	build.Dir = dir
	return nil
}

func copyGoSum(src, dst string) error {
	r, err := os.OpenFile(src, os.O_RDONLY, 0o666)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	return nil
}
