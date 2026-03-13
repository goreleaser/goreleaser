package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestFindMains(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/foo/bar/v10"), 0o644))
	for _, m := range []string{
		"main.go",
		"cmd/a/main.go",
		"cmd/b/main.go",
		"cmd/c/main.go",
		"c/main.go",
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(m)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, m), []byte("package main\nfunc main(){}"), 0o644))
	}

	mains, err := findMains(dir, "./...")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"bar": ".",
		"a":   "./cmd/a",
		"b":   "./cmd/b",
		"c":   "./c",
	}, mains)
}

func TestFindMainsErrors(t *testing.T) {
	mains, err := findMains(t.TempDir(), "./...")
	require.ErrorIs(t, err, errNoMains)
	require.Nil(t, mains)
}

func TestComputeBinaryName(t *testing.T) {
	for expected, pkg := range map[string]packages.Package{
		"foo": {
			Dir: "./cmd/foo",
		},
		"bar": {
			Module: &packages.Module{
				Path: "github.com/foo/bar",
				Dir:  "./bar/",
			},
			Dir: "./bar/",
		},
		"zaz": {
			Module: &packages.Module{
				Path: "github.com/foo/zaz/v10",
				Dir:  ".",
			},
			Dir: ".",
		},
	} {
		t.Run(expected, func(t *testing.T) {
			require.Equal(t, expected, computeBinaryName(&pkg))
		})
	}
}
