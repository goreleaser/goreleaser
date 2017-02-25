package context

import "github.com/goreleaser/goreleaser/config"

// GitInfo includes tags and diffs used in some point
type GitInfo struct {
	CurrentTag  string
	PreviousTag string
	Diff        string
}

// Context carries along some data through the pipes
type Context struct {
	Config   config.Project
	Token    string
	Git      GitInfo
	Archives map[string]string
	Version  string
}

// New context
func New(config config.Project) *Context {
	return &Context{
		Config:   config,
		Archives: map[string]string{},
	}
}
