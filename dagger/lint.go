package main

import (
	"context"
	"fmt"
)

// Lint Goreleaser
func (g *Goreleaser) Lint(
	ctx context.Context,
	// Version of golangci-lint to use
	// +default="v1.58.1"
	golangciLintVersion string,
) (string, error) {
	lintImage := fmt.Sprintf("golangci/golangci-lint:%s", golangciLintVersion)
	return dag.Container().From(lintImage).
		WithMountedDirectory("/src", g.Source).
		WithWorkdir("/src").
		WithExec([]string{
			"golangci-lint",
			"run",
			"--config",
			"./.golangci.yaml",
			"./...",
		}).
		Stdout(ctx)
}
