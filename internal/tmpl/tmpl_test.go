package tmpl

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestWithArtifact(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	ctx.Env = map[string]string{
		"FOO": "bar",
	}
	ctx.Version = "1.2.3"
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Semver = context.Semver{
		Major: 1,
		Minor: 2,
		Patch: 3,
	}
	ctx.Git.Commit = "commit"
	ctx.Git.FullCommit = "fullcommit"
	ctx.Git.ShortCommit = "shortcommit"
	for expect, tmpl := range map[string]string{
		"bar":         "{{.Env.FOO}}",
		"Linux":       "{{.Os}}",
		"amd64":       "{{.Arch}}",
		"6":           "{{.Arm}}",
		"softfloat":   "{{.Mips}}",
		"1.2.3":       "{{.Version}}",
		"v1.2.3":      "{{.Tag}}",
		"1-2-3":       "{{.Major}}-{{.Minor}}-{{.Patch}}",
		"commit":      "{{.Commit}}",
		"fullcommit":  "{{.FullCommit}}",
		"shortcommit": "{{.ShortCommit}}",
		"binary":      "{{.Binary}}",
		"proj":        "{{.ProjectName}}",
		"":            "{{.ArtifactUploadHash}}",
	} {
		tmpl := tmpl
		expect := expect
		t.Run(expect, func(tt *testing.T) {
			tt.Parallel()
			result, err := New(ctx).WithArtifact(
				&artifact.Artifact{
					Name:   "not-this-binary",
					Goarch: "amd64",
					Goos:   "linux",
					Goarm:  "6",
					Gomips: "softfloat",
					Extra: map[string]interface{}{
						"Binary": "binary",
					},
				},
				map[string]string{"linux": "Linux"},
			).Apply(tmpl)
			require.NoError(tt, err)
			require.Equal(tt, expect, result)
		})
	}

	t.Run("artifact with gitlab ArtifactUploadHash", func(tt *testing.T) {
		tt.Parallel()
		uploadHash := "820ead5d9d2266c728dce6d4d55b6460"
		result, err := New(ctx).WithArtifact(
			&artifact.Artifact{
				Name:   "another-binary",
				Goarch: "amd64",
				Goos:   "linux",
				Goarm:  "6",
				Extra: map[string]interface{}{
					"ArtifactUploadHash": uploadHash,
				},
			}, map[string]string{},
		).Apply("{{ .ArtifactUploadHash }}")
		require.NoError(tt, err)
		require.Equal(tt, uploadHash, result)
	})

	t.Run("artifact without binary name", func(tt *testing.T) {
		tt.Parallel()
		result, err := New(ctx).WithArtifact(
			&artifact.Artifact{
				Name:   "another-binary",
				Goarch: "amd64",
				Goos:   "linux",
				Goarm:  "6",
			}, map[string]string{},
		).Apply("{{ .Binary }}")
		require.NoError(tt, err)
		require.Equal(tt, ctx.Config.ProjectName, result)
	})

	t.Run("template using artifact Fields with no artifact", func(tt *testing.T) {
		tt.Parallel()
		result, err := New(ctx).Apply("{{ .Os }}")
		require.EqualError(tt, err, `template: tmpl:1:3: executing "tmpl" at <.Os>: map has no entry for key "Os"`)
		require.Empty(tt, result)
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
	var ctx = context.New(config.Project{})
	ctx.Env = map[string]string{
		"FOO": "BAR",
	}
	ctx.Git.CurrentTag = "v1.2.3"
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, _ := New(ctx).Apply(tC.in)
			require.Equal(t, tC.out, out)
		})
	}
}

func TestWithEnv(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Env = map[string]string{
		"FOO": "BAR",
	}
	ctx.Git.CurrentTag = "v1.2.3"
	out, err := New(ctx).WithEnvS([]string{
		"FOO=foo",
		"BAR=bar",
	}).Apply("{{ .Env.FOO }}-{{ .Env.BAR }}")
	require.NoError(t, err)
	require.Equal(t, "foo-bar", out)
}

func TestFuncMap(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	wd, err := os.Getwd()
	require.NoError(t, err)

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
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{})
	ctx.Git.CurrentTag = "v1.1.1"
	_, err := New(ctx).Apply("{{{.Foo}")
	require.EqualError(t, err, "template: tmpl:1: unexpected \"{\" in command")
}

func TestEnvNotFound(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v1.2.4"
	result, err := New(ctx).Apply("{{.Env.FOO}}")
	require.Empty(t, result)
	require.EqualError(t, err, `template: tmpl:1:6: executing "tmpl" at <.Env.FOO>: map has no entry for key "FOO"`)
}

func TestWithExtraFields(t *testing.T) {
	var ctx = context.New(config.Project{})
	out, _ := New(ctx).WithExtraFields(Fields{
		"MyCustomField": "foo",
	}).Apply("{{ .MyCustomField }}")
	require.Equal(t, "foo", out)

}
