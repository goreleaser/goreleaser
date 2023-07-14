// Package metadata provides the pipe implementation that creates an artifacts.json file in the dist folder.
package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe implementation.
type Pipe struct{}

func (Pipe) String() string               { return "storing release metadata" }
func (Pipe) Skip(_ *context.Context) bool { return false }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if err := tmpl.New(ctx).ApplyAll(&ctx.Config.Metadata.ModTimestamp); err != nil {
		return err
	}
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
	_ = ctx.Artifacts.Visit(func(a *artifact.Artifact) error {
		a.TypeS = a.Type.String()
		a.Path = filepath.ToSlash(filepath.Clean(a.Path))
		return nil
	})
	return writeJSON(ctx, ctx.Artifacts.List(), "artifacts.json")
}

func writeJSON(ctx *context.Context, j interface{}, name string) error {
	bts, err := json.Marshal(j)
	if err != nil {
		return err
	}
	path := filepath.Join(ctx.Config.Dist, name)
	log.Log.WithField("file", path).Info("writing")
	if err := os.WriteFile(path, bts, 0o644); err != nil {
		return err
	}

	if ctx.Config.Metadata.ModTimestamp == "" {
		return nil
	}

	modUnix, err := strconv.ParseInt(ctx.Config.Metadata.ModTimestamp, 10, 64)
	if err != nil {
		return err
	}
	modTime := time.Unix(modUnix, 0)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		return fmt.Errorf("failed to change times for %s: %w", path, err)
	}
	return nil
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
