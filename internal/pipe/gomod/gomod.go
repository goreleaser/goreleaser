package gomod

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	goPreModulesError      = "flag provided but not defined: -m"
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
	flags := []string{"list", "-m"}
	if ctx.Config.GoMod.Mod != "" {
		flags = append(flags, "-mod="+ctx.Config.GoMod.Mod)
	}
	cmd := exec.CommandContext(ctx, ctx.Config.GoMod.GoBinary, flags...)
	cmd.Env = append(ctx.Env.Strings(), ctx.Config.GoMod.Env...)
	if dir := ctx.Config.GoMod.Dir; dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if strings.HasPrefix(result, goPreModulesError) {
		return pipe.Skip("go version does not support modules")
	}
	if result == go115NotAGoModuleError || result == go116NotAGoModuleError {
		return pipe.Skip("not a go module")
	}
	if err != nil {
		return fmt.Errorf("failed to get module path: %w: %s", err, string(out))
	}

	// Splits and use the first line in case a `go.work` file exists with multiple modules.
	// The first module is/should be `.` in the `go.work` file, so this should be correct.
	// Running `go work sync` also always puts `.` as the first line in `use`.
	ctx.ModulePath = strings.Split(result, "\n")[0]
	return nil
}
