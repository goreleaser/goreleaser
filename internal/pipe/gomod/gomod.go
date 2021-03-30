package gomod

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for env.
type Pipe struct{}

func (Pipe) String() string {
	return "loading go mod information"
}

const (
	go115NotAGoModuleError = "go list -m: not using modules"
	go116NotAGoModuleError = "command-line-arguments"
)

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	out, err := exec.CommandContext(ctx, "go", "list", "-m").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get module path: %w: %s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == go115NotAGoModuleError || result == go116NotAGoModuleError {
		return pipe.Skip("not a go module")
	}

	ctx.ModulePath = result

	return nil
}
