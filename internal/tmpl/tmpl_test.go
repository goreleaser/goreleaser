package tmpl

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestWithArtifact(t *testing.T) {
	t.Parallel()
	ctx := testctx.NewWithCfg(
		config.Project{ProjectName: "proj"},
		testctx.WithVersion("1.2.3"),
		testctx.WithGitInfo(context.GitInfo{
			PreviousTag: "v1.2.2",
			CurrentTag:  "v1.2.3",
			Branch:      "test-branch",
			Commit:      "commit",
			FullCommit:  "fullcommit",
			ShortCommit: "shortcommit",
			TagSubject:  "awesome release",
			TagContents: "awesome release\n\nanother line",
			TagBody:     "another line",
		}),
		testctx.WithEnv(map[string]string{
			"FOO":       "bar",
			"MULTILINE": "something with\nmultiple lines\nremove this\nto test things",
		}),
		testctx.WithSemver(1, 2, 3, ""),
		func(ctx *context.Context) {
			ctx.ModulePath = "github.com/goreleaser/goreleaser"
			ctx.ReleaseNotes = "test release notes"
			ctx.Date = time.Unix(1678327562, 0)
		},
	)
	for expect, tmpl := range map[string]string{
		"bar":                              "{{.Env.FOO}}",
		"Linux":                            "{{.Os}}",
		"amd64":                            "{{.Arch}}",
		"6":                                "{{.Arm}}",
		"softfloat":                        "{{.Mips}}",
		"v3":                               "{{.Amd64}}",
		"1.2.3":                            "{{.Version}}",
		"v1.2.3":                           "{{.Tag}}",
		"1-2-3":                            "{{.Major}}-{{.Minor}}-{{.Patch}}",
		"test-branch":                      "{{.Branch}}",
		"commit":                           "{{.Commit}}",
		"fullcommit":                       "{{.FullCommit}}",
		"shortcommit":                      "{{.ShortCommit}}",
		"binary":                           "{{.Binary}}",
		"proj":                             "{{.ProjectName}}",
		"github.com/goreleaser/goreleaser": "{{ .ModulePath }}",
		"v2.0.0":                           "{{.Tag | incmajor }}",
		"2.0.0":                            "{{.Version | incmajor }}",
		"v1.3.0":                           "{{.Tag | incminor }}",
		"1.3.0":                            "{{.Version | incminor }}",
		"v1.2.4":                           "{{.Tag | incpatch }}",
		"1.2.4":                            "{{.Version | incpatch }}",
		"test release notes":               "{{ .ReleaseNotes }}",
		"v1.2.2":                           "{{ .PreviousTag }}",
		"awesome release":                  "{{ .TagSubject }}",
		"awesome release\n\nanother line":  "{{ .TagContents }}",
		"another line":                     "{{ .TagBody }}",
		"runtime: " + runtime.GOOS:         "runtime: {{ .Runtime.Goos }}",
		"runtime: " + runtime.GOARCH:       "runtime: {{ .Runtime.Goarch }}",
		"artifact name: not-this-binary":   "artifact name: {{ .ArtifactName }}",
		"artifact ext: .exe":               "artifact ext: {{ .ArtifactExt }}",
		"artifact path: /tmp/foo.exe":      "artifact path: {{ .ArtifactPath }}",
		"artifact basename: foo.exe":       "artifact basename: {{ base .ArtifactPath }}",
		"artifact dir: /tmp":               "artifact dir: {{ dir .ArtifactPath }}",
		"2023":                             `{{ .Now.Format "2006" }}`,
		"2023-03-09T02:06:02Z":             `{{ .Date }}`,
		"1678327562":                       `{{ .Timestamp }}`,

		"remove this": "{{ filter .Env.MULTILINE \".*remove.*\" }}",
		"something with\nmultiple lines\nto test things": "{{ reverseFilter .Env.MULTILINE \".*remove.*\" }}",
	} {
		tmpl := tmpl
		expect := expect
		t.Run(expect, func(t *testing.T) {
			t.Parallel()
			result, err := New(ctx).WithArtifactReplacements(
				&artifact.Artifact{
					Name:    "not-this-binary",
					Path:    "/tmp/foo.exe",
					Goarch:  "amd64",
					Goos:    "linux",
					Goarm:   "6",
					Gomips:  "softfloat",
					Goamd64: "v3",
					Extra: map[string]interface{}{
						artifact.ExtraBinary: "binary",
						artifact.ExtraExt:    ".exe",
					},
				},
				map[string]string{"linux": "Linux"},
			).Apply(tmpl)
			require.NoError(t, err)
			require.Equal(t, expect, result)
		})
	}

	t.Run("artifact without binary name", func(t *testing.T) {
		t.Parallel()
		result, err := New(ctx).WithArtifactReplacements(
			&artifact.Artifact{
				Name:   "another-binary",
				Goarch: "amd64",
				Goos:   "linux",
				Goarm:  "6",
			}, map[string]string{},
		).Apply("{{ .Binary }}")
		require.NoError(t, err)
		require.Equal(t, ctx.Config.ProjectName, result)
	})

	t.Run("template using artifact Fields with no artifact", func(t *testing.T) {
		t.Parallel()
		result, err := New(ctx).Apply("{{ .Os }}")
		require.EqualError(t, err, `template: tmpl:1:3: executing "tmpl" at <.Os>: map has no entry for key "Os"`)
		require.Empty(t, result)
	})
}

func TestEnv(t *testing.T) {
	testCases := []struct {
		desc string
		in   string
		out  string
	}{
		{
			desc: "with env",
			in:   "{{ .Env.FOO }}",
			out:  "BAR",
		},
		{
			desc: "with env",
			in:   "{{ .Env.BAR }}",
			out:  "",
		},
	}
	ctx := testctx.New(
		testctx.WithEnv(map[string]string{"FOO": "BAR"}),
		testctx.WithCurrentTag("v1.2.3"),
	)
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, _ := New(ctx).Apply(tC.in)
			require.Equal(t, tC.out, out)
		})
	}
}

func TestWithEnv(t *testing.T) {
	ctx := testctx.New(
		testctx.WithEnv(map[string]string{"FOO": "BAR"}),
		testctx.WithCurrentTag("v1.2.3"),
	)
	tpl := New(ctx).WithEnvS([]string{
		"FOO=foo",
		"BAR=bar",
		"NOVAL=",
		"=NOKEY",
		"=",
		"NOTHING",
	})
	out, err := tpl.Apply("{{ .Env.FOO }}-{{ .Env.BAR }}")
	require.NoError(t, err)
	require.Equal(t, "foo-bar", out)

	out, err = tpl.Apply(`{{ range $idx, $key := .Env }}{{ $idx }},{{ end }}`)
	require.NoError(t, err)
	require.Equal(t, "BAR,FOO,NOVAL,", out)
}

func TestFuncMap(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "proj",
		Env: []string{
			"FOO=bar",
		},
	})
	wd, err := os.Getwd()
	require.NoError(t, err)

	ctx.Git.URL = "https://github.com/foo/bar.git"
	ctx.ReleaseURL = "https://github.com/foo/bar/releases/tag/v1.0.0"
	ctx.Git.CurrentTag = "v1.2.4"
	for _, tc := range []struct {
		Template string
		Name     string
		Expected string
	}{
		{
			Template: `{{ replace "v1.24" "v" "" }}`,
			Name:     "replace",
			Expected: "1.24",
		},
		{
			Template: `{{ if index .Env "SOME_ENV"  }}{{ .Env.SOME_ENV }}{{ else }}default value{{ end }}`,
			Name:     "default value",
			Expected: "default value",
		},
		{
			Template: `{{ if index .Env "FOO"  }}{{ .Env.FOO }}{{ else }}default value{{ end }}`,
			Name:     "default value set",
			Expected: "bar",
		},
		{
			Template: `{{ time "2006-01-02" }}`,
			Name:     "time YYYY-MM-DD",
		},
		{
			Template: `{{ time "01/02/2006" }}`,
			Name:     "time MM/DD/YYYY",
		},
		{
			Template: `{{ time "01/02/2006" }}`,
			Name:     "time MM/DD/YYYY",
		},
		{
			Template: `{{ tolower "TEST" }}`,
			Name:     "tolower",
			Expected: "test",
		},
		{
			Template: `{{ trimprefix "v1.2.4" "v" }}`,
			Name:     "trimprefix",
			Expected: "1.2.4",
		},
		{
			Template: `{{ trimsuffix .GitURL ".git" }}`,
			Name:     "trimsuffix",
			Expected: "https://github.com/foo/bar",
		},
		{
			Template: `{{ title "file" }}`,
			Name:     "title",
			Expected: "File",
		},
		{
			Template: `{{ .ReleaseURL }}`,
			Name:     "trimsuffix",
			Expected: "https://github.com/foo/bar/releases/tag/v1.0.0",
		},
		{
			Template: `{{ toupper "test" }}`,
			Name:     "toupper",
			Expected: "TEST",
		},
		{
			Template: `{{ trim " test " }}`,
			Name:     "trim",
			Expected: "test",
		},
		{
			Template: `{{ abs "file" }}`,
			Name:     "abs",
			Expected: filepath.Join(wd, "file"),
		},
	} {
		out, err := New(ctx).Apply(tc.Template)
		require.NoError(t, err)
		if tc.Expected != "" {
			require.Equal(t, tc.Expected, out)
		} else {
			require.NotEmpty(t, out)
		}
	}
}

func TestApplySingleEnvOnly(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Env: []string{
			"FOO=value",
			"BAR=another",
		},
	})

	testCases := []struct {
		name        string
		tpl         string
		expectedErr error
	}{
		{
			"empty tpl",
			"",
			nil,
		},
		{
			"whitespaces",
			" 	",
			nil,
		},
		{
			"plain-text only",
			"raw-token",
			ExpectedSingleEnvErr{},
		},
		{
			"variable with spaces",
			"{{ .Env.FOO }}",
			nil,
		},
		{
			"variable without spaces",
			"{{.Env.FOO}}",
			nil,
		},
		{
			"variable with outer spaces",
			"  {{ .Env.FOO }} ",
			nil,
		},
		{
			"unknown variable",
			"{{ .Env.UNKNOWN }}",
			template.ExecError{},
		},
		{
			"other interpolation",
			"{{ .ProjectName }}",
			ExpectedSingleEnvErr{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(ctx).ApplySingleEnvOnly(tc.tpl)
			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInvalidTemplate(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v1.1.1"))
	_, err := New(ctx).Apply("{{{.Foo}")
	require.EqualError(t, err, "template: tmpl:1: unexpected \"{\" in command")
}

func TestEnvNotFound(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v1.2.4"))
	result, err := New(ctx).Apply("{{.Env.FOO}}")
	require.Empty(t, result)
	require.EqualError(t, err, `template: tmpl:1:6: executing "tmpl" at <.Env.FOO>: map has no entry for key "FOO"`)
}

func TestWithExtraFields(t *testing.T) {
	ctx := testctx.New()
	out, _ := New(ctx).WithExtraFields(Fields{
		"MyCustomField": "foo",
	}).Apply("{{ .MyCustomField }}")
	require.Equal(t, "foo", out)
}

func TestBool(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		for _, v := range []string{
			" TruE   ",
			"true",
			"TRUE",
		} {
			t.Run(v, func(t *testing.T) {
				ctx := testctx.NewWithCfg(config.Project{
					Env: []string{"FOO=" + v},
				})
				b, err := New(ctx).Bool("{{.Env.FOO}}")
				require.NoError(t, err)
				require.True(t, b)
			})
		}
	})
	t.Run("false", func(t *testing.T) {
		for _, v := range []string{
			"    ",
			"",
			"false",
			"yada yada",
		} {
			t.Run(v, func(t *testing.T) {
				ctx := testctx.NewWithCfg(config.Project{
					Env: []string{"FOO=" + v},
				})
				b, err := New(ctx).Bool("{{.Env.FOO}}")
				require.NoError(t, err)
				require.False(t, b)
			})
		}
	})
}
