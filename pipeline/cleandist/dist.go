// Package cleandist provides checks to make sure the dist folder is always
// empty.
package cleandist

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
)

// Pipe for cleandis
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Checking ./dist"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	_, err = os.Stat(ctx.Config.Dist)
	if os.IsNotExist(err) {
		log.Debug("./dist doesn't exist, moving on")
		return nil
	}
	if ctx.RmDist {
		log.Info("--rm-dist is set, removing ./dist")
		return os.RemoveAll(ctx.Config.Dist)
	}
	files, err := ioutil.ReadDir(ctx.Config.Dist)
	if err != nil {
		return
	}
	if len(files) > 0 {
		log.WithField("files", len(files)).Debug("./dist is not empty")
		return fmt.Errorf(
			"%s is not empty, remove it before running goreleaser or use the --rm-dist flag",
			ctx.Config.Dist,
		)
	}
	log.Debug("./dist is empty")
	return
}
