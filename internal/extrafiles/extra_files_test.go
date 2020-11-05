package extrafiles

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestShouldGetSpecificFile(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/file1.golden"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 1, len(files))

	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
}

func TestShouldGetFilesWithSuperStar(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./**/file?.golden"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))

	require.Equal(t, "testdata/file2.golden", files["file2.golden"])
	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
	require.Equal(t, "testdata/sub/file5.golden", files["file5.golden"])
}

func TestShouldGetAllFilesWithGoldenExtension(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/*.golden"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 2, len(files))

	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
	require.Equal(t, "testdata/file2.golden", files["file2.golden"])
}

func TestShouldGetAllFilesInsideTestdata(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/*"},
	}

	files, err := Find(globs)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))

	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
	require.Equal(t, "testdata/file2.golden", files["file2.golden"])
	require.Equal(t, "testdata/file3.gold", files["file3.gold"])
}
