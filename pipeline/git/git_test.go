package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipe(t *testing.T) {
	var assert = assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.NotEmpty(ctx.Git.CurrentTag)
	assert.NotEmpty(ctx.Git.PreviousTag)
	assert.NotEmpty(ctx.Git.Diff)
}

func TestGoodRepo(t *testing.T) {
	var assert = assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.doRun(ctx, getFixture("good-repo")))
	assert.Equal(
		context.GitInfo{
			CurrentTag:  "v0.0.2",
			PreviousTag: "v0.0.1",
			Diff:        "593de7f commit5\nf365005 commit4\n3eb6c7b commit3\n",
			Commit:      "593de7f025f3817cc0a56bb11f5a6f0131c67452",
		},
		ctx.Git,
	)
}

func TestSingleCommitNoTags(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.doRun(ctx, getFixture("single-commit-no-tags-repo")))
	assert.Equal(
		context.GitInfo{
			CurrentTag:  "211cca43da0ebbe5109c1cf09bee3ea0bb0bf04f",
			PreviousTag: "211cca43da0ebbe5109c1cf09bee3ea0bb0bf04f",
			Commit:      "211cca43da0ebbe5109c1cf09bee3ea0bb0bf04f",
		},
		ctx.Git,
	)
}

func TestSingleCommit(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.doRun(ctx, getFixture("single-commit-repo")))
	assert.Equal(
		context.GitInfo{
			CurrentTag:  "v0.0.1",
			PreviousTag: "4bf27bfd08049ae6187cefa5e9d50e2e0f205ebe",
			Commit:      "4bf27bfd08049ae6187cefa5e9d50e2e0f205ebe",
		},
		ctx.Git,
	)
}

func TestNewRepository(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.doRun(ctx, getFixture("new-repo")))
	assert.Equal(context.GitInfo{}, ctx.Git)
}

func TestInvalidTagFormat(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(
		Pipe{}.doRun(ctx, getFixture("invalid-tag-format-repo")),
		"invalid-tag-name is not in a valid version format",
	)
	assert.Equal(
		context.GitInfo{
			CurrentTag:  "invalid-tag-name",
			PreviousTag: "v0.0.1",
			Diff:        "7cafca4 commit5\n1781c0e commit4\n633c559 commit3\n",
			Commit:      "7cafca4c382e2d83b123281bb31dd7b4f0e19a8b",
		},
		ctx.Git,
	)
}

func getFixture(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "fixtures", name)
}
