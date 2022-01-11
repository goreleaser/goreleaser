package commitauthor

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		author, err := Get(context.New(config.Project{
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
			context.New(config.Project{}),
			config.CommitAuthor{
				Name:  "{{.Env.NOPE}}",
				Email: "a",
			})
		require.Error(t, err)
	})

	t.Run("invalid email tmpl", func(t *testing.T) {
		_, err := Get(
			context.New(config.Project{}),
			config.CommitAuthor{
				Name:  "a",
				Email: "{{.Env.NOPE}}",
			})
		require.Error(t, err)
	})
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Equal(t, Default(config.CommitAuthor{}), config.CommitAuthor{
			Name:  defaultName,
			Email: defaultEmail,
		})
	})

	t.Run("no name", func(t *testing.T) {
		require.Equal(t, Default(config.CommitAuthor{
			Email: "a",
		}), config.CommitAuthor{
			Name:  defaultName,
			Email: "a",
		})
	})

	t.Run("no email", func(t *testing.T) {
		require.Equal(t, Default(config.CommitAuthor{
			Name: "a",
		}), config.CommitAuthor{
			Name:  "a",
			Email: defaultEmail,
		})
	})
}
