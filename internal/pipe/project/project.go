// Package project sets "high level" defaults related to the project.
package project

import "github.com/goreleaser/goreleaser/pkg/context"

// Pipe implemens defaulter to set the project name
type Pipe struct{}

func (Pipe) String() string {
	return "project name"
}

// Default set project defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.ProjectName == "" {
		switch {
		case ctx.Config.Release.GitHub.Name != "":
			ctx.Config.ProjectName = ctx.Config.Release.GitHub.Name
		case ctx.Config.Release.GitLab.Name != "":
			ctx.Config.ProjectName = ctx.Config.Release.GitLab.Name
		case ctx.Config.Release.Gitea.Name != "":
			ctx.Config.ProjectName = ctx.Config.Release.Gitea.Name
		}
	}
	return nil
}
