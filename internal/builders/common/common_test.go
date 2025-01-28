package common

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestValidateNonGoConfig(t *testing.T) {
	cases := map[string]config.Build{
		"ldflags": {
			BuildDetails: config.BuildDetails{
				Ldflags: []string{"-a"},
			},
		},
		"goos": {
			Goos: []string{"a"},
		},
		"goarch": {
			Goarch: []string{"a"},
		},
		"goamd64": {
			Goamd64: []string{"a"},
		},
		"go386": {
			Go386: []string{"a"},
		},
		"goarm": {
			Goarm: []string{"a"},
		},
		"goarm64": {
			Goarm64: []string{"a"},
		},
		"gomips": {
			Gomips: []string{"a"},
		},
		"goppc64": {
			Goppc64: []string{"a"},
		},
		"goriscv64": {
			Goriscv64: []string{"a"},
		},
		"ignore": {
			Ignore: []config.IgnoredBuild{{}},
		},
		"overrides": {
			BuildDetailsOverrides: []config.BuildDetailsOverride{{}},
		},
		"buildmode": {
			BuildDetails: config.BuildDetails{
				Buildmode: "a",
			},
		},
		"tags": {
			BuildDetails: config.BuildDetails{
				Tags: []string{"a"},
			},
		},
		"asmflags": {
			BuildDetails: config.BuildDetails{
				Asmflags: []string{"a"},
			},
		},
	}
	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			require.Error(t, ValidateNonGoConfig(v))
		})
	}
}

func TestChTimes(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
		name := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(name, []byte("foo"), 0o644))
		build := config.Build{
			ModTimestamp: "{{.Env.A}}",
		}
		tpl := tmpl.New(testctx.New()).SetEnv("A=" + strconv.FormatInt(modTime.Unix(), 10))
		require.NoError(t, ChTimes(build, tpl, &artifact.Artifact{
			Path: name,
		}))

		st, err := os.Stat(name)
		require.NoError(t, err)
		require.Equal(t, modTime, st.ModTime().UTC())
	})
	t.Run("invalid template", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "file")
		build := config.Build{
			ModTimestamp: "{{.Env.A}}",
		}
		tpl := tmpl.New(testctx.New())
		require.Error(t, ChTimes(build, tpl, &artifact.Artifact{
			Path: name,
		}))
	})
	t.Run("invalid timestamp", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "file")
		build := config.Build{
			ModTimestamp: "invalid",
		}
		tpl := tmpl.New(testctx.New())
		require.Error(t, ChTimes(build, tpl, &artifact.Artifact{
			Path: name,
		}))
	})
	t.Run("empty", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "file")
		build := config.Build{}
		tpl := tmpl.New(testctx.New())
		require.NoError(t, ChTimes(build, tpl, &artifact.Artifact{
			Path: name,
		}))
	})
}

func TestTemplateEnv(t *testing.T) {
	build := config.Build{
		BuildDetails: config.BuildDetails{
			Env: []string{
				"FOO={{.Env.FU}}",
				"BAR={{.Env.FOO}}_{{.Env.FU}}",
				`OS={{- if eq .Os "windows" -}}
					w
				{{- else if eq .Os "darwin" -}}
					d
				{{- else if eq .Os "linux" -}}
					l
				{{- end -}}`,
			},
		},
	}
	tpl := tmpl.New(testctx.New()).SetEnv("FU=foobar").WithArtifact(&artifact.Artifact{
		Goos: "linux",
	})

	got, err := TemplateEnv(build.Env, tpl)
	require.NoError(t, err)
	require.Equal(t, []string{
		"FOO=foobar",
		"BAR=foobar_foobar",
		"OS=l",
	}, got)
}

func TestExec(t *testing.T) {
	require.NoError(t, Exec(testctx.New(), []string{"echo", "$FOO"}, []string{"FOO=foobar"}, "."))
}
