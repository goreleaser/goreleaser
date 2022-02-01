// Package metadata provides the pipe implementation that creates a artifacts.json file in the dist folder.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe implementation.
type Pipe struct{}

func (Pipe) String() string                 { return "storing release metadata" }
func (Pipe) Skip(ctx *context.Context) bool { return false }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if err := writeArtifacts(ctx); err != nil {
		return err
	}
	return writeMetadata(ctx)
}

func writeMetadata(ctx *context.Context) error {
	return writeJSON(ctx, metadata{
		ProjectName: ctx.Config.ProjectName,
		Tag:         ctx.Git.CurrentTag,
		PreviousTag: ctx.Git.PreviousTag,
		Version:     ctx.Version,
		Commit:      ctx.Git.Commit,
		Date:        ctx.Date,
		Runtime: metaRuntime{
			Goos:   ctx.Runtime.Goos,
			Goarch: ctx.Runtime.Goarch,
		},
	}, "metadata.json")
}

func writeArtifacts(ctx *context.Context) error {
	return writeJSON(ctx, ctx.Artifacts.List(), "artifacts.json")
}

func writeJSON(ctx *context.Context, j interface{}, name string) error {
	bts, err := json.Marshal(j)
	if err != nil {
		return err
	}
	path := filepath.Join(ctx.Config.Dist, name)
	log.Log.WithField("file", path).Info("writing")
	return os.WriteFile(path, bts, 0o644)
}

type metadata struct {
	ProjectName string      `json:"project_name"`
	Tag         string      `json:"tag"`
	PreviousTag string      `json:"previous_tag"`
	Version     string      `json:"version"`
	Commit      string      `json:"commit"`
	Date        time.Time   `json:"date"`
	Runtime     metaRuntime `json:"runtime"`
}

type metaRuntime struct {
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
}
