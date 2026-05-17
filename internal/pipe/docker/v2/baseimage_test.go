package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestParseBaseImage(t *testing.T) {
	for file, want := range map[string]string{
		"empty":                  "",
		"comment":                "",
		"no-from":                "",
		"simple":                 "alpine:3.20",
		"with-digest":            "alpine@sha256:abc123",
		"with-platform-flag":     "alpine:3.20",
		"multiple-flags":         "alpine:3.20",
		"multi-stage":            "alpine:3.20",
		"follows-alias":          "alpine:3.20",
		"alias-chain":            "alpine:3.20",
		"alias-case-insensitive": "alpine:3.20",
		"arg-simple":             "alpine:3.20",
		"arg-with-default":       "alpine:3.20",
		"arg-dollar-form":        "alpine:3.20",
		"arg-after-from":         "alpine:3.19",
		"line-continuation":      "alpine:3.20",
		"scratch":                "scratch",
		"lowercase-from":         "alpine:3.20",
		"quoted-arg-default":     "alpine:3.20",
	} {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata", "dockerfiles", file))
			require.NoError(t, err)
			require.Equal(t, want, parseBaseImage(string(content)))
		})
	}
}

func TestGetBaseImage(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := getBaseImage(testctx.Wrap(t.Context()), "nope.Dockerfile")
		require.Error(t, err)
	})

	t.Run("scratch", func(t *testing.T) {
		img, err := getBaseImage(testctx.Wrap(t.Context()), filepath.Join("testdata", "dockerfiles", "scratch"))
		require.ErrorIs(t, err, errNoBaseImage)
		require.Empty(t, img.name)
		require.Empty(t, img.digest)
	})

	t.Run("digest pinned in FROM", func(t *testing.T) {
		const ref = "alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1"
		img, err := getBaseImage(testctx.Wrap(t.Context()), filepath.Join("testdata", "dockerfiles", "pinned-digest"))
		require.NoError(t, err)
		require.Equal(t, ref, img.name)
		require.Equal(t, "sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1", img.digest)
	})

	t.Run("digest resolution error returns base", func(t *testing.T) {
		img, err := getBaseImage(testctx.Wrap(t.Context()), filepath.Join("testdata", "dockerfiles", "unknown-image"))
		require.Error(t, err)
		require.Equal(t, "goreleaser-nonexistent-image:nope", img.name)
		require.Empty(t, img.digest)
	})
}

func TestMakeArgsWithBaseImage(t *testing.T) {
	const ref = "alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1"
	const digest = "sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1"
	da, err := makeArgs(
		testctx.Wrap(t.Context()),
		config.DockerV2{
			Dockerfile: filepath.Join("testdata", "dockerfiles", "pinned-digest"),
			Images:     []string{"ghcr.io/foo/bar"},
			Tags:       []string{"latest"},
			Platforms:  []string{"linux/amd64"},
			Annotations: map[string]string{
				"org.opencontainers.image.base.name":   "{{.BaseImage}}",
				"org.opencontainers.image.base.digest": "{{.BaseImageDigest}}",
			},
		},
		nil,
	)
	require.NoError(t, err)
	require.Contains(t, da.args, "org.opencontainers.image.base.name="+ref)
	require.Contains(t, da.args, "org.opencontainers.image.base.digest="+digest)
}
