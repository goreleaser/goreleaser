package extrafiles

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestTemplate(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/file{{ .Env.ONE }}.golden"},
	}

	ctx := context.New(config.Project{})
	ctx.Env["ONE"] = "1"

	files, err := Find(ctx, globs)
	require.NoError(t, err)
	require.Len(t, files, 1)

	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
}

func TestBadTemplate(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/file{{ .Env.NOPE }}.golden"},
	}

	ctx := context.New(config.Project{})

	files, err := Find(ctx, globs)
	require.Empty(t, files)
	require.EqualError(t, err, `failed to apply template to glob "./testdata/file{{ .Env.NOPE }}.golden": template: tmpl:1:22: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`)
}

func TestShouldGetSpecificFile(t *testing.T) {
	globs := []config.ExtraFile{
		{},                        // empty glob, will be ignored
		{Glob: "./testdata/sub3"}, // will get a file1.golden as well, but will be overridden
		{Glob: "./testdata/file1.golden"},
	}

	files, err := Find(context.New(config.Project{}), globs)
	require.NoError(t, err)
	require.Len(t, files, 1)

	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
}

func TestFailToGetSpecificFile(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/file453.golden"},
	}

	files, err := Find(context.New(config.Project{}), globs)
	require.EqualError(t, err, "globbing failed for pattern ./testdata/file453.golden: matching \"./testdata/file453.golden\": file does not exist")
	require.Empty(t, files)
}

func TestShouldGetFilesWithSuperStar(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./**/file?.golden"},
	}

	files, err := Find(context.New(config.Project{}), globs)
	require.NoError(t, err)
	require.Len(t, files, 3)

	require.Equal(t, "testdata/file2.golden", files["file2.golden"])
	require.Equal(t, "testdata/sub3/file1.golden", files["file1.golden"])
	require.Equal(t, "testdata/sub/file5.golden", files["file5.golden"])
}

func TestShouldGetAllFilesWithGoldenExtension(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/*.golden"},
	}

	files, err := Find(context.New(config.Project{}), globs)
	require.NoError(t, err)
	require.Len(t, files, 2)

	require.Equal(t, "testdata/file1.golden", files["file1.golden"])
	require.Equal(t, "testdata/file2.golden", files["file2.golden"])
}

func TestShouldGetAllFilesInsideTestdata(t *testing.T) {
	globs := []config.ExtraFile{
		{Glob: "./testdata/*"},
	}

	files, err := Find(context.New(config.Project{}), globs)
	require.NoError(t, err)
	require.Len(t, files, 4)

	require.Equal(t, "testdata/sub3/file1.golden", files["file1.golden"])
	require.Equal(t, "testdata/file2.golden", files["file2.golden"])
	require.Equal(t, "testdata/file3.gold", files["file3.gold"])
	require.Equal(t, "testdata/sub/file5.golden", files["file5.golden"])
}
