// A module for Goreleaser Dagger functions

package main

type Goreleaser struct {
	// +private
	Source *Directory
}

func New(
	// The Goreleaser source code to use
	// +optional
	Source *Directory,
) *Goreleaser {
	if Source == nil {
		Source = dag.Git(
			"https://github.com/goreleaser/goreleaser.git",
			GitOpts{KeepGitDir: true},
		).
			Branch("main").
			Tree()
	}
	return &Goreleaser{Source: Source}
}
