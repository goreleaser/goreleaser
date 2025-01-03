package tmpl

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestWithArtifact(t *testing.T) {
	t.Parallel()
	ctx := testctx.NewWithCfg(
		config.Project{
			ProjectName: "proj",
			Release: config.Release{
				Draft: true,
			},
		},
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
			Dirty:       true,
		}),
		testctx.WithEnv(map[string]string{
			"FOO":          "bar",
			"MULTILINE":    "something with\nmultiple lines\nremove this\nto test things",
			"WITH_SLASHES": "foo/bar",
		}),
		testctx.WithSemver(1, 2, 3, ""),
		testctx.Snapshot,
		func(ctx *context.Context) {
			ctx.ModulePath = "github.com/goreleaser/goreleaser/v2"
			ctx.ReleaseNotes = "test release notes"
			ctx.Date = time.Unix(1678327562, 0)
			ctx.SingleTarget = true
		},
	)
	for expect, tmpl := range map[string]string{
		"bar":                                 "{{.Env.FOO}}",
		"linux":                               "{{.Os}}",
		"amd64":                               "{{.Arch}}",
		"6":                                   "{{.Arm}}",
		"softfloat":                           "{{.Mips}}",
		"v3":                                  "{{.Amd64}}",
		"sse2":                                "{{.I386}}",
		"power8":                              "{{.Ppc64}}",
		"rva22u64":                            "{{.Riscv64}}",
		"v8.0":                                "{{.Arm64}}",
		"a_fake_target":                       "{{.Target}}",
		"1.2.3":                               "{{.Version}}",
		"v1.2.3":                              "{{.Tag}}",
		"1-2-3":                               "{{.Major}}-{{.Minor}}-{{.Patch}}",
		"test-branch":                         "{{.Branch}}",
		"commit":                              "{{.Commit}}",
		"fullcommit":                          "{{.FullCommit}}",
		"shortcommit":                         "{{.ShortCommit}}",
		"binary":                              "{{.Binary}}",
		"proj":                                "{{.ProjectName}}",
		"github.com/goreleaser/goreleaser/v2": "{{ .ModulePath }}",
		"v2.0.0":                              "{{.Tag | incmajor }}",
		"2.0.0":                               "{{.Version | incmajor }}",
		"v1.3.0":                              "{{.Tag | incminor }}",
		"1.3.0":                               "{{.Version | incminor }}",
		"v1.2.4":                              "{{.Tag | incpatch }}",
		"1.2.4":                               "{{.Version | incpatch }}",
		"test release notes":                  "{{ .ReleaseNotes }}",
		"v1.2.2":                              "{{ .PreviousTag }}",
		"awesome release":                     "{{ .TagSubject }}",
		"awesome release\n\nanother line":     "{{ .TagContents }}",
		"another line":                        "{{ .TagBody }}",
		"runtime: " + runtime.GOOS:            "runtime: {{ .Runtime.Goos }}",
		"runtime: " + runtime.GOARCH:          "runtime: {{ .Runtime.Goarch }}",
		"artifact name: not-this-binary":      "artifact name: {{ .ArtifactName }}",
		"artifact ext: .exe":                  "artifact ext: {{ .ArtifactExt }}",
		"artifact path: /tmp/foo.exe":         "artifact path: {{ .ArtifactPath }}",
		"artifact basename: foo.exe":          "artifact basename: {{ base .ArtifactPath }}",
		"2023":                                `{{ .Now.Format "2006" }}`,
		"2023-03-09T02:06:02Z":                `{{ .Date }}`,
		"1678327562":                          `{{ .Timestamp }}`,
		"snapshot true":                       `snapshot {{.IsSnapshot}}`,
		"singletarget true":                   `singletarget {{.IsSingleTarget}}`,
		"nightly false":                       `nightly {{.IsNightly}}`,
		"draft true":                          `draft {{.IsDraft}}`,
		"dirty true":                          `dirty {{.IsGitDirty}}`,
		"clean false":                         `clean {{.IsGitClean}}`,
		"state dirty":                         `state {{.GitTreeState}}`,
		"env bar: barrrrr":                    `env bar: {{ envOrDefault "BAR" "barrrrr" }}`,
		"env foo: bar":                        `env foo: {{ envOrDefault "FOO" "barrrrr" }}`,
		"env foo is set: true":                `env foo is set: {{ isEnvSet "FOO" }}`,
		"/foo%2Fbar":                          `/{{ urlPathEscape .Env.WITH_SLASHES}}`,

		"artifact dir: " + filepath.FromSlash("/tmp"): "artifact dir: {{ dir .ArtifactPath }}",

		"remove this": "{{ filter .Env.MULTILINE \".*remove.*\" }}",
		"something with\nmultiple lines\nto test things": "{{ reverseFilter .Env.MULTILINE \".*remove.*\" }}",

		// maps
		"123": `{{ $m := map "a" "1" "b" "2" }}{{ index $m "a" }}{{ indexOrDefault $m "b" "10" }}{{ indexOrDefault $m "c" "3" }}{{ index $m "z" }}`,
	} {
		t.Run(expect, func(t *testing.T) {
			t.Parallel()
			result, err := New(ctx).WithArtifact(
				&artifact.Artifact{
					Name:      "not-this-binary",
					Path:      "/tmp/foo.exe",
					Goarch:    "amd64",
					Goos:      "linux",
					Goarm:     "6",
					Gomips:    "softfloat",
					Goamd64:   "v3",
					Goarm64:   "v8.0",
					Go386:     "sse2",
					Goppc64:   "power8",
					Goriscv64: "rva22u64",
					Target:    "a_fake_target",
					Extra: map[string]interface{}{
						artifact.ExtraBinary: "binary",
						artifact.ExtraExt:    ".exe",
					},
				},
			).Apply(tmpl)
			require.NoError(t, err)
			require.Equal(t, expect, result)
		})
	}

	t.Run("artifact without binary name", func(t *testing.T) {
		t.Parallel()
		result, err := New(ctx).WithArtifact(
			&artifact.Artifact{
				Name:   "another-binary",
				Goarch: "amd64",
				Goos:   "linux",
				Goarm:  "6",
			},
		).Apply("{{ .Binary }}")
		require.NoError(t, err)
		require.Equal(t, ctx.Config.ProjectName, result)
	})

	t.Run("template using artifact Fields with no artifact", func(t *testing.T) {
		t.Parallel()
		result, err := New(ctx).Apply("{{ .Os }}")
		require.ErrorAs(t, err, &Error{})
		require.EqualError(t, err, `template: failed to apply "{{ .Os }}": map has no entry for key "Os"`)
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

func TestWithEnvS(t *testing.T) {
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

	out, err = tpl.Apply(`{{ envOrDefault "NOPE" "no" }}`)
	require.NoError(t, err)
	require.Equal(t, "no", out)

	out, err = tpl.Apply(`{{ isEnvSet "NOPE" }}`)
	require.NoError(t, err)
	require.Equal(t, "false", out)
}

func TestSetEnv(t *testing.T) {
	ctx := testctx.New()
	tpl := New(ctx).
		WithEnvS([]string{
			"FOO=foo",
		}).
		SetEnv("BAR=bar").
		SetEnv("NOVAL=").
		SetEnv("=NOKEY").
		SetEnv("=").
		SetEnv("NOTHING")

	out, err := tpl.Apply("{{ .Env.FOO }}-{{ .Env.BAR }}")
	require.NoError(t, err)
	require.Equal(t, "foo-bar", out)
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
			Template: `{{ if contains "TEST_TEST_TEST" "TEST" }}it does{{else}}nope{{end}}`,
			Name:     "contains",
			Expected: "it does",
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

func TestApplyAll(t *testing.T) {
	tpl := New(testctx.New()).WithEnvS([]string{
		"FOO=bar",
	})
	t.Run("success", func(t *testing.T) {
		foo := "{{.Env.FOO}}"
		require.NoError(t, tpl.ApplyAll(&foo))
		require.Equal(t, "bar", foo)
	})
	t.Run("failure", func(t *testing.T) {
		foo := "{{.Env.FOO}}"
		bar := "{{.Env.NOPE}}"
		require.Error(t, tpl.ApplyAll(&foo, &bar))
		require.Equal(t, "bar", foo)
		require.Equal(t, "{{.Env.NOPE}}", bar)
	})
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
		{
			"bad template",
			"{{ .Env.NOPE }",
			Error{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(ctx).ApplySingleEnvOnly(tc.tpl)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.NotEmpty(t, err.Error())
				eerr, ok := err.(Error)
				if ok {
					require.Error(t, eerr.Unwrap())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInvalidTemplate(t *testing.T) {
	ctx := testctx.New()
	_, err := New(ctx).Apply("{{{.Foo}")
	require.ErrorAs(t, err, &Error{})
	require.EqualError(t, err, `template: failed to apply "{{{.Foo}": unexpected "{" in command`)
}

func TestEnvNotFound(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v1.2.4"))
	result, err := New(ctx).Apply("{{.Env.FOO}}")
	require.Empty(t, result)
	require.ErrorAs(t, err, &Error{})
	require.EqualError(t, err, `template: failed to apply "{{.Env.FOO}}": map has no entry for key "FOO"`)
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

func TestMdv2Escape(t *testing.T) {
	require.Equal(
		t,
		"aaa\\_\\*\\[\\]\\(\\)\\~\\`\\>\\#\\+\\-\\=\\|\\{\\}\\.\\!",
		mdv2Escape("aaa_*[]()~`>#+-=|{}.!"))
}

func TestInvalidMap(t *testing.T) {
	_, err := New(testctx.New()).Apply(`{{ $m := map "a" }}`)
	require.ErrorContains(t, err, "map expects even number of arguments, got 1")
}

func TestWithBuildOptions(t *testing.T) {
	// testtarget doesn ot set riscv64, it still should not fail to compile the template
	ts := "{{.Name}}_{{.Path}}_{{.Ext}}_{{.Target}}_{{.Os}}_{{.Arch}}_{{.Amd64}}_{{.Arm}}_{{.Mips}}{{with .Riscv64}}{{.}}{{end}}"
	out, err := New(testctx.New()).WithBuildOptions(build.Options{
		Name: "name",
		Path: "./path",
		Ext:  ".ext",
		Target: testTarget{
			Target:  "target",
			Goos:    "os",
			Goarch:  "arch",
			Goamd64: "amd64",
			Goarm:   "arm",
			Gomips:  "mips",
		},
	}).Apply(ts)
	require.NoError(t, err)
	require.Equal(t, "name_./path_.ext_target_os_arch_amd64_arm_mips", out)
}

func TestReuseTpl(t *testing.T) {
	tp := New(testctx.New()).WithExtraFields(Fields{
		"foo": "bar",
	})
	s1, err := tp.Apply("{{.foo}}")
	require.NoError(t, err)
	require.Equal(t, "bar", s1)

	s2, err := tp.WithExtraFields(Fields{"foo": "not-bar"}).Apply("{{.foo}}")
	require.NoError(t, err)
	require.Equal(t, "not-bar", s2)

	s3, err := tp.Apply("{{.foo}}")
	require.NoError(t, err)
	require.Equal(t, "bar", s3)
}

func TestSlice(t *testing.T) {
	ctx := testctx.New(
		testctx.WithVersion("1.2.3"),
		testctx.WithCurrentTag("5.6.7"),
	)

	artifact := &artifact.Artifact{
		Name:   "name",
		Goos:   "darwin",
		Goarch: "amd64",
		Goarm:  "7",
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "binary",
		},
	}

	source := []string{
		"flag",
		"{{.Version}}",
		"{{.Os}}",
		"{{.Arch}}",
		"{{.Arm}}",
		"{{.Binary}}",
		"{{.ArtifactName}}",
	}

	expected := []string{
		"-testflag=flag",
		"-testflag=1.2.3",
		"-testflag=darwin",
		"-testflag=amd64",
		"-testflag=7",
		"-testflag=binary",
		"-testflag=name",
	}

	flags, err := New(ctx).WithArtifact(artifact).Slice(source, WithPrefix("-testflag="))
	require.NoError(t, err)
	require.Len(t, flags, 7)
	require.Equal(t, expected, flags)
}

func TestSliceInvalid(t *testing.T) {
	ctx := testctx.New()
	source := []string{
		"{{.Version}",
	}
	flags, err := New(ctx).Slice(source)
	require.ErrorAs(t, err, &Error{})
	require.Nil(t, flags)
}

func TestSliceIgnoreEmptyFlags(t *testing.T) {
	ctx := testctx.New()
	source := []string{
		"{{if eq 1 2}}-ignore-me{{end}}",
	}
	flags, err := New(ctx).Slice(source, NonEmpty())
	require.NoError(t, err)
	require.Empty(t, flags)
}

type testTarget struct {
	Target  string
	Goos    string
	Goarch  string
	Goamd64 string
	Goarm   string
	Gomips  string
}

func (t testTarget) String() string { return t.Target }

func (t testTarget) Fields() map[string]string {
	return map[string]string{
		target:   t.Target,
		KeyOS:    t.Goos,
		KeyArch:  t.Goarch,
		KeyAmd64: t.Goamd64,
		KeyArm:   t.Goarm,
		KeyMips:  t.Gomips,
	}
}
