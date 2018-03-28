package before

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

// Pipe is a global hook pipe
type Pipe struct{}

// String is the name of this pipe
func (Pipe) String() string {
	return "Run global hooks before starting the relase process"
}

// Default initialized the default values
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Before.Hooks == nil {
		ctx.Config.Before.Hooks = []string{}
	}
	return nil
}

// Run executes the hooks
func (Pipe) Run(ctx *context.Context) error {
	/* #nosec */
	for _, step := range ctx.Config.Before.Hooks {
		args := strings.Fields(step)
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("hook failed: %s\n%v", step, string(out))
		}
	}
	return nil
}
