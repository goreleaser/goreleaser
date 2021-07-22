// Package snapshot provides the snapshotting functionality to goreleaser.
package snapshot

import (
	"fmt"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for checksums.
type Pipe struct{}

func (Pipe) String() string {
	return "snapshotting"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Snapshot.NameTemplate == "" {
		ctx.Config.Snapshot.NameTemplate = "{{ incpatch .Tag }}"
	}
	return nil
}

func (Pipe) Run(ctx *context.Context) error {
	if !ctx.Snapshot {
		log.Infof("releasing %s, commit %s", ctx.Git.CurrentTag, ctx.Git.Commit)
		return pipe.ErrSkipDisabledPipe
	}
	name, err := tmpl.New(ctx).Apply(ctx.Config.Snapshot.NameTemplate)
	if err != nil {
		return fmt.Errorf("failed to generate snapshot name: %w", err)
	}
	if name == "" {
		return fmt.Errorf("empty snapshot name")
	}
	ctx.Version = name
	log.Infof("creating snapshot %s", ctx.Version)
	return nil
}
