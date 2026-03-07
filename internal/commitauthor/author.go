// Package commitauthor provides common commit author functionality.
package commitauthor

import (
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultName  = "goreleaserbot"
	defaultEmail = "bot@goreleaser.com"
)

// Get templates the commit author and returns a new [config.CommitAuthor].
func Get(ctx *context.Context, og config.CommitAuthor) (config.CommitAuthor, error) {
	author := config.CommitAuthor{
		Name:              og.Name,
		Email:             og.Email,
		UseGitHubAppToken: og.UseGitHubAppToken,
		Signing: config.CommitSigning{
			Enabled: og.Signing.Enabled,
			Key:     og.Signing.Key,
			Program: og.Signing.Program,
			Format:  og.Signing.Format,
		},
	}
	if err := tmpl.New(ctx).ApplyAll(
		&author.Name,
		&author.Email,
		&author.Signing.Key,
		&author.Signing.Program,
		&author.Signing.Format,
	); err != nil {
		return config.CommitAuthor{}, err
	}
	return author, nil
}

// Default sets the default commit author name and email.
func Default(og config.CommitAuthor) config.CommitAuthor {
	if og.Name == "" {
		og.Name = defaultName
	}
	if og.Email == "" {
		og.Email = defaultEmail
	}

	// set default signing format if enabled but format not specified
	if og.Signing.Enabled && og.Signing.Format == "" {
		og.Signing.Format = "openpgp"
	}

	return og
}
