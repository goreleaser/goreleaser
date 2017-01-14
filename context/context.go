package context

import "github.com/goreleaser/goreleaser/config"

// GitInfo includes tags and diffs used in some point
type GitInfo struct {
	CurrentTag  string
	PreviousTag string
	Diff        string
}

type Repo struct {
	Owner, Name string
}

type Context struct {
	Config      *config.ProjectConfig
	Token       *string
	Git         *GitInfo
	ReleaseRepo *Repo
	BrewRepo    *Repo
	Archives    map[string]string
}

func New(config config.ProjectConfig) *Context {
	return &Context{
		Config:   &config,
		Archives: map[string]string{},
	}
}
