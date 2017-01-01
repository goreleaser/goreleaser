package cleanup

import (
	"log"
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
	log.Println("Cleaning up..")
	return os.RemoveAll("dist")
}
