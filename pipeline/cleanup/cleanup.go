package cleanup

import (
	"os"

	"github.com/goreleaser/releaser/config"
)

// Pipe for cleanup
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Cleanup"
}

// Run the pipe
func (Pipe) Run(config config.ProjectConfig) error {
	return os.RemoveAll("dist")
}
