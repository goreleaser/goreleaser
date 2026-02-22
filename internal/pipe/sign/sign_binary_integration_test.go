//go:build integration

package sign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestIntegrationBinarySign(t *testing.T) {
	testlib.CheckPath(t, "gpg")
	testlib.SkipIfWindows(t, "tries to use /usr/bin/gpg-agent")
	doTest := func(tb testing.TB, sign config.BinarySign) []*artifact.Artifact {
		tb.Helper()
		tmpdir := tb.TempDir()

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
				`echo "siging signature=$signature artifact=$artifact"`,
				"shell",
			},
		})
		require.Len(t, sigs, 2)
		require.Equal(t,
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
