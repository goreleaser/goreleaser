package archivefiles

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestEval(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tmpl := tmpl.New(context.New(config.Project{}))

	t.Run("single file", func(t *testing.T) {
		result, err := Eval(tmpl, []config.File{
			{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar/d.txt",
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "var/foobar/d.txt/testdata/a/b/c/d.txt",
			},
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
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "usr/local/test/testdata/a/a.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
				},
			},
			{
				Source:      "testdata/a/b/a.txt",
				Destination: "usr/local/test/testdata/a/b/a.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
				},
			},
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "usr/local/test/testdata/a/b/c/d.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
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
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "usr/local/test/a.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
				},
			},
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "usr/local/test/d.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0o755,
					MTime: now,
				},
			},
		}, result)
	})
}
