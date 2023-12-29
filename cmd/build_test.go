package cmd

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	setup(t)
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestBuildSingleTarget(t *testing.T) {
	setup(t)
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated", "--single-target"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestBuildInvalidConfig(t *testing.T) {
	setup(t)
	createFile(t, "goreleaser.yml", "foo: bar")
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 1: field foo not found in type config.Project")
}

func TestBuildBrokenProject(t *testing.T) {
	setup(t)
	createFile(t, "main.go", "not a valid go file")
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2"})
	require.EqualError(t, cmd.cmd.Execute(), "failed to parse dir: .: main.go:1:1: expected 'package', found not")
}

func TestSetupPipeline(t *testing.T) {
	t.Run("regular", func(t *testing.T) {
		require.Equal(
			t,
			pipeline.BuildCmdPipeline,
			setupPipeline(testctx.New(), buildOpts{}),
		)
	})

	t.Run("single-target", func(t *testing.T) {
		require.Equal(
			t,
			pipeline.BuildCmdPipeline,
			setupPipeline(testctx.New(), buildOpts{
				singleTarget: true,
			}),
		)
	})

	t.Run("single-target and id", func(t *testing.T) {
		require.Equal(
			t,
			pipeline.BuildCmdPipeline,
			setupPipeline(testctx.New(), buildOpts{
				singleTarget: true,
				ids:          []string{"foo"},
			}),
		)
	})

	t.Run("single-target and id, given output", func(t *testing.T) {
		require.Equal(
			t,
			append(pipeline.BuildCmdPipeline, withOutputPipe{"foobar"}),
			setupPipeline(testctx.New(), buildOpts{
				singleTarget: true,
				ids:          []string{"foo"},
				output:       ".",
			}),
		)
	})

	t.Run("single-target and single build on config", func(t *testing.T) {
		require.Equal(
			t,
			pipeline.BuildCmdPipeline,
			setupPipeline(
				testctx.NewWithCfg(config.Project{
					Builds: []config.Build{{}},
				}),
				buildOpts{
					singleTarget: true,
				},
			),
		)
	})

	t.Run("single-target, id and output", func(t *testing.T) {
		require.Equal(
			t,
			append(pipeline.BuildCmdPipeline, withOutputPipe{"foobar"}),
			setupPipeline(
				testctx.New(),
				buildOpts{
					singleTarget: true,
					ids:          []string{"foo"},
					output:       "foobar",
				},
			),
		)
	})

	t.Run("single-target, single build on config and output", func(t *testing.T) {
		require.Equal(
			t,
			append(pipeline.BuildCmdPipeline, withOutputPipe{"zaz"}),
			setupPipeline(
				testctx.NewWithCfg(config.Project{
					Builds: []config.Build{{}},
				}),
				buildOpts{
					singleTarget: true,
					output:       "zaz",
				},
			),
		)
	})
}

func TestBuildFlags(t *testing.T) {
	setup := func(opts buildOpts) *context.Context {
		ctx := testctx.New()
		require.NoError(t, setupBuildContext(ctx, opts))
		return ctx
	}

	t.Run("action", func(t *testing.T) {
		ctx := setup(buildOpts{})
		require.Equal(t, context.ActionBuild, ctx.Action)
	})

	t.Run("snapshot", func(t *testing.T) {
		ctx := setup(buildOpts{
			snapshot: true,
		})
		require.True(t, ctx.Snapshot)
		requireAll(t, ctx, skips.Validate)
		require.True(t, ctx.SkipTokenCheck)
	})

	t.Run("skips (old)", func(t *testing.T) {
		ctx := setup(buildOpts{
			skipValidate:  true,
			skipPostHooks: true,
		})
		requireAll(t, ctx, skips.Validate, skips.PostBuildHooks)
		require.True(t, ctx.SkipTokenCheck)
	})

	t.Run("skips", func(t *testing.T) {
		ctx := setup(buildOpts{
			skips: []string{
				string(skips.Validate),
				string(skips.PostBuildHooks),
			},
		})
		requireAll(t, ctx, skips.Validate, skips.PostBuildHooks)
		require.True(t, ctx.SkipTokenCheck)
	})

	t.Run("parallelism", func(t *testing.T) {
		require.Equal(t, 1, setup(buildOpts{
			parallelism: 1,
		}).Parallelism)
	})

	t.Run("rm dist", func(t *testing.T) {
		require.True(t, setup(buildOpts{
			clean: true,
		}).Clean)
	})

	t.Run("id", func(t *testing.T) {
		t.Run("match", func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					{
						ID: "default",
					},
					{
						ID: "foo",
					},
				},
			})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				ids: []string{"foo"},
			}))
		})

		t.Run("match-multiple", func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					{
						ID: "default",
					},
					{
						ID: "foo",
					},
				},
			})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				ids: []string{"foo", "default"},
			}))
		})

		t.Run("match-partial", func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					{
						ID: "default",
					},
					{
						ID: "foo",
					},
				},
			})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				ids: []string{"foo", "notdefault"},
			}))
		})

		t.Run("dont match", func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					{
						ID: "foo",
					},
					{
						ID: "bazz",
					},
				},
			})
			require.EqualError(t, setupBuildContext(ctx, buildOpts{
				ids: []string{"bar", "fooz"},
			}), "no builds with ids bar, fooz")
		})

		t.Run("default config", func(t *testing.T) {
			ctx := testctx.New()
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				ids: []string{"aaa"},
			}))
		})

		t.Run("single build config", func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					{
						ID: "foo",
					},
				},
			})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				ids: []string{"not foo but doesnt matter"},
			}))
		})
	})
}
