package commitauthor

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		author, err := Get(testctx.NewWithCfg(config.Project{
			Env: []string{"NAME=foo", "MAIL=foo@bar"},
		}), config.CommitAuthor{
			Name:  "{{.Env.NAME}}",
			Email: "{{.Env.MAIL}}",
		})
		require.NoError(t, err)
		require.Equal(t, config.CommitAuthor{
			Name:  "foo",
			Email: "foo@bar",
		}, author)
	})

	t.Run("invalid name tmpl", func(t *testing.T) {
		_, err := Get(
			testctx.New(),
			config.CommitAuthor{
				Name:  "{{.Env.NOPE}}",
				Email: "a",
			})
		require.Error(t, err)
	})

	t.Run("invalid email tmpl", func(t *testing.T) {
		_, err := Get(
			testctx.New(),
			config.CommitAuthor{
				Name:  "a",
				Email: "{{.Env.NOPE}}",
			})
		require.Error(t, err)
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
}
