package commitauthor

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		author, err := Get(testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"NAME=foo", "MAIL=foo@bar"},
		}),

			config.CommitAuthor{
				Name:  "{{.Env.NAME}}",
				Email: "{{.Env.MAIL}}",
			})
		require.NoError(t, err)
		require.Equal(t, config.CommitAuthor{
			Name:  "foo",
			Email: "foo@bar",
		}, author)
	})

	t.Run("valid with signing", func(t *testing.T) {
		author, err := Get(testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"NAME=foo", "MAIL=foo@bar", "SIGNING_KEY=ABC123", "GPG_PROGRAM=/usr/bin/gpg"},
		}),

			config.CommitAuthor{
				Name:  "{{.Env.NAME}}",
				Email: "{{.Env.MAIL}}",
				Signing: config.CommitSigning{
					Enabled: true,
					Key:     "{{.Env.SIGNING_KEY}}",
					Program: "{{.Env.GPG_PROGRAM}}",
					Format:  "openpgp",
				},
			})
		require.NoError(t, err)
		require.Equal(t, config.CommitAuthor{
			Name:  "foo",
			Email: "foo@bar",
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ABC123",
				Program: "/usr/bin/gpg",
				Format:  "openpgp",
			},
		}, author)
	})

	t.Run("invalid name tmpl", func(t *testing.T) {
		_, err := Get(
			testctx.Wrap(t.Context()),
			config.CommitAuthor{
				Name:  "{{.Env.NOPE}}",
				Email: "a",
			})
		require.Error(t, err)
	})

	t.Run("invalid email tmpl", func(t *testing.T) {
		_, err := Get(
			testctx.Wrap(t.Context()),
			config.CommitAuthor{
				Name:  "a",
				Email: "{{.Env.NOPE}}",
			})
		require.Error(t, err)
	})

	t.Run("invalid signing key tmpl", func(t *testing.T) {
		_, err := Get(
			testctx.Wrap(t.Context()),
			config.CommitAuthor{
				Name:  "a",
				Email: "b",
				Signing: config.CommitSigning{
					Enabled: true,
					Key:     "{{.Env.NOPE}}",
				},
			})
		require.Error(t, err)
	})

	t.Run("invalid signing program tmpl", func(t *testing.T) {
		_, err := Get(
			testctx.Wrap(t.Context()),
			config.CommitAuthor{
				Name:  "a",
				Email: "b",
				Signing: config.CommitSigning{
					Enabled: true,
					Program: "{{.Env.NOPE}}",
				},
			})
		require.Error(t, err)
	})

	t.Run("use github app token", func(t *testing.T) {
		author, err := Get(testctx.Wrap(t.Context()), config.CommitAuthor{
			UseGitHubAppToken: true,
		})
		require.NoError(t, err)
		require.Equal(t, config.CommitAuthor{
			UseGitHubAppToken: true,
		}, author)
	})
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:  defaultName,
			Email: defaultEmail,
		}, Default(config.CommitAuthor{}))
	})

	t.Run("no name", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:  defaultName,
			Email: "a",
		}, Default(config.CommitAuthor{
			Email: "a",
		}))
	})

	t.Run("no email", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:  "a",
			Email: defaultEmail,
		}, Default(config.CommitAuthor{
			Name: "a",
		}))
	})

	t.Run("signing enabled without format", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:  defaultName,
			Email: defaultEmail,
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ABC123",
				Format:  "openpgp",
			},
		}, Default(config.CommitAuthor{
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ABC123",
			},
		}))
	})

	t.Run("signing disabled", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:  defaultName,
			Email: defaultEmail,
			Signing: config.CommitSigning{
				Enabled: false,
				Key:     "ABC123",
			},
		}, Default(config.CommitAuthor{
			Signing: config.CommitSigning{
				Enabled: false,
				Key:     "ABC123",
			},
		}))
	})

	t.Run("signing with custom format", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:  defaultName,
			Email: defaultEmail,
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ABC123",
				Format:  "ssh",
			},
		}, Default(config.CommitAuthor{
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ABC123",
				Format:  "ssh",
			},
		}))
	})

	t.Run("use github app token", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:              defaultName,
			Email:             defaultEmail,
			UseGitHubAppToken: true,
		}, Default(config.CommitAuthor{
			UseGitHubAppToken: true,
		}))
	})

	t.Run("use github app token with name and email set", func(t *testing.T) {
		require.Equal(t, config.CommitAuthor{
			Name:              "explicit-name",
			Email:             "explicit@email.com",
			UseGitHubAppToken: true,
		}, Default(config.CommitAuthor{
			Name:              "explicit-name",
			Email:             "explicit@email.com",
			UseGitHubAppToken: true,
		}))
	})
}
