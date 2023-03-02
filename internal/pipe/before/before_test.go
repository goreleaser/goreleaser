package before

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	defer log.SetLevel(log.InfoLevel)
	os.Exit(m.Run())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	for _, tc := range [][]string{
		nil,
		{},
		{"go version"},
		{"go version", "go list"},
		{`bash -c "go version; echo \"lala spaces and such\""`},
	} {
		ctx := testctx.NewWithCfg(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		require.NoError(t, Pipe{}.Run(ctx))
	}
}

func TestRunPipeInvalidCommand(t *testing.T) {
	ctx := testctx.NewWithCfg(
		config.Project{
			Before: config.Before{
				Hooks: []string{`bash -c "echo \"unterminated command\"`},
			},
		},
	)
	require.EqualError(t, Pipe{}.Run(ctx), "invalid command line string")
}

func TestRunPipeFail(t *testing.T) {
	for err, tc := range map[string][]string{
		"hook failed: go tool foobar: exit status 2; output: go: no such tool \"foobar\"\n": {"go tool foobar"},
		"hook failed: sh ./testdata/foo.sh: exit status 1; output: lalala\n":                {"sh ./testdata/foo.sh"},
	} {
		ctx := testctx.NewWithCfg(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		require.EqualError(t, Pipe{}.Run(ctx), err)
	}
}

func TestRunWithEnv(t *testing.T) {
	f := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, Pipe{}.Run(testctx.NewWithCfg(
		config.Project{
			Env: []string{
				"TEST_FILE=" + f,
			},
			Before: config.Before{
				Hooks: []string{"touch {{ .Env.TEST_FILE }}"},
			},
		},
	)))
	require.FileExists(t, f)
}

func TestInvalidTemplate(t *testing.T) {
	require.EqualError(t, Pipe{}.Run(testctx.NewWithCfg(
		config.Project{
			Before: config.Before{
				Hooks: []string{"touch {{ .fasdsd }"},
			},
		},
	)), `template: tmpl:1: unexpected "}" in operand`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("skip before", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Before: config.Before{
				Hooks: []string{""},
			},
		})
		ctx.SkipBefore = true
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Before: config.Before{
				Hooks: []string{""},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
