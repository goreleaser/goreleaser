// Package snapshot provides the snapshoting functionality to goreleaser.
package snapshot

import "github.com/goreleaser/goreleaser/context"

// Pipe for checksums
type Pipe struct{}

func (Pipe) String() string {
	return "snapshoting"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Snapshot.NameTemplate == "" {
		ctx.Config.Snapshot.NameTemplate = "SNAPSHOT-{{ .Commit }}"
	}
	return nil
}
