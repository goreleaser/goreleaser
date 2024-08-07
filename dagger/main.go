// A module for Goreleaser Dagger functions

package main

import "github.com/goreleaser/goreleaser/dagger/internal/dagger"

type Goreleaser struct {
	// +private
	Source *dagger.Directory
}

func New(
	// The Goreleaser source code to use
	// +optional
	Source *dagger.Directory,
) *Goreleaser {
	if Source == nil {
		Source = dag.Git(
			"https://github.com/goreleaser/goreleaser.git",
			dagger.GitOpts{KeepGitDir: true},
		).
			Branch("main").
			Tree()
	}
	return &Goreleaser{Source: Source}
}
