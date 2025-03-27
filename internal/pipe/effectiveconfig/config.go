// Package effectiveconfig writes the effective config file to dist.
package effectiveconfig

import (
	"os"
	"path/filepath"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/yaml"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe that writes the effective config file to dist.
type Pipe struct{}

func (Pipe) String() string { return "" }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) (err error) {
	path := filepath.Join(ctx.Config.Dist, "config.yaml")
	bts, err := yaml.Marshal(ctx.Config)
	if err != nil {
		return err
	}
	log.WithField("path", path).Debug("writing effective configuration")
	return os.WriteFile(path, bts, 0o644) //nolint:gosec
}
