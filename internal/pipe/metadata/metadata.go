// Package metadata provides the pipe implementation that creates an artifacts.json file in the dist folder.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type (
	// Pipe implementation.
	Pipe struct{}
	// MetaPipe implementation.
	MetaPipe struct{}
	// ArtifactsPipe implementation.
	ArtifactsPipe struct{}
)

func (Pipe) String() string { return "setting up metadata" }
func (Pipe) Run(ctx *context.Context) error {
	return tmpl.New(ctx).ApplyAll(&ctx.Config.Metadata.ModTimestamp)
}

func (MetaPipe) String() string                 { return "storing release metadata" }
func (MetaPipe) Run(ctx *context.Context) error { return writeMetadata(ctx) }

func (ArtifactsPipe) String() string                 { return "storing artifacts metadata" }
func (ArtifactsPipe) Run(ctx *context.Context) error { return writeArtifacts(ctx) }

func writeMetadata(ctx *context.Context) error {
	const name = "metadata.json"
	path, err := writeJSON(ctx, metadata{
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
	}, name)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: name,
		Path: path,
		Type: artifact.Metadata,
	})
	return err
}

func writeArtifacts(ctx *context.Context) error {
	_ = ctx.Artifacts.Visit(func(a *artifact.Artifact) error {
		a.TypeS = a.Type.String()
		a.Path = filepath.ToSlash(filepath.Clean(a.Path))
		return nil
	})
	_, err := writeJSON(ctx, ctx.Artifacts.List(), "artifacts.json")
	return err
}

func writeJSON(ctx *context.Context, j interface{}, name string) (string, error) {
	bts, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	path := filepath.Join(ctx.Config.Dist, name)
	log.Log.WithField("file", path).Info("writing")
	if err := os.WriteFile(path, bts, 0o644); err != nil {
		return "", err
	}

	return path, gio.Chtimes(path, ctx.Config.Metadata.ModTimestamp)
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
