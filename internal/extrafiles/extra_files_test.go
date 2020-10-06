package extrafiles

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestShouldGetAllFiles(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/file1.golden"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 1, len(files))

	path, ok := files["file1.golden"]
	require.True(t, ok)
	require.Equal(t, path, "./testdata/file1.golden")
}

func TestShouldGetAllFilesWithGoldenExtension(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/*.golden"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 2, len(files))

	path, ok := files["file1.golden"]
	require.True(t, ok)
	require.Equal(t, path, "testdata/file1.golden")

	path, ok = files["file2.golden"]
	require.True(t, ok)
	require.Equal(t, path, "testdata/file2.golden")
}

func TestShouldGetAllFilesInsideTestdata(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/*"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))

	path, ok := files["file1.golden"]
	require.True(t, ok)
	require.Equal(t, path, "testdata/file1.golden")

	path, ok = files["file2.golden"]
	require.True(t, ok)
	require.Equal(t, path, "testdata/file2.golden")

	path, ok = files["file3.gold"]
	require.True(t, ok)
	require.Equal(t, path, "testdata/file3.gold")
}
