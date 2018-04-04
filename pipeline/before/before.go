package before

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/context"
)

// Pipe is a global hook pipe
type Pipe struct{}

// String is the name of this pipe
func (Pipe) String() string {
	return "Running before hooks"
}

// Run executes the hooks
func (Pipe) Run(ctx *context.Context) error {
	/* #nosec */
	for _, step := range ctx.Config.Before.Hooks {
		args := strings.Fields(step)
		log.Infof("running %s", color.CyanString(step))
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Debug(string(out))
			return fmt.Errorf("hook failed: %s\n%v", step, string(out))
		}
	}
	return nil
}
