// A module for Goreleaser Dagger functions

package main

type Goreleaser struct {
	// +private
	Source *Directory
	// +private
	GoVersion string
}

func New(
	// The Goreleaser source code to use
	// +optional
	Source *Directory,
	// The Go version to use
	// +default="1.22.3"
	GoVersion string,
) *Goreleaser {
	// TODO: remove
	if Source == nil {
		Source = dag.Git(
			"https://github.com/goreleaser/goreleaser.git",
			GitOpts{KeepGitDir: true},
		).
			Branch("main").
			Tree()
	}
	return &Goreleaser{Source: Source, GoVersion: GoVersion}
}
