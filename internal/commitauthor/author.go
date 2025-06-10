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

// Get templates the commit author and returns the filled fields.
func Get(ctx *context.Context, og config.CommitAuthor) (config.CommitAuthor, error) {
	var author config.CommitAuthor
	var err error

	author.Name, err = tmpl.New(ctx).Apply(og.Name)
	if err != nil {
		return author, err
	}
	author.Email, err = tmpl.New(ctx).Apply(og.Email)
	if err != nil {
		return author, err
	}

	// Apply templates to signing configuration
	author.Signing.Enabled = og.Signing.Enabled
	author.Signing.Key, err = tmpl.New(ctx).Apply(og.Signing.Key)
	if err != nil {
		return author, err
	}
	author.Signing.Program, err = tmpl.New(ctx).Apply(og.Signing.Program)
	if err != nil {
		return author, err
	}
	
	return author.Signing.Format, err = tmpl.New(ctx).Apply(og.Signing.Format)
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
