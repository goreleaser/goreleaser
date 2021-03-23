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
	template := tmpl.New(ctx)

	log.Infof("proxying %s@%s", ctx.ModulePath, ctx.Git.CurrentTag)

	mod, err := template.Apply(`
module {{ .ProjectName }}

require {{ .ModulePath }} {{ .Tag }}
`)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	main, err := template.Apply(`
// +build main
package main

import _ "{{ .ModulePath }}"
`)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	dir := fmt.Sprintf("%s/proxy", ctx.Config.Dist)

	log.Debugf("creating needed files")

	if err := os.Mkdir(dir, 0o755); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := os.WriteFile(dir+"/main.go", []byte(main), 0o650); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if err := os.WriteFile(dir+"/go.mod", []byte(mod), 0o650); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	sumr, err := os.OpenFile("go.sum", os.O_RDONLY, 0o650)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	sumw, err := os.OpenFile(dir+"/go.sum", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o650)
	if err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	if _, err := io.Copy(sumw, sumr); err != nil {
		return fmt.Errorf("failed to proxy module: %w", err)
	}

	log.Debugf("tidying")
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to proxy module: %w: %s", err, string(out))
	}

	for i := range ctx.Config.Builds {
		build := &ctx.Config.Builds[i]
		build.Main = path.Join(ctx.ModulePath, build.Main)
		build.Dir = dir
	}

	return nil
}
