package gomod

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	go115NotAGoModuleError = "go list -m: not using modules"
	go116NotAGoModuleError = "command-line-arguments"
)

// Pipe for gomod.
type Pipe struct{}

func (Pipe) String() string { return "loading go mod information" }

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
	return nil
}
