package notary

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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

func TestMacOSRun(t *testing.T) {
	t.Run("bad tmpl", func(t *testing.T) {
		for name, fn := range map[string]func(ctx *context.Context){
			"enabled": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "{{.Nope}}",
				})
			},
			"certificate": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "true",
					Sign: config.MacOSSign{
						Certificate: "{{.Nope}}",
					},
				})
			},
			"password": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "true",
					Sign: config.MacOSSign{
						Password: "{{.Nope}}",
					},
				})
			},
			"entitlements": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "true",
					Sign: config.MacOSSign{
						Entitlements: "{{.Nope}}",
					},
				})
			},
			"key": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "true",
					Notarize: config.MacOSNotarize{
						Key: "{{.Nope}}",
					},
				})
			},
			"keyid": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "true",
					Notarize: config.MacOSNotarize{
						KeyID: "{{.Nope}}",
					},
				})
			},
			"issuerid": func(ctx *context.Context) {
				ctx.Config.Notarize.MacOS = append(ctx.Config.Notarize.MacOS, config.MacOSSignNotarize{
					Enabled: "true",
					Notarize: config.MacOSNotarize{
						IssuerID: "{{.Nope}}",
					},
				})
			},
		} {
			t.Run(name, func(t *testing.T) {
				ctx := testctx.NewWithCfg(config.Project{
					Notarize: config.Notarize{
						MacOS: []config.MacOSSignNotarize{
							{},
						},
					},
				})
				fn(ctx)
				testlib.RequireTemplateError(t, MacOS{}.Run(ctx))
			})
		}
	})
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Notarize: config.Notarize{
				MacOS: []config.MacOSSignNotarize{
					{},
					{
						Enabled: "{{.Env.SKIP}}",
					},
				},
			},
		}, testctx.WithEnv(map[string]string{"SKIP": "false"}))
		testlib.AssertSkipped(t, MacOS{}.Run(ctx))
	})
}
