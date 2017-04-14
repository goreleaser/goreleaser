package checksums

import (
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/sha256sum"
)

// Pipe for checksums
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Calculating checksums"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	file, err := os.OpenFile(
		filepath.Join(ctx.Config.Dist, "CHECKSUMS.txt"),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0600,
	)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	for _, artifact := range ctx.Artifacts {
		sha, err := sha256sum.For(filepath.Join(ctx.Config.Dist, artifact))
		if err != nil {
			return err
		}
		if _, err = file.WriteString(artifact + " sha256sum: " + sha + "\n"); err != nil {
			return err
		}
	}
	ctx.AddArtifact(file.Name())
	return
}
