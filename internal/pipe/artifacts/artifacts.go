// Package artifacts provides the pipe implementation that creates a artifacts.json file in the dist folder.
package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe implementation.
type Pipe struct{}

func (Pipe) String() string                 { return "storing artifact list" }
func (Pipe) Skip(ctx *context.Context) bool { return false }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	bts, err := json.Marshal(ctx.Artifacts.List())
	if err != nil {
		return err
	}
	path := filepath.Join(ctx.Config.Dist, "artifacts.json")
	log.Log.WithField("file", path).Info("writing")
	return os.WriteFile(path, bts, 0o600)
}
