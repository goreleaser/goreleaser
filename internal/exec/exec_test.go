package exec

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "blah",
		Env: []string{
			"TEST_A_SECRET=x",
			"TEST_A_USERNAME=u2",
		},
	}, testctx.WithVersion("2.1.0"))

	folder := t.TempDir()
	for _, a := range []struct {
		id  string
		ext string
		typ artifact.Type
	}{
		{"debpkg", "deb", artifact.LinuxPackage},
		{"binary", "bin", artifact.Binary},
		{"archive", "tar", artifact.UploadableArchive},
		{"ubinary", "ubi", artifact.UploadableBinary},
		{"checksum", "sum", artifact.Checksum},
		{"metadata", "json", artifact.Metadata},
		{"signature", "sig", artifact.Signature},
		{"signature", "pem", artifact.Certificate},
	} {
		file := filepath.ToSlash(filepath.Join(folder, "a."+a.ext))
		require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "a." + a.ext,
			Goos:   "linux",
			Goarch: "amd64",
			Path:   file,
			Type:   a.typ,
			Extra: map[string]any{
				artifact.ExtraID: a.id,
			},
		})
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foobar-amd64",
		Goos:   "linux",
		Goarch: "amd64",
		Path:   "foobar-amd64",
		Type:   artifact.DockerImage,
		Extra: map[string]any{
			artifact.ExtraID: "img",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "foobar",
		Path: "foobar",
		Type: artifact.DockerManifest,
		Extra: map[string]any{
			artifact.ExtraID: "mnf",
		},
	})

	testCases := []struct {
		name        string
		publishers  func(outDir string) []config.Publisher
		check       func(tb testing.TB, outDir string)
		expectErr   error
		expectErrAs any
	}{
		{
			name: "filter by IDs",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					IDs:  []string{"archive"},
					Cmd:  testlib.Touch(outDir + "/{{ .ArtifactName }}"),
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.Equal(tb, []string{"a.tar"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "no filter",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name:    "test",
					Cmd:     testlib.Touch(outDir + "/{{ .ArtifactName }}"),
					Disable: "false",
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"a.deb", "a.tar", "a.ubi", "foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "disabled",
			publishers: func(string) []config.Publisher {
				return []config.Publisher{{
					Name:    "test",
					Cmd:     testlib.Echo("nope"),
					Disable: "true",
				}}
			},
			expectErr: pipe.ErrSkip{},
		},
		{
			name: "disabled invalid tmpl",
			publishers: func(string) []config.Publisher {
				return []config.Publisher{{
					Name:    "test",
					Cmd:     testlib.Echo("nope"),
					Disable: "{{ .NOPE }}",
				}}
			},
			expectErrAs: &tmpl.Error{},
		},
		{
			name: "include checksum",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name:     "test",
					Checksum: true,
					Cmd:      testlib.Touch(outDir + "/{{ .ArtifactName }}"),
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"a.deb", "a.sum", "a.tar", "a.ubi", "foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "include metadata",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					Meta: true,
					Cmd:  testlib.Touch(outDir + "/{{ .ArtifactName }}"),
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"a.deb", "a.json", "a.tar", "a.ubi", "foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "include signatures",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name:      "test",
					Signature: true,
					Cmd:       testlib.Touch(outDir + "/{{ .ArtifactName }}"),
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"a.deb", "a.pem", "a.sig", "a.tar", "a.ubi", "foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "docker",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					IDs:  []string{"img", "mnf"},
					Cmd:  testlib.Touch(outDir + "/{{ .ArtifactName }}"),
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "extra files",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					Cmd:  testlib.Touch(outDir + "/{{ .ArtifactName }}"),
					ExtraFiles: []config.ExtraFile{
						{Glob: path.Join("testdata", "*.txt")},
					},
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"a.deb", "a.tar", "a.txt", "a.ubi", "foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "extra files with rename",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					Cmd:  testlib.Touch(outDir + "/{{ .ArtifactName }}"),
					ExtraFiles: []config.ExtraFile{
						{
							Glob:         path.Join("testdata", "*.txt"),
							NameTemplate: "b.txt",
						},
					},
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.ElementsMatch(tb, []string{"a.deb", "a.tar", "a.ubi", "b.txt", "foobar", "foobar-amd64"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "try dir templating",
			publishers: func(outDir string) []config.Publisher {
				return []config.Publisher{{
					Name:      "test",
					Signature: true,
					IDs:       []string{"debpkg"},
					Dir:       "{{ dir .ArtifactPath }}",
					Cmd:       testlib.Touch(outDir + "/{{ .ArtifactName }}"),
				}}
			},
			check: func(tb testing.TB, outDir string) {
				require.Equal(tb, []string{"a.deb"}, dirFiles(tb, outDir))
			},
		},
		{
			name: "check env templating",
			publishers: func(string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					IDs:  []string{"debpkg"},
					Cmd:  assertEnv(map[string]string{"PROJECT": "blah", "ARTIFACT": "a.deb", "SECRET": "x"}),
					Env: []string{
						"PROJECT={{.ProjectName}}",
						"ARTIFACT={{.ArtifactName}}",
						"SECRET={{.Env.TEST_A_SECRET}}",
					},
				}}
			},
		},
		{
			name: "override path",
			publishers: func(string) []config.Publisher {
				return []config.Publisher{{
					Name: "test",
					IDs:  []string{"debpkg"},
					Cmd:  assertEnv(map[string]string{"PATH": "/something-else"}),
					Env: []string{
						"PATH=/something-else",
					},
				}}
			},
		},
		{
			name: "command error",
			publishers: func(string) []config.Publisher {
				return []config.Publisher{
					{
						Disable: "true",
					},
					{
						Name: "test",
						IDs:  []string{"debpkg"},
						Cmd:  testlib.ShC("exit 1"),
					},
				}
			},
			expectErr: fmt.Errorf(`exit status 1`),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			outDir := t.TempDir()
			err := Execute(ctx, tc.publishers(outDir))
			if tc.expectErr != nil {
				require.Error(t, err)
				require.True(t, strings.HasPrefix(err.Error(), tc.expectErr.Error()), err.Error())
				return
			}
			if tc.expectErrAs != nil {
				require.ErrorAs(t, err, tc.expectErrAs)
				return
			}
			require.NoError(t, err)
			if tc.check != nil {
				tc.check(t, outDir)
			}
		})
	}
}

func assertEnv(kvs map[string]string) string {
	var (
		fmt_ string
		join string
		wrap string
	)
	if testlib.IsWindows() {
		fmt_ = `if not "%%%s%%"=="%s" exit /b 1`
		join = " & "
		wrap = "cmd.exe /c '%s'"
	} else {
		fmt_ = `test "$%s" = "%s"`
		join = " && "
		wrap = "sh -c '%s'"
	}
	var parts []string
	for k, v := range kvs {
		parts = append(parts, fmt.Sprintf(fmt_, k, v))
	}
	return fmt.Sprintf(wrap, strings.Join(parts, join))
}

func dirFiles(tb testing.TB, dir string) []string {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(tb, err)
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	sort.Strings(names)
	return names
}
