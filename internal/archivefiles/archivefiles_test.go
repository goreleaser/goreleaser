package archivefiles

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestEval(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ctx := testctx.NewWithCfg(config.Project{
		Env: []string{"OWNER=carlos", "FOLDER=d"},
	})
	ctx.Git.CommitDate = now
	tmpl := tmpl.New(ctx)

	t.Run("invalid glob", func(t *testing.T) {
		_, err := Eval(tmpl, []config.File{
			{
				Source:      "../testdata/**/nope.txt",
				Destination: "var/foobar/d.txt",
			},
		})
		require.Error(t, err)
	})

	t.Run("templated src", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/**/{{ .Env.FOLDER }}.txt",
				Destination: "var/foobar/",
			},
		})
		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "var/foobar/d.txt",
			},
		}, result)
	})

	t.Run("templated src error", func(t *testing.T) {
		_, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/**/{{ .Env.NOPE }}.txt",
				Destination: "var/foobar/d.txt",
			},
		})
		testlib.RequireTemplateError(t, err)
	})

	t.Run("templated info", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar/",
				Info: config.FileInfo{
					MTime: "{{.CommitDate}}",
					Owner: "{{ .Env.OWNER }}",
					Group: "{{ .Env.OWNER }}",
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "var/foobar/d.txt",
				Info: config.FileInfo{
					MTime:       now.UTC().Format(time.RFC3339),
					ParsedMTime: now.UTC(),
					Owner:       "carlos",
					Group:       "carlos",
				},
			},
		}, result)
	})

	t.Run("template info errors", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			_, err := Eval(tmpl, []config.File{{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar",
				Info: config.FileInfo{
					Owner: "{{ .Env.NOPE }}",
				},
			}})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("group", func(t *testing.T) {
			_, err := Eval(tmpl, []config.File{{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar",
				Info: config.FileInfo{
					Group: "{{ .Env.NOPE }}",
				},
			}})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("mtime", func(t *testing.T) {
			_, err := Eval(tmpl, []config.File{{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar",
				Info: config.FileInfo{
					MTime: "{{ .Env.NOPE }}",
				},
			}})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("mtime format", func(t *testing.T) {
			_, err := Eval(tmpl, []config.File{{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar",
				Info: config.FileInfo{
					MTime: "2005-123-123",
				},
			}})
			require.Error(t, err)
		})
	})

	t.Run("single file", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar",
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "var/foobar/d.txt",
			},
		}, result)
	})

	t.Run("rlcp", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{{
			Source:      "./testdata/a/**/*",
			Destination: "foo/bar",
		}})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{Source: "testdata/a/b/a.txt", Destination: "foo/bar/a.txt"},
			{Source: "testdata/a/b/c/d.txt", Destination: "foo/bar/c/d.txt"},
		}, result)
	})

	t.Run("rlcp empty destination", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{{
			Source: "./testdata/a/**/*",
		}})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{Source: "testdata/a/b/a.txt", Destination: "testdata/a/b/a.txt"},
			{Source: "testdata/a/b/c/d.txt", Destination: "testdata/a/b/c/d.txt"},
		}, result)
	})

	t.Run("rlcp no results", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{{
			Source:      "./testdata/abc/**/*",
			Destination: "foo/bar",
		}})

		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("strip parent plays nicely with destination omitted", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{{Source: "./testdata/a/b", StripParent: true}})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{Source: "testdata/a/b/a.txt", Destination: "a.txt"},
			{Source: "testdata/a/b/c/d.txt", Destination: "d.txt"},
		}, result)
	})

	t.Run("strip parent plays nicely with destination as an empty string", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{{Source: "./testdata/a/b", Destination: "", StripParent: true}})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{Source: "testdata/a/b/a.txt", Destination: "a.txt"},
			{Source: "testdata/a/b/c/d.txt", Destination: "d.txt"},
		}, result)
	})

	t.Run("match multiple files within tree without destination", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{{Source: "./testdata/a"}})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{Source: "testdata/a/a.txt", Destination: "testdata/a/a.txt"},
			{Source: "testdata/a/b/a.txt", Destination: "testdata/a/b/a.txt"},
			{Source: "testdata/a/b/c/d.txt", Destination: "testdata/a/b/c/d.txt"},
		}, result)
	})

	t.Run("match multiple files within tree specific destination", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/a",
				Destination: "usr/local/test",
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "usr/local/test/a.txt",
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
			{
				Source:      "testdata/a/b/a.txt",
				Destination: "usr/local/test/b/a.txt",
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "usr/local/test/b/c/d.txt",
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
		}, result)
	})

	t.Run("match multiple files within tree specific destination stripping parents", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/a",
				Destination: "usr/local/test",
				StripParent: true,
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "usr/local/test/a.txt",
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "usr/local/test/d.txt",
				Info: config.FileInfo{
					Owner:       "carlos",
					Group:       "users",
					Mode:        0o755,
					ParsedMTime: now,
				},
			},
		}, result)
	})
}

func TestStrlcp(t *testing.T) {
	for k, v := range map[string][2]string{
		"/var/":       {"/var/lib/foo", "/var/share/aaa"},
		"/var/lib/":   {"/var/lib/foo", "/var/lib/share/aaa"},
		"/usr/share/": {"/usr/share/lib", "/usr/share/bin"},
		"/usr/":       {"/usr/share/lib", "/usr/bin"},
	} {
		t.Run(k, func(t *testing.T) {
			require.Equal(t, k, strlcp(v[0], v[1]))
		})
	}
}

