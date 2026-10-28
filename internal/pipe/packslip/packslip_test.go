package packslip

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		Packslips: []config.Packslip{
			{},
			{Project: "github.com/acme/tool/crates/cli"},
			{Project: "tool.example.com"},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, []config.Packslip{
		{Project: "github.com/acme/tool", NameTemplate: "packslip.sigstore.json"},
		{Project: "github.com/acme/tool/crates/cli", NameTemplate: "packslip.crates-cli.sigstore.json"},
		{Project: "tool.example.com", NameTemplate: "packslip.sigstore.json"},
	}, ctx.Config.Packslips)
}

func TestDefaultDuplicateProject(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release:   config.Release{GitHub: config.Repo{Owner: "acme", Name: "tool"}},
		Packslips: []config.Packslip{{}, {Project: "github.com/acme/tool"}},
	})
	require.ErrorContains(t, Pipe{}.Default(ctx), "multiple packslips with the project 'github.com/acme/tool'")
}

func TestSkip(t *testing.T) {
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Packslip))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("not configured", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("disabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Packslips: []config.Packslip{{Disable: "true"}}})
		require.False(t, Pipe{}.Skip(ctx))
		require.True(t, pipe.IsSkip(Pipe{}.Run(ctx)))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Packslips: []config.Packslip{{Project: "example.com/tool"}}}, testctx.Snapshot)
		require.ErrorIs(t, Pipe{}.Run(ctx), pipe.ErrSnapshotEnabled)
	})
	t.Run("skip publish", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Packslips: []config.Packslip{{Project: "example.com/tool"}}}, testctx.Skip(skips.Publish))
		require.ErrorIs(t, Pipe{}.Run(ctx), pipe.ErrSkipPublishEnabled)
	})
}

func fakeToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
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
					artifact.ExtraFormat:     "tar.gz",
					artifact.ExtraWrappedIn:  "tool_1.0.0",
					artifact.ExtraBinaries:   []string{"tool"},
					artifact.ExtraStaticLink: true,
				},
			},
			want: packslip.Artifact{
				Name: "tool_1.0.0_linux_amd64.tar.gz", OS: "linux", Arch: "x86_64", Format: "tar.gz",
				URL:      "https://example.com/x",
				Bin:      []packslip.Bin{packslip.NewBin("tool_1.0.0/tool")},
				Requires: &packslip.Requires{Libs: &[]string{}},
			},
		},
		"archive ignores non-executable members": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool_1.0.0_linux_amd64.tar.gz",
				Goos: "linux", Goarch: "amd64",
				Extra: map[string]any{
					artifact.ExtraFormat:       "tar.gz",
					artifact.ExtraWrappedIn:    "tool_1.0.0",
					artifact.ExtraBinaries:     []string{"tool", "tool.h"},
					artifact.ExtraPackslipBins: []string{"tool"},
					artifact.ExtraStaticLink:   true,
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
					artifact.ExtranDynLink:   true,
					artifact.ExtraLibcFamily: "gnu",
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
		"tar xz alias": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool_linux_amd64.txz", Goos: "linux", Goarch: "amd64",
				Extra: map[string]any{artifact.ExtraFormat: "txz", artifact.ExtraBinaries: []string{"tool"}, artifact.ExtraStaticLink: true},
			},
			want: packslip.Artifact{
				Name: "tool_linux_amd64.txz", OS: "linux", Arch: "x86_64", Format: "tar.xz",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool")},
				Requires: &packslip.Requires{Libs: &[]string{}},
			},
		},
		"tar zstd alias": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool_linux_amd64.tzst", Goos: "linux", Goarch: "amd64",
				Extra: map[string]any{artifact.ExtraFormat: "tzst", artifact.ExtraBinaries: []string{"tool"}, artifact.ExtraStaticLink: true},
			},
			want: packslip.Artifact{
				Name: "tool_linux_amd64.tzst", OS: "linux", Arch: "x86_64", Format: "tar.zst",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool")},
				Requires: &packslip.Requires{Libs: &[]string{}},
			},
		},
		"bare binary in a directory": {
			in: artifact.Artifact{
				Type: artifact.UploadableBinary, Name: "app_linux_amd64", Goos: "linux", Goarch: "amd64",
				Extra: map[string]any{artifact.ExtraFormat: "binary", artifact.ExtraBinary: "bin/app", artifact.ExtraStaticLink: true},
			},
			want: packslip.Artifact{
				Name: "app_linux_amd64", OS: "linux", Arch: "x86_64", Format: "raw",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NamedBin("app_linux_amd64", "app")},
				Requires: &packslip.Requires{Libs: &[]string{}},
			},
		},
		"compressed binary in a directory": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "app_linux_amd64.gz", Goos: "linux", Goarch: "amd64",
				Extra: map[string]any{artifact.ExtraFormat: "gz", artifact.ExtraBinaries: []string{"bin/app"}, artifact.ExtraStaticLink: true},
			},
			want: packslip.Artifact{
				Name: "app_linux_amd64.gz", OS: "linux", Arch: "x86_64", Format: "gz",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NamedBin("app_linux_amd64", "app")},
				Requires: &packslip.Requires{Libs: &[]string{}},
			},
		},
		// the archive writer turns backslashes into slashes in every entry.
		"backslash path in an archive": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool_1.0.0_windows_amd64.zip", Goos: "windows", Goarch: "amd64",
				Extra: map[string]any{
					artifact.ExtraFormat:    "zip",
					artifact.ExtraWrappedIn: "tool_1.0.0",
					artifact.ExtraBinaries:  []string{`bin\\app.exe`},
				},
			},
			want: packslip.Artifact{
				Name: "tool_1.0.0_windows_amd64.zip", OS: "windows", Arch: "x86_64", Format: "zip",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool_1.0.0/bin/app.exe")},
			},
		},
		"backslash path in a compressed binary": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "app_windows_amd64.gz", Goos: "windows", Goarch: "amd64",
				Extra: map[string]any{artifact.ExtraFormat: "gz", artifact.ExtraBinaries: []string{`bin\\app.exe`}},
			},
			want: packslip.Artifact{
				Name: "app_windows_amd64.gz", OS: "windows", Arch: "x86_64", Format: "gz",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NamedBin("app_windows_amd64", "app")},
			},
		},
		"backslash path in a raw binary": {
			in: artifact.Artifact{
				Type: artifact.UploadableBinary, Name: "app_windows_amd64.exe", Goos: "windows", Goarch: "amd64",
				Extra: map[string]any{artifact.ExtraFormat: "binary", artifact.ExtraBinary: `bin\\app.exe`},
			},
			want: packslip.Artifact{
				Name: "app_windows_amd64.exe", OS: "windows", Arch: "x86_64", Format: "raw",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NamedBin("app_windows_amd64.exe", "app")},
			},
		},
		"amd64 v3": {
			in: artifact.Artifact{
				Type: artifact.UploadableArchive, Name: "tool.zip", Goos: "windows", Goarch: "amd64", Goamd64: "v3",
				Extra: map[string]any{artifact.ExtraFormat: "zip", artifact.ExtraBinaries: []string{"tool.exe"}},
			},
			want: packslip.Artifact{
				Name: "tool.zip", OS: "windows", Arch: "x86_64", Variant: "v3", Format: "zip",
				URL: "https://example.com/x", Bin: []packslip.Bin{packslip.NewBin("tool.exe")},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := describe(&tt.in, "https://example.com/x")
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			// every case must also form a valid statement on its own.
			file := filepath.Join(t.TempDir(), tt.in.Name)
			require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
			identity, err := packslip.IdentityFromToken(fakeToken(t, map[string]any{
				"iss":              "https://token.actions.githubusercontent.com",
				"job_workflow_ref": "acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0",
			}))
			require.NoError(t, err)
			_, err = packslip.Build(packslip.Predicate{
				Project: "example.com/tool", Version: "1.0.0", PublishedAt: time.Now(), Identity: identity,
			}, []packslip.ArtifactInput{{Path: file, Artifact: got}})
			require.NoError(t, err)
		})
	}
}

func TestDescribeLinuxLibc(t *testing.T) {
	for name, tt := range map[string]struct {
		in   artifact.Artifact
		want string
		err  string
		// no libraries are claimed, as when linkage was never inspected.
		unchecked bool
	}{
		"static": {
			in: artifact.Artifact{Goos: "linux", Goarch: "amd64", Extra: map[string]any{artifact.ExtraStaticLink: true}},
		},
		"not inspected": {
			in:        artifact.Artifact{Goos: "linux", Goarch: "amd64"},
			unchecked: true,
		},
		"glibc": {
			in: artifact.Artifact{Goos: "linux", Goarch: "amd64", Extra: map[string]any{
				artifact.ExtranDynLink:   true,
				artifact.ExtraLibcFamily: "gnu",
			}},
			want: "gnu",
		},
		"musl": {
			in: artifact.Artifact{Goos: "linux", Goarch: "amd64", Extra: map[string]any{
				artifact.ExtranDynLink:   true,
				artifact.ExtraLibcFamily: "musl",
			}},
			want: "musl",
		},
		"unknown dynamic": {
			in:  artifact.Artifact{Goos: "linux", Goarch: "amd64", Name: "tool", Extra: map[string]any{artifact.ExtranDynLink: true}},
			err: "cannot determine libc",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := describe(&tt.in, "https://example.com/x")
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.Libc)
			if tt.unchecked {
				require.Nil(t, got.Requires, "libs stay unset when linkage is unknown")
			} else if tt.want == "" {
				require.NotNil(t, got.Requires)
				require.Equal(t, &[]string{}, got.Requires.Libs)
			}
		})
	}
}

// A Node SEA uses the stock glibc-linked Linux Node runtime. The Node builder
// records DynamicallyLinked and LibcFamily; the archive pipe copies them onto
// the uploadable artifact. Without that metadata, describe would treat a
// missing linkage flag as static (requires.libs: []) and a conforming
// installer could select the artifact on musl/Alpine.
func TestDescribeNodeSEAArchive(t *testing.T) {
	archive := &artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "tool_1.0.0_linux_amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Extra: map[string]any{
			artifact.ExtraFormat:     "tar.gz",
			artifact.ExtraBinaries:   []string{"tool"},
			artifact.ExtranDynLink:   true,
			artifact.ExtraLibcFamily: "gnu",
		},
	}
	got, err := describe(archive, "https://example.com/tool_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, err)
	require.Equal(t, "gnu", got.Libc)
	require.Nil(t, got.Requires)

	file := filepath.Join(t.TempDir(), archive.Name)
	require.NoError(t, os.WriteFile(file, []byte("sea"), 0o644))
	identity, err := packslip.IdentityFromToken(fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"job_workflow_ref": "acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0",
	}))
	require.NoError(t, err)
	_, err = packslip.Build(packslip.Predicate{
		Project: "example.com/tool", Version: "1.0.0", PublishedAt: time.Now(), Identity: identity,
	}, []packslip.ArtifactInput{{Path: file, Artifact: got}})
	require.NoError(t, err)
}

func cpuVariant(a *artifact.Artifact) string { return goTarget(a).Variant() }

func TestCPUVariant(t *testing.T) {
	for want, a := range map[string]artifact.Artifact{
		"":          {Goarch: "amd64", Goamd64: "v1"},
		"v2":        {Goarch: "amd64", Goamd64: "v2"},
		"softfloat": {Goarch: "386", Go386: "softfloat"},
		"v8.2-lse":  {Goarch: "arm64", Goarm64: "v8.2,lse"},
		"power9":    {Goarch: "ppc64le", Goppc64: "power9"},
		"rva23u64":  {Goarch: "riscv64", Goriscv64: "rva23u64"},
	} {
		require.Equal(t, want, cpuVariant(&a), a)
	}
	for _, a := range []artifact.Artifact{
		{Goarch: "arm", Goarm: "6"},
		{Goarch: "arm", Goarm: "7", Extra: map[string]any{artifact.ExtraABI: "hardfloat"}},
		{Goarch: "arm", Goarm: "5", Extra: map[string]any{artifact.ExtraABI: "softfloat"}},
	} {
		require.Empty(t, cpuVariant(&a), a)
	}
	require.Equal(t, "softfloat", cpuVariant(&artifact.Artifact{
		Goarch: "arm", Goarm: "7", Extra: map[string]any{artifact.ExtraABI: "softfloat"},
	}))
	require.Equal(t, "softfloat", cpuVariant(&artifact.Artifact{Goarch: "mipsle", Gomips: "softfloat"}))

	// Rust and Zig keep the target's ABI instead of Go's float option.
	for abi, want := range map[string]string{
		"gnueabi":    "softfloat",
		"musleabi":   "softfloat",
		"gnueabihf":  "",
		"musleabihf": "",
	} {
		require.Equal(t, want, cpuVariant(&artifact.Artifact{
			Goarch: "arm", Goarm: "6", Extra: map[string]any{artifact.ExtraABI: abi},
		}), abi)
	}
	require.Empty(t, cpuVariant(&artifact.Artifact{
		Goarch: "arm", Goarm: "5", Extra: map[string]any{artifact.ExtraABI: "gnueabi"},
	}))
}

// arm-unknown-linux-gnueabi and arm-unknown-linux-gnueabihf are both armv6 with
// glibc, so only the variant tells the archives apart. Without it the
// selectors are ambiguous and the statement is rejected.
func TestRunRustARMFloatABIs(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:       dist,
		Release:    config.Release{GitHub: config.Repo{Owner: "acme", Name: "tools"}},
		GitHubURLs: config.GitHubURLs{Download: "https://github.com"},
		Packslips:  []config.Packslip{{}},
	}, testctx.GitHubTokenType, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"), testctx.WithCommit("abc123"))
	ctx.Git.FullCommit = "0123456789abcdef0123456789abcdef01234567"
	ctx.Git.URL = "git@github.com:acme/tools.git"
	for _, abi := range []string{"gnueabi", "gnueabihf"} {
		name := "tool_1.0.0_linux_arm_" + abi + ".tar.gz"
		path := filepath.Join(dist, name)
		require.NoError(t, os.WriteFile(path, []byte(abi), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableArchive, Name: name, Path: path,
			Goos: "linux", Goarch: "arm", Goarm: "6",
			Extra: map[string]any{
				artifact.ExtraID:         "default",
				artifact.ExtraFormat:     "tar.gz",
				artifact.ExtraBinaries:   []string{"tool"},
				artifact.ExtraABI:        abi,
				artifact.ExtranDynLink:   true,
				artifact.ExtraLibcFamily: "gnu",
			},
		})
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	statement := signed["github.com/acme/tools"]
	require.NoError(t, statement.Validate())
	variants := map[string]string{}
	for _, a := range statement.Predicate.Artifacts {
		require.Equal(t, "armv6", a.Arch)
		require.Equal(t, "gnu", a.Libc)
		variants[a.Name] = a.Variant
	}
	require.Equal(t, map[string]string{
		"tool_1.0.0_linux_arm_gnueabi.tar.gz":   "softfloat",
		"tool_1.0.0_linux_arm_gnueabihf.tar.gz": "",
	}, variants)
}

func TestSourceRepo(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:acme/tool.git":                      "https://github.com/acme/tool",
		"https://github.com/acme/tool.git":                  "https://github.com/acme/tool",
		"ssh://git@github.com/acme/tool":                    "https://github.com/acme/tool",
		"ssh://git@github.com:22/acme/tool":                 "https://github.com/acme/tool",
		"https://gitlab-ci-token:x@gitlab.com/a/b/tool.git": "https://gitlab.com/a/b/tool",
		"http://user:pass@forge.example:8443/acme/tool.git": "http://forge.example:8443/acme/tool",
		"https://forge.example:9443/acme/tool.git":          "https://forge.example:9443/acme/tool",
		"":            "",
		"/local/path": "",
	} {
		require.Equal(t, want, sourceRepo(in), in)
	}
}

func TestProjectSigner(t *testing.T) {
	token := fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"repository":       "acme/releases",
		"job_workflow_ref": "acme/build/.github/workflows/release.yml@refs/tags/v1.0.0",
	})
	// The reusable workflow is hosted in acme/build, but Fulcio certifies the
	// caller acme/releases from the repository claim.
	require.NoError(t, packslip.CheckProjectSigner("github.com/acme/releases/tool", token))
	require.ErrorContains(t, packslip.CheckProjectSigner("github.com/acme/build", token), "does not belong")

	missingRepo := fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"job_workflow_ref": "acme/build/.github/workflows/release.yml@refs/tags/v1.0.0",
	})
	require.ErrorContains(t, packslip.CheckProjectSigner("github.com/acme/releases", missingRepo), "no repository claim")
}

func TestRun(t *testing.T) {
	t.Setenv("SIGSTORE_ID_TOKEN", fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"repository":       "acme/releases",
		"job_workflow_ref": "acme/build/.github/workflows/release.yml@refs/tags/v1.0.0",
	}))
	dist := t.TempDir()
	archive := filepath.Join(dist, "tool_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(archive, []byte("archive"), 0o644))

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:    dist,
		Release: config.Release{GitHub: config.Repo{Owner: "acme", Name: "releases"}},
		GitHubURLs: config.GitHubURLs{
			Download: "https://github.com",
		},
		Packslips: []config.Packslip{{}},
	}, testctx.GitHubTokenType, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"), testctx.WithCommit("abc123"))
	ctx.Git.FullCommit = "0123456789abcdef0123456789abcdef01234567"
	ctx.Git.URL = "git@github.com:acme/source.git"
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive, Name: filepath.Base(archive), Path: archive,
		Goos: "linux", Goarch: "amd64", Goamd64: "v1",
		Extra: map[string]any{
			artifact.ExtraID:       "default",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"tool"},
		},
	})
	archiveV3 := filepath.Join(dist, "tool_1.0.0_linux_amd64_v3.tar.gz")
	require.NoError(t, os.WriteFile(archiveV3, []byte("archive v3"), 0o644))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive, Name: filepath.Base(archiveV3), Path: archiveV3,
		Goos: "linux", Goarch: "amd64", Goamd64: "v3",
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

	require.Equal(t, "github.com/acme/releases", signed.Predicate.Project)
	require.Equal(t, "1.0.0", signed.Predicate.Version)
	require.Equal(t, &packslip.Source{
		Repo:   "https://github.com/acme/source",
		Commit: "0123456789abcdef0123456789abcdef01234567",
		Tag:    "v1.0.0",
	}, signed.Predicate.Source)
	require.Equal(t, "https://github.com/acme/build/.github/workflows/release.yml@refs/tags/v1.0.0", signed.Predicate.Identity.KeyID)
	require.Len(t, signed.Predicate.Artifacts, 2)
	byName := map[string]packslip.Artifact{}
	for _, a := range signed.Predicate.Artifacts {
		byName[a.Name] = a
	}
	require.Equal(t, "https://github.com/acme/releases/releases/download/v1.0.0/tool_1.0.0_linux_amd64.tar.gz", byName["tool_1.0.0_linux_amd64.tar.gz"].URL)
	require.Empty(t, byName["tool_1.0.0_linux_amd64.tar.gz"].Variant)
	require.Equal(t, "v3", byName["tool_1.0.0_linux_amd64_v3.tar.gz"].Variant)
	require.Len(t, signed.Subject, 2)
	require.NoError(t, signed.Validate())

	files := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableFile)).List()
	require.Len(t, files, 1)
	require.Equal(t, "packslip.sigstore.json", files[0].Name)
	require.True(t, artifact.ExtraOr(*files[0], artifact.ExtraSkipChecksum, false))
	got, err := os.ReadFile(files[0].Path)
	require.NoError(t, err)
	require.JSONEq(t, `{"bundle":true}`, string(got))
}

func TestRunNoArtifacts(t *testing.T) {
	setToken(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:      t.TempDir(),
		Packslips: []config.Packslip{{Project: "example.com/tool", NameTemplate: "packslip.sigstore.json"}},
	})
	require.True(t, pipe.IsSkip(Pipe{}.Run(ctx)))
}

func TestRunSkipsLibraryArtifacts(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	dist := t.TempDir()
	for _, name := range []string{"libtool.tar.gz", "libtool.so"} {
		file := filepath.Join(dist, name)
		require.NoError(t, os.WriteFile(file, []byte(name), 0o644))
	}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:       dist,
		Release:    config.Release{GitHub: config.Repo{Owner: "acme", Name: "tool"}},
		GitHubURLs: config.GitHubURLs{Download: "https://github.com"},
		Packslips:  []config.Packslip{{}},
	}, testctx.GitHubTokenType, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
	for _, tt := range []struct {
		name string
		typ  artifact.Type
	}{
		{name: "libtool.tar.gz", typ: artifact.UploadableArchive},
		{name: "libtool.so", typ: artifact.UploadableBinary},
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: tt.typ, Name: tt.name, Path: filepath.Join(dist, tt.name),
			Goos: "linux", Goarch: "amd64",
			Extra: map[string]any{
				artifact.ExtraFormat:       "tar.gz",
				artifact.ExtraSkipPackslip: true,
			},
		})
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(Pipe{}.Run(ctx)))
	require.Empty(t, signed)
	require.Empty(t, ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableFile)).List())
}

func TestRunNoProject(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{Packslips: []config.Packslip{{}}})
	require.NoError(t, Pipe{}.Default(ctx))
	require.ErrorContains(t, Pipe{}.Run(ctx), "packslips.project must be set")
}

func setToken(t *testing.T) {
	t.Helper()
	t.Setenv("SIGSTORE_ID_TOKEN", fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"repository":       "acme/tools",
		"job_workflow_ref": "acme/tools/.github/workflows/release.yml@refs/tags/v1.0.0",
	}))
}

// stubSign replaces signFunc for the test and returns the statements it was
// given, keyed by project.
func stubSign(t *testing.T) map[string]packslip.Statement {
	t.Helper()
	signed := map[string]packslip.Statement{}
	orig := signFunc
	t.Cleanup(func() { signFunc = orig })
	signFunc = func(_ *context.Context, statement packslip.Statement, _ string) ([]byte, error) {
		signed[statement.Predicate.Project] = statement
		return []byte(`{}`), nil
	}
	return signed
}

func monorepoCtx(t *testing.T, packslips []config.Packslip) *context.Context {
	t.Helper()
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:       dist,
		Release:    config.Release{GitHub: config.Repo{Owner: "acme", Name: "tools"}},
		GitHubURLs: config.GitHubURLs{Download: "https://github.com"},
		Packslips:  packslips,
	}, testctx.GitHubTokenType, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
	for _, tool := range []string{"foo", "bar"} {
		file := filepath.Join(dist, tool+"_linux_amd64.tar.gz")
		require.NoError(t, os.WriteFile(file, []byte(tool), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableArchive, Name: filepath.Base(file), Path: file,
			Goos: "linux", Goarch: "amd64",
			Extra: map[string]any{
				artifact.ExtraID:       tool,
				artifact.ExtraFormat:   "tar.gz",
				artifact.ExtraBinaries: []string{tool},
			},
		})
	}
	require.NoError(t, Pipe{}.Default(ctx))
	return ctx
}

func TestRunMonorepo(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	ctx := monorepoCtx(t, []config.Packslip{
		{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo"},
		{IDs: []string{"bar"}, Project: "github.com/acme/tools/cmd/bar"},
	})
	require.NoError(t, Pipe{}.Run(ctx))

	require.Len(t, signed, 2)
	for project, want := range map[string]string{
		"github.com/acme/tools/foo":     "foo_linux_amd64.tar.gz",
		"github.com/acme/tools/cmd/bar": "bar_linux_amd64.tar.gz",
	} {
		require.Len(t, signed[project].Predicate.Artifacts, 1, project)
		require.Equal(t, want, signed[project].Predicate.Artifacts[0].Name)
	}
	var names []string
	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableFile)).List() {
		names = append(names, a.Name)
		require.FileExists(t, a.Path)
	}
	require.ElementsMatch(t, []string{"packslip.foo.sigstore.json", "packslip.cmd-bar.sigstore.json"}, names)
}

func TestRunDuplicateName(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	ctx := monorepoCtx(t, []config.Packslip{
		{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo", NameTemplate: "packslip.sigstore.json"},
		{IDs: []string{"bar"}, Project: "github.com/acme/tools/bar", NameTemplate: "packslip.sigstore.json"},
	})
	require.ErrorContains(t, Pipe{}.Run(ctx), "multiple packslips named 'packslip.sigstore.json'")
	require.Empty(t, signed, "nothing is signed when an entry is invalid")
}

func TestRunReleaseIDs(t *testing.T) {
	t.Setenv("SIGSTORE_ID_TOKEN", fakeToken(t, map[string]any{
		"iss":              "https://token.actions.githubusercontent.com",
		"repository":       "acme/tool",
		"job_workflow_ref": "acme/tool/.github/workflows/release.yml@refs/tags/v1.0.0",
	}))
	dist := t.TempDir()
	newCtx := func(t *testing.T, cfg config.Packslip) *context.Context {
		t.Helper()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:       dist,
			Release:    config.Release{GitHub: config.Repo{Owner: "acme", Name: "tool"}, IDs: []string{"linux"}},
			GitHubURLs: config.GitHubURLs{Download: "https://github.com"},
			Packslips:  []config.Packslip{cfg},
		}, testctx.GitHubTokenType, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
		for _, goos := range []string{"linux", "windows"} {
			file := filepath.Join(dist, "tool_"+goos+".zip")
			require.NoError(t, os.WriteFile(file, []byte(goos), 0o644))
			ctx.Artifacts.Add(&artifact.Artifact{
				Type: artifact.UploadableArchive, Name: filepath.Base(file), Path: file,
				Goos: goos, Goarch: "amd64",
				Extra: map[string]any{
					artifact.ExtraID:       goos,
					artifact.ExtraFormat:   "zip",
					artifact.ExtraBinaries: []string{"tool"},
				},
			})
		}
		require.NoError(t, Pipe{}.Default(ctx))
		return ctx
	}
	var signed packslip.Statement
	orig := signFunc
	t.Cleanup(func() { signFunc = orig })
	signFunc = func(_ *context.Context, statement packslip.Statement, _ string) ([]byte, error) {
		signed = statement
		return []byte(`{}`), nil
	}

	t.Run("release urls", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(newCtx(t, config.Packslip{})))
		require.Len(t, signed.Predicate.Artifacts, 1)
		require.Equal(t, "tool_linux.zip", signed.Predicate.Artifacts[0].Name)
	})
	t.Run("custom urls", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(newCtx(t, config.Packslip{
			URLTemplate: "https://dl.example.com/{{ .ArtifactName }}",
		})))
		require.Len(t, signed.Predicate.Artifacts, 2)
	})
}

func TestRunNameOverwritesArtifact(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	ctx := monorepoCtx(t, []config.Packslip{
		{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo", NameTemplate: "foo_linux_amd64.tar.gz"},
	})
	archive := filepath.Join(ctx.Config.Dist, "foo_linux_amd64.tar.gz")

	err := Pipe{}.Run(ctx)
	require.ErrorContains(t, err, "would overwrite")
	require.ErrorContains(t, err, "foo_linux_amd64.tar.gz")
	require.Empty(t, signed, "nothing is signed when the name is taken")
	got, err := os.ReadFile(archive)
	require.NoError(t, err)
	require.Equal(t, "foo", string(got), "the archive is left alone")
}

func TestRunNameNotAFileName(t *testing.T) {
	setToken(t)
	for _, name := range []string{
		"../foo_linux_amd64.tar.gz",
		"sub/packslip.sigstore.json",
		`sub\packslip.sigstore.json`,
		"..",
	} {
		t.Run(name, func(t *testing.T) {
			signed := stubSign(t)
			ctx := monorepoCtx(t, []config.Packslip{
				{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo", NameTemplate: name},
			})
			require.ErrorContains(t, Pipe{}.Run(ctx), "must be a file name")
			require.Empty(t, signed)
		})
	}
}

func TestRunNameOverwritesExtraFile(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	ctx := monorepoCtx(t, []config.Packslip{
		{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo", NameTemplate: "notes.json"},
	})
	extra := filepath.Join(t.TempDir(), "notes.json")
	require.NoError(t, os.WriteFile(extra, []byte("notes"), 0o644))
	ctx.Config.Release.ExtraFiles = []config.ExtraFile{{Glob: extra}}

	require.ErrorContains(t, Pipe{}.Run(ctx), "would overwrite the release extra file 'notes.json'")
	require.Empty(t, signed)
}

func TestRunIDsMatchNothing(t *testing.T) {
	setToken(t)
	signed := stubSign(t)
	ctx := monorepoCtx(t, []config.Packslip{
		{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo"},
		{IDs: []string{"baz"}, Project: "github.com/acme/tools/baz"},
	})
	err := Pipe{}.Run(ctx)
	require.ErrorContains(t, err, "packslips.ids [baz] match no archives or binaries")
	require.ErrorContains(t, err, "github.com/acme/tools/baz")
	require.Empty(t, signed, "nothing is signed when an entry is invalid")
}

func TestRunReleaseDisable(t *testing.T) {
	setToken(t)
	t.Run("release urls", func(t *testing.T) {
		signed := stubSign(t)
		ctx := monorepoCtx(t, []config.Packslip{
			{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo"},
		})
		ctx.Config.Release.Disable = "true"
		require.ErrorContains(t, Pipe{}.Run(ctx), "release.disable")
		require.Empty(t, signed)
	})
	t.Run("custom urls", func(t *testing.T) {
		signed := stubSign(t)
		ctx := monorepoCtx(t, []config.Packslip{
			{
				IDs: []string{"foo"}, Project: "github.com/acme/tools/foo",
				URLTemplate: "https://dl.example.com/{{ .ArtifactName }}",
			},
		})
		ctx.Config.Release.Disable = "true"
		require.NoError(t, Pipe{}.Run(ctx))
		require.Len(t, signed, 1)
	})
}

func TestRunSkipUpload(t *testing.T) {
	setToken(t)
	t.Run("release urls", func(t *testing.T) {
		signed := stubSign(t)
		ctx := monorepoCtx(t, []config.Packslip{
			{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo"},
		})
		ctx.Config.Release.SkipUpload = "true"
		require.ErrorContains(t, Pipe{}.Run(ctx), "release.skip_upload")
		require.Empty(t, signed, "nothing is signed for assets that are never uploaded")
	})
	t.Run("templated", func(t *testing.T) {
		signed := stubSign(t)
		ctx := monorepoCtx(t, []config.Packslip{
			{IDs: []string{"foo"}, Project: "github.com/acme/tools/foo"},
		})
		ctx.Config.Release.SkipUpload = "{{ .IsSnapshot }}"
		require.NoError(t, Pipe{}.Run(ctx))
		require.Len(t, signed, 1)
	})
	t.Run("custom urls", func(t *testing.T) {
		signed := stubSign(t)
		ctx := monorepoCtx(t, []config.Packslip{
			{
				IDs: []string{"foo"}, Project: "github.com/acme/tools/foo",
				URLTemplate: "https://dl.example.com/{{ .ArtifactName }}",
			},
		})
		ctx.Config.Release.SkipUpload = "true"
		require.NoError(t, Pipe{}.Run(ctx))
		require.Len(t, signed, 1)
		require.Equal(t,
			"https://dl.example.com/foo_linux_amd64.tar.gz",
			signed["github.com/acme/tools/foo"].Predicate.Artifacts[0].URL,
		)
	})
}
