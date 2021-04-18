// Package gomod provides go modules utilities, such as template variables and the ability to proxy the module from
// proxy.golang.org.
package gomod

import (
	"errors"
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

const (
	go115NotAGoModuleError = "go list -m: not using modules"
	go116NotAGoModuleError = "command-line-arguments"
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
	result := strings.TrimSpace(string(out))
	if result == go115NotAGoModuleError || result == go116NotAGoModuleError {
		return pipe.Skip("not a go module")
	}
	if err != nil {
		return fmt.Errorf("failed to get module path: %w: %s", err, string(out))
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

// ErrProxy happens when something goes wrong while proxying the current go module.
type ErrProxy struct {
	err     error
	details string
}

func newErrProxy(err error) error {
	return ErrProxy{
		err: err,
	}
}

func newDetailedErrProxy(err error, details string) error {
	return ErrProxy{
		err:     err,
		details: details,
	}
}

func (e ErrProxy) Error() string {
	out := fmt.Sprintf("failed to proxy module: %v", e.err)
	if e.details != "" {
		return fmt.Sprintf("%s: %s", out, e.details)
	}
	return out
}

func (e ErrProxy) Unwrap() error {
	return e.err
}

func proxyBuild(ctx *context.Context, build *config.Build) error {
	mainPackage := path.Join(ctx.ModulePath, build.Main)
	if strings.HasSuffix(build.Main, ".go") {
		pkg := path.Dir(build.Main)
		log.Warnf("guessing package of '%s' to be '%s', if this is incorrect, setup 'build.%s.main' to be the correct package", build.Main, pkg, build.ID)
		mainPackage = path.Join(ctx.ModulePath, pkg)
	}
	template := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"Main":    mainPackage,
		"BuildID": build.ID,
	})

	log.Infof("proxying %s@%s to build %s", ctx.ModulePath, ctx.Git.CurrentTag, mainPackage)

	mod, err := template.Apply(goModTpl)
	if err != nil {
		return newErrProxy(err)
	}

	main, err := template.Apply(mainGoTpl)
	if err != nil {
		return newErrProxy(err)
	}

	dir := filepath.Join(ctx.Config.Dist, "proxy", build.ID)

	log.Debugf("creating needed files")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return newErrProxy(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0o666); err != nil {
		return newErrProxy(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o666); err != nil {
		return newErrProxy(err)
	}

	if err := copyGoSum("go.sum", filepath.Join(dir, "go.sum")); err != nil {
		return newErrProxy(err)
	}

	log.Debugf("tidying")
	cmd := exec.CommandContext(ctx, ctx.Config.GoMod.GoBinary, "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(ctx.Config.GoMod.Env, os.Environ()...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return newDetailedErrProxy(err, string(out))
	}

	build.Main = mainPackage
	build.Dir = dir
	return nil
}

func copyGoSum(src, dst string) error {
	r, err := os.OpenFile(src, os.O_RDONLY, 0o666)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}
