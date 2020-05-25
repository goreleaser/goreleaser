package extrafiles

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestShouldGetAllFiles(t *testing.T) {
	assert := assert.New(t)

	globs := []config.ExtraFile{
		{Glob: "./testdata/file1.golden"},
	}

	files, err := Find(globs)
	assert.NoError(err)
	assert.Equal(1, len(files))

	path, ok := files["file1.golden"]
	assert.True(ok)
	assert.Equal(path, "./testdata/file1.golden")
}

func TestShouldGetAllFilesWithGoldenExtension(t *testing.T) {
	assert := assert.New(t)

	globs := []config.ExtraFile{
		{Glob: "./testdata/*.golden"},
	}

	files, err := Find(globs)
	assert.NoError(err)
	assert.Equal(2, len(files))

	path, ok := files["file1.golden"]
	assert.True(ok)
	assert.Equal(path, "testdata/file1.golden")

	path, ok = files["file2.golden"]
	assert.True(ok)
	assert.Equal(path, "testdata/file2.golden")
}

func TestShouldGetAllFilesInsideTestdata(t *testing.T) {
	assert := assert.New(t)

	globs := []config.ExtraFile{
		{Glob: "./testdata/*"},
	}

	files, err := Find(globs)
	assert.NoError(err)
	assert.Equal(3, len(files))

	path, ok := files["file1.golden"]
	assert.True(ok)
	assert.Equal(path, "testdata/file1.golden")

	path, ok = files["file2.golden"]
	assert.True(ok)
	assert.Equal(path, "testdata/file2.golden")

	path, ok = files["file3.gold"]
	assert.True(ok)
	assert.Equal(path, "testdata/file3.gold")
}
