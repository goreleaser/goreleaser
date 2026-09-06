package sign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestBinarySignDescription(t *testing.T) {
	require.NotEmpty(t, BinaryPipe{}.String())
}

func TestBinarySignDefault(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
	t.Setenv("GIT_CONFIG_PARAMETERS", "")
	for _, tc := range []struct {
		name     string
		program  string
		expected string
	}{
		{name: "default", expected: "gpg"},
		{name: "configured", program: "not-really-gpg", expected: "not-really-gpg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testlib.Mktmp(t)
			testlib.GitInit(t)
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				BinarySigns: []config.BinarySign{{}},
			})
			setGpg(t, ctx, tc.program)

			require.NoError(t, BinaryPipe{}.Default(ctx))
			require.Equal(t, tc.expected, ctx.Config.BinarySigns[0].Cmd)
			require.Equal(t, defaultSignatureName, ctx.Config.BinarySigns[0].Signature)
			require.Equal(t, []string{"--output", "$signature", "--detach-sig", "$artifact"}, ctx.Config.BinarySigns[0].Args)
			require.Equal(t, "binary", ctx.Config.BinarySigns[0].Artifacts)
		})
	}
}

func TestBinarySignDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{
			{Artifacts: "none"},
		},
	})

	err := BinaryPipe{}.Run(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestBinarySignDisabledDoesNotStopOthers(t *testing.T) {
	dist := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dist, "bin"), []byte("foo"), 0o644))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: dist,
		BinarySigns: []config.BinarySign{
			{ID: "disabled", Artifacts: "none"},
			{ID: "enabled", Artifacts: "binary", Cmd: "false"},
		},
	})
	ctx.Parallelism = 1
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "bin",
		Path: filepath.Join(dist, "bin"),
		Type: artifact.Binary,
	})

	require.NoError(t, BinaryPipe{}.Default(ctx))
	err := BinaryPipe{}.Run(ctx)
	require.ErrorContains(t, err, "exit status 1")
}

func TestBinarySignInvalidOption(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{
			{Artifacts: "archive"},
		},
	})

	err := BinaryPipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: archive")
}

func TestBinarySkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, BinaryPipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Sign))
		require.True(t, BinaryPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			BinarySigns: []config.BinarySign{
				{},
			},
		})

		require.False(t, BinaryPipe{}.Skip(ctx))
	})
}

func TestBinaryDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{
			{Cmd: "cosign"},
			{Cmd: "gpg2"},
		},
	})

	require.Equal(t, []string{"cosign", "gpg2"}, BinaryPipe{}.Dependencies(ctx))
}

func TestBinarySign(t *testing.T) {
	testlib.CheckPath(t, "gpg")
	testlib.SkipIfWindows(t, "tries to use /usr/bin/gpg-agent")
	doTest := func(tb testing.TB, sign config.BinarySign) []*artifact.Artifact {
		tb.Helper()
		// chdir: the templated-signature case below uses a signature name
		// that is relative to the working directory.
		tmpdir := testlib.Mktmp(tb)

		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			BinarySigns: []config.BinarySign{sign},
		})

		require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "bin1"), []byte("foo"), 0o644))
		require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "bin2"), []byte("foo"), 0o644))

		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "bin1",
			Path:   filepath.Join(tmpdir, "bin1"),
			Type:   artifact.Binary,
			Goarch: "amd64",
			Extra: map[string]any{
				artifact.ExtraID: "foo",
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "bin2",
			Path:   filepath.Join(tmpdir, "bin2"),
			Type:   artifact.Binary,
			Goarch: "arm64",
			Extra: map[string]any{
				artifact.ExtraID: "bar",
			},
		})

		pipe := BinaryPipe{}
		require.NoError(tb, pipe.Default(ctx))

		for i := range ctx.Config.BinarySigns {
			ctx.Config.BinarySigns[i].Env = append(
				ctx.Config.BinarySigns[i].Env,
				"GNUPGHOME="+keyring,
			)
		}
		require.NoError(tb, pipe.Run(ctx))
		return ctx.Artifacts.
			Filter(artifact.ByType(artifact.Signature)).
			List()
	}

	t.Run("default", func(t *testing.T) {
		sigs := doTest(t, config.BinarySign{})
		require.Len(t, sigs, 2)
	})

	t.Run("templated-signature", func(t *testing.T) {
		sigs := doTest(t, config.BinarySign{
			Signature: "prefix_{{ .Arch }}_suffix",
			Cmd:       "/bin/sh",
			Args: []string{
				"-c",
				`echo "siging signature=$signature artifact=$artifact" > "$signature"`,
				"shell",
			},
		})
		require.Len(t, sigs, 2)
		require.Equal(
			t,
			[]*artifact.Artifact{
				{Name: "prefix_amd64_suffix", Path: "prefix_amd64_suffix", Type: 13, Extra: artifact.Extras{"ID": "default"}},
				{Name: "prefix_arm64_suffix", Path: "prefix_arm64_suffix", Type: 13, Extra: artifact.Extras{"ID": "default"}},
			},
			sigs,
		)
	})

	t.Run("filter", func(t *testing.T) {
		sigs := doTest(t, config.BinarySign{
			ID:  "bar",
			IDs: []string{"bar"},
		})
		require.Len(t, sigs, 1)
	})
}

func TestBinarySignUniversalBinary(t *testing.T) {
	testlib.SkipIfWindows(t, "uses /bin/sh")
	dist := t.TempDir()
	calls := filepath.Join(dist, "calls")
	for _, name := range []string{"binary", "universal", "excluded"} {
		require.NoError(t, os.WriteFile(filepath.Join(dist, name), []byte("foo"), 0o644))
	}

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: dist,
		BinarySigns: []config.BinarySign{
			{
				Artifacts: "binary",
				IDs:       []string{"foo"},
				Signature: "{{ .ArtifactName }}_{{ .Os }}_{{ .Arch }}.sig",
				Cmd:       "/bin/sh",
				Args: []string{
					"-c",
					`printf "%s\n" "$artifact" >> "$CALLS" && printf signature > "$signature"`,
				},
				Env: []string{"CALLS=" + calls},
			},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "binary",
		Path:   filepath.Join(dist, "binary"),
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "app",
			artifact.ExtraID:     "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "universal",
		Path:   filepath.Join(dist, "universal"),
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UniversalBinary,
		Extra: map[string]any{
			artifact.ExtraBinary:   "app",
			artifact.ExtraID:       "foo",
			artifact.ExtraReplaces: true,
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "excluded",
		Path:   filepath.Join(dist, "excluded"),
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UniversalBinary,
		Extra: map[string]any{
			artifact.ExtraBinary:   "app",
			artifact.ExtraID:       "bar",
			artifact.ExtraReplaces: true,
		},
	})

	require.NoError(t, BinaryPipe{}.Default(ctx))
	require.NoError(t, BinaryPipe{}.Run(ctx))

	callBytes, err := os.ReadFile(calls)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		filepath.Join(dist, "binary"),
		filepath.Join(dist, "universal"),
	}, strings.Split(strings.TrimSpace(string(callBytes)), "\n"))

	sigs := ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List()
	require.Len(t, sigs, 2)
	require.ElementsMatch(t, []string{
		"binary_darwin_amd64.sig",
		"universal_darwin_all.sig",
	}, []string{sigs[0].Name, sigs[1].Name})
	require.NoFileExists(t, filepath.Join(dist, "excluded_darwin_all.sig"))
}

// When `replace` is true the per-arch binaries are gone, and the universal
// binary carries `universal_binaries.id` (the project name by default), not the
// build IDs it was made from. This is the same contract as `archives.ids`.
func TestBinarySignUniversalBinaryReplaced(t *testing.T) {
	testlib.SkipIfWindows(t, "uses /bin/sh")

	newContext := func(tb testing.TB, ids []string) (*context.Context, string) {
		tb.Helper()

		dist := tb.TempDir()
		require.NoError(tb, os.WriteFile(filepath.Join(dist, "universal"), []byte("foo"), 0o644))
		ctx := testctx.WrapWithCfg(tb.Context(), config.Project{
			Dist: dist,
			BinarySigns: []config.BinarySign{
				{
					Artifacts: "binary",
					IDs:       ids,
					Signature: "{{ .ArtifactName }}.sig",
					Cmd:       "/bin/sh",
					Args:      []string{"-c", `printf signature > "$signature"`},
				},
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "universal",
			Path:   filepath.Join(dist, "universal"),
			Goos:   "darwin",
			Goarch: "all",
			Type:   artifact.UniversalBinary,
			Extra: map[string]any{
				artifact.ExtraBinary:   "app",
				artifact.ExtraID:       "proj",
				artifact.ExtraReplaces: true,
			},
		})
		return ctx, dist
	}

	t.Run("no ids signs it", func(t *testing.T) {
		ctx, dist := newContext(t, nil)
		require.NoError(t, BinaryPipe{}.Default(ctx))
		require.NoError(t, BinaryPipe{}.Run(ctx))
		require.Len(t, ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List(), 1)
		require.FileExists(t, filepath.Join(dist, "universal.sig"))
	})

	t.Run("universal binary id signs it", func(t *testing.T) {
		ctx, dist := newContext(t, []string{"proj"})
		require.NoError(t, BinaryPipe{}.Default(ctx))
		require.NoError(t, BinaryPipe{}.Run(ctx))
		require.Len(t, ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List(), 1)
		require.FileExists(t, filepath.Join(dist, "universal.sig"))
	})

	t.Run("build ids do not sign it", func(t *testing.T) {
		ctx, dist := newContext(t, []string{"darwin-amd64", "darwin-arm64"})
		require.NoError(t, BinaryPipe{}.Default(ctx))
		require.NoError(t, BinaryPipe{}.Run(ctx))
		require.Empty(t, ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List())
		require.NoFileExists(t, filepath.Join(dist, "universal.sig"))
	})
}
