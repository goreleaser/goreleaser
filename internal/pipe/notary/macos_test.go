package notary

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestMacOSString(t *testing.T) {
	require.NotEmpty(t, MacOS{}.String())
}

func TestMacOSSkip(t *testing.T) {
	p := MacOS{}
	t.Run("skip notarize", func(t *testing.T) {
		require.True(t,
			p.Skip(testctx.NewWithCfg(config.Project{
				Notarize: config.Notarize{
					MacOS: []config.MacOSSignNotarize{
						{},
					},
				},
			}, testctx.Skip(skips.Notarize))))
	})
	t.Run("skip no configs", func(t *testing.T) {
		require.True(t,
			p.Skip(testctx.NewWithCfg(config.Project{})))
	})
	t.Run("dont skip", func(t *testing.T) {
		require.False(t,
			p.Skip(testctx.NewWithCfg(config.Project{
				Notarize: config.Notarize{
					MacOS: []config.MacOSSignNotarize{
						{},
					},
				},
			})))
	})
}

func TestMacOSDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
		Notarize: config.Notarize{
			MacOS: []config.MacOSSignNotarize{
				{},
				{
					Notarize: config.MacOSNotarize{
						Timeout: time.Second,
					},
				},
				{
					IDs: []string{"hi"},
				},
			},
		},
	})
	require.NoError(t, MacOS{}.Default(ctx))
	require.Equal(t, []config.MacOSSignNotarize{
		{
			IDs: []string{"foo"},
			Notarize: config.MacOSNotarize{
				Timeout: 10 * time.Minute,
			},
		},
		{
			IDs: []string{"foo"},
			Notarize: config.MacOSNotarize{
				Timeout: time.Second,
			},
		},
		{
			IDs: []string{"hi"},
			Notarize: config.MacOSNotarize{
				Timeout: 10 * time.Minute,
			},
		},
	}, ctx.Config.Notarize.MacOS)
}
