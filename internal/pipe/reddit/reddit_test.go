package reddit

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "reddit")
}

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.Announce.Reddit.TitleTemplate, defaultTitleTemplate)
}

func TestAnnounceDisabled(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Announce(ctx))
}

func TestAnnounceInvalidURLTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Reddit: config.Reddit{
				Enabled:     true,
				URLTemplate: "{{ .Foo }",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `announce: failed to announce to reddit: template: tmpl:1: unexpected "}" in operand`)
}

func TestAnnounceInvalidTitleTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Reddit: config.Reddit{
				Enabled:       true,
				TitleTemplate: "{{ .Foo }",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `announce: failed to announce to reddit: template: tmpl:1: unexpected "}" in operand`)
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Reddit: config.Reddit{
				Enabled: true,
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `announce: failed to announce to reddit: env: environment variable "REDDIT_SECRET" should not be empty`)
}

func TestAnnounceSkipAnnounce(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Reddit: config.Reddit{
				Enabled: true,
			},
		},
	})
	ctx.SkipAnnounce = true
	testlib.AssertSkipped(t, Pipe{}.Announce(ctx))
}
