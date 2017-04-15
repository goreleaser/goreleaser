// Package client contains the client implementations for several providers.
package client

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/goreleaser/goreleaser/context"
)

// Info of the repository
type Info struct {
	Description string
	Homepage    string
	URL         string
}

// Client interface
type Client interface {
	GetInfo(ctx *context.Context) (info Info, err error)
	CreateRelease(ctx *context.Context) (releaseID int, err error)
	CreateFile(ctx *context.Context, content bytes.Buffer, path string) (err error)
	Upload(ctx *context.Context, releaseID int, name string, file *os.File) (err error)
}

func describeRelease(diff string) string {
	result := "## Changelog\n" + diff + "\n\n--\nAutomated with @goreleaser"
	cmd := exec.Command("go", "version")
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return result
	}
	return result + "\nBuilt with " + string(bts)
}
