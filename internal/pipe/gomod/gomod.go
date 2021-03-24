package gomod

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
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

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	out, err := exec.CommandContext(ctx, "go", "list", "-m").CombinedOutput()
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

func proxyBuild(ctx *context.Context, build *config.Build) error {
	mainPackage := path.Join(ctx.ModulePath, build.Main)
	template := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"Main":    mainPackage,
		"BuildID": build.ID,
	})

	log.Infof("proxying %s@%s to build %s", ctx.ModulePath, ctx.Git.CurrentTag, mainPackage)

	mod, err := template.Apply(`
module {{ .BuildID }}

require {{ .ModulePath }} {{ .Tag }}
`)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	main, err := template.Apply(`
// +build main
package main

import _ "{{ .Main }}"
`)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	dir := fmt.Sprintf("%s/proxy/%s", ctx.Config.Dist, build.ID)

	log.Debugf("creating needed files")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := os.WriteFile(dir+"/main.go", []byte(main), 0o666); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := os.WriteFile(dir+"/go.mod", []byte(mod), 0o666); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	sumr, err := os.OpenFile("go.sum", os.O_RDONLY, 0o666)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	sumw, err := os.Create(dir + "/go.sum")
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}
	defer sumw.Close()

	if _, err := io.Copy(sumw, sumr); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	log.Debugf("tidying")
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to proxy module: %w: %s", err, string(out))
	}

	build.Main = mainPackage
	build.Dir = dir
	return nil
}
