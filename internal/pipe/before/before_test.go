package before

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	m.Run()
	log.SetLevel(log.InfoLevel)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	table := [][]string{
		nil,
		{},
		{"go version"},
		{"go version", "go list"},
	}
	if testlib.InPath("bash") {
		table = append(table, []string{`bash -c "go version; echo \"lala spaces and such\""`})
	}
	for _, tc := range table {
		ctx := testctx.WrapWithCfg(t.Context(),
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			})

		require.NoError(t, Pipe{}.Run(ctx))
	}
}

func TestRunPipeInvalidCommand(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Before: config.Before{
				Hooks: []string{`bash -c "echo \"unterminated command\"`},
			},
		})

	require.EqualError(t, Pipe{}.Run(ctx), "invalid command line string")
}

func TestRunPipeFail(t *testing.T) {
	for _, tc := range []string{
		"go tool foobar",
		"sh ./testdata/foo.sh",
	} {
		ctx := testctx.WrapWithCfg(t.Context(),
			config.Project{
				Before: config.Before{
					Hooks: []string{tc},
				},
			})

		err := Pipe{}.Run(ctx)
		require.Error(t, err)

		de, ok := errors.AsType[gerrors.ErrDetailed](err)
		require.True(t, ok)

		require.Contains(t, de.Messages(), "hook failed")
	}
}

func TestRunWithEnv(t *testing.T) {
	f := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, Pipe{}.Run(testctx.WrapWithCfg(t.Context(),
		config.Project{
			Env: []string{
				"TEST_FILE=" + f,
			},
			Before: config.Before{
				Hooks: []string{testlib.Touch("{{ .Env.TEST_FILE }}")},
			},
		})))
	require.FileExists(t, f)
}

func TestRunRedactsHook(t *testing.T) {
	testlib.SkipIfWindows(t, "uses sh")

	for _, exit := range []string{"0", "1"} {
		t.Run("exit "+exit, func(t *testing.T) {
			var logs bytes.Buffer
			previous := log.Log
			log.Log = log.New(&logs)
			log.SetLevel(log.InfoLevel)
			t.Cleanup(func() { log.Log = previous })

			const secret = "synthetic-registry-password"
			const passphrase = "synthetic-signing-passphrase"
			result := filepath.Join(t.TempDir(), "received")
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Before: config.Before{Hooks: []string{
					`sh -c 'printf "%s" "{{ .Env.REGISTRY_PASSWORD }}" > "{{ .Env.HOOK_RESULT }}"; echo "diagnostic {{ .Env.NFPM_PASSPHRASE }}"; exit ` + exit + `'`,
				}},
			}, testctx.WithEnv(map[string]string{
				"REGISTRY_PASSWORD": secret,
				"NFPM_PASSPHRASE":   passphrase,
				"HOOK_RESULT":       result,
			}))

			err := Pipe{}.Run(ctx)
			if exit == "0" {
				require.NoError(t, err)
			} else {
				de, ok := errors.AsType[gerrors.ErrDetailed](err)
				require.True(t, ok, "expected detailed hook error, got %v", err)
				require.Contains(t, de.Output(), "diagnostic $NFPM_PASSPHRASE")
				require.NotContains(t, de.Output(), passphrase)
			}
			received, err := os.ReadFile(result)
			require.NoError(t, err)
			require.Equal(t, secret, string(received))
			require.Contains(t, logs.String(), "running")
			require.Contains(t, logs.String(), "$REGISTRY_PASSWORD")
			require.Contains(t, logs.String(), "$NFPM_PASSPHRASE")
			require.NotContains(t, logs.String(), secret)
			require.NotContains(t, logs.String(), passphrase)
		})
	}
}

func TestInvalidTemplate(t *testing.T) {
	testlib.RequireTemplateError(t, Pipe{}.Run(testctx.WrapWithCfg(t.Context(),
		config.Project{
			Before: config.Before{
				Hooks: []string{"doesnt-matter {{ .fasdsd }"},
			},
		})))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("skip before", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Before: config.Before{
				Hooks: []string{""},
			},
		}, testctx.Skip(skips.Before))

		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Before: config.Before{
				Hooks: []string{""},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}
