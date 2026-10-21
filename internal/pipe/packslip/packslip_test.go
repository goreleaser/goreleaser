package packslip

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	packslip "github.com/jdx/packslip/go"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{GitHub: config.Repo{Owner: "acme", Name: "tool"}},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "packslip.sigstore.json", ctx.Config.Packslip.NameTemplate)
	require.Equal(t, "github.com/acme/tool", ctx.Config.Packslip.Project)
}

func TestSkip(t *testing.T) {
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Packslip))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("not enabled", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.True(t, pipe.IsSkip(Pipe{}.Run(ctx)))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Packslip: config.Packslip{Enabled: "true"}}, testctx.Snapshot)
		require.ErrorIs(t, Pipe{}.Run(ctx), pipe.ErrSnapshotEnabled)
	})
	t.Run("skip publish", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Packslip: config.Packslip{Enabled: "true"}}, testctx.Skip(skips.Publish))
		require.ErrorIs(t, Pipe{}.Run(ctx), pipe.ErrSkipPublishEnabled)
	})
}

func fakeToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func TestIdentityFromToken(t *testing.T) {
	t.Run("github actions", func(t *testing.T) {
		id, err := identityFromToken(fakeToken(t, map[string]any{
			"iss":              "https://token.actions.githubusercontent.com",
			"job_workflow_ref": "acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0",
		}))
		require.NoError(t, err)
		require.Equal(t, packslip.Identity{
			Scheme: "sigstore-oidc",
			KeyID:  "https://github.com/acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0",
			Issuer: "https://token.actions.githubusercontent.com",
		}, id)
	})
	t.Run("verified email", func(t *testing.T) {
		id, err := identityFromToken(fakeToken(t, map[string]any{
			"iss": "https://accounts.example.com", "email": "me@example.com", "email_verified": true,
		}))
		require.NoError(t, err)
		require.Equal(t, "me@example.com", id.KeyID)
	})
	t.Run("unverified email", func(t *testing.T) {
		_, err := identityFromToken(fakeToken(t, map[string]any{
			"iss": "https://accounts.example.com", "email": "me@example.com",
		}))
		require.Error(t, err)
	})
	t.Run("not a jwt", func(t *testing.T) {
		_, err := identityFromToken("nope")
		require.Error(t, err)
	})
}

func TestOIDCTokenMissing(t *testing.T) {
	t.Setenv("SIGSTORE_ID_TOKEN", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")
	_, err := oidcToken(testctx.Wrap(t.Context()))
	require.ErrorContains(t, err, "id-token: write")
}

func TestDescribe(t *testing.T) {
	for name, tt := range map[string]struct {
		in   artifact.Artifact
		want packslip.Artifact
	}{
		"linux archive, static": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool_1.0.0_linux_amd64.tar.gz",
				Goos: "linux", Goarch: "amd64",
				Extra: map[string]any{
					artifact.ExtraFormat:    "tar.gz",
					artifact.ExtraWrappedIn: "tool_1.0.0",
					artifact.ExtraBinaries:  []string{"tool"},
				},
			},
			want: packslip.Artifact{
				Name: "tool_1.0.0_linux_amd64.tar.gz", OS: "linux", Arch: "x86_64", Format: "tar.gz",
				URL:      "https://example.com/x",
				Bin:      []packslip.Bin{packslip.NewBin("tool_1.0.0/tool")},
				Requires: &packslip.Requires{Libs: &[]string{}},
			},
		},
		"linux archive, cgo": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool.tar.gz", Goos: "linux", Goarch: "arm64",
				Extra: map[string]any{
					artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"tool"},
					artifact.ExtranDynLink: true,
				},
			},
			want: packslip.Artifact{
				Name: "tool.tar.gz", OS: "linux", Arch: "aarch64", Libc: "gnu", Format: "tar.gz",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool")},
			},
		},
		"windows zip": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool.zip", Goos: "windows", Goarch: "386",
				Extra: map[string]any{artifact.ExtraFormat: "zip", artifact.ExtraBinaries: []string{"tool.exe"}},
			},
			want: packslip.Artifact{
				Name: "tool.zip", OS: "windows", Arch: "i686", Format: "zip",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool.exe")},
			},
		},
		"universal macOS archive": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool.tar.gz", Goos: "darwin", Goarch: "all",
				Extra: map[string]any{artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"tool"}},
			},
			want: packslip.Artifact{
				Name: "tool.tar.gz", OS: "darwin", Format: "tar.gz",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool")},
			},
		},
		"bare binary": {
			in: artifact.Artifact{
				Type: artifact.UploadableBinary, Name: "tool_windows_arm64.exe", Goos: "windows", Goarch: "arm64",
				Extra: map[string]any{artifact.ExtraFormat: "binary", artifact.ExtraBinary: "tool.exe"},
			},
			want: packslip.Artifact{
				Name: "tool_windows_arm64.exe", OS: "windows", Arch: "aarch64", Format: "raw",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NamedBin("tool_windows_arm64.exe", "tool.exe")},
			},
		},
		"armv7 compressed binary": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool.gz", Goos: "freebsd", Goarch: "arm", Goarm: "7",
				Extra: map[string]any{artifact.ExtraFormat: "gz", artifact.ExtraBinaries: []string{"tool"}},
			},
			want: packslip.Artifact{
				Name: "tool.gz", OS: "freebsd", Arch: "armv7", Format: "gz",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NamedBin("tool", "tool")},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.want, describe(&tt.in, "https://example.com/x"))
		})
	}
}

func TestRun(t *testing.T) {
	t.Setenv("SIGSTORE_ID_TOKEN", fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"job_workflow_ref": "acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0",
	}))
	dist := t.TempDir()
	archive := filepath.Join(dist, "tool_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte("archive"), 0o644))

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:    dist,
		Release: config.Release{GitHub: config.Repo{Owner: "acme", Name: "tool"}},
		GitHubURLs: config.GitHubURLs{
			Download: "https://github.com",
		},
		Packslip: config.Packslip{Enabled: "true"},
	}, testctx.GitHubTokenType, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"), testctx.WithCommit("abc123"))
	ctx.Git.FullCommit = "0123456789abcdef0123456789abcdef01234567"
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive, Name: filepath.Base(archive), Path: archive,
		Goos: "linux", Goarch: "amd64",
		Extra: map[string]any{
			artifact.ExtraID:       "default",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"tool"},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))

	var signed packslip.Statement
	orig := signFunc
	t.Cleanup(func() { signFunc = orig })
	signFunc = func(_ *context.Context, statement packslip.Statement, token string) ([]byte, error) {
		require.NotEmpty(t, token)
		signed = statement
		return []byte(`{"bundle":true}`), nil
	}

	require.NoError(t, Pipe{}.Run(ctx))

	require.Equal(t, "github.com/acme/tool", signed.Predicate.Project)
	require.Equal(t, "1.0.0", signed.Predicate.Version)
	require.Equal(t, &packslip.Source{
		Repo:   "https://github.com/acme/tool",
		Commit: "0123456789abcdef0123456789abcdef01234567",
		Tag:    "v1.0.0",
	}, signed.Predicate.Source)
	require.Equal(t, "https://github.com/acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0", signed.Predicate.Identity.KeyID)
	require.Len(t, signed.Predicate.Artifacts, 1)
	require.Equal(t, "https://github.com/acme/tool/releases/download/v1.0.0/tool_1.0.0_linux_amd64.tar.gz", signed.Predicate.Artifacts[0].URL)
	require.Len(t, signed.Subject, 1)
	require.NoError(t, signed.Validate())

	files := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableFile)).List()
	require.Len(t, files, 1)
	require.Equal(t, "packslip.sigstore.json", files[0].Name)
	got, err := os.ReadFile(files[0].Path)
	require.NoError(t, err)
	require.JSONEq(t, `{"bundle":true}`, string(got))
}

func TestRunNoArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:     t.TempDir(),
		Packslip: config.Packslip{Enabled: "true", Project: "example.com/tool"},
	})
	require.True(t, pipe.IsSkip(Pipe{}.Run(ctx)))
}
