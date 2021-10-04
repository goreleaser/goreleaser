package milestone

import (
	"errors"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDefaultWithRepoConfig(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:githubowner/githubrepo.git")

	ctx := &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{
				{
					Repo: config.Repo{
						Name:  "configrepo",
						Owner: "configowner",
					},
				},
			},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "configrepo", ctx.Config.Milestones[0].Repo.Name)
	require.Equal(t, "configowner", ctx.Config.Milestones[0].Repo.Owner)
}

func TestDefaultWithRepoRemote(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:githubowner/githubrepo.git")

	ctx := context.New(config.Project{
		Milestones: []config.Milestone{{}},
	})
	ctx.TokenType = context.TokenTypeGitHub
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "githubrepo", ctx.Config.Milestones[0].Repo.Name)
	require.Equal(t, "githubowner", ctx.Config.Milestones[0].Repo.Owner)
}

func TestDefaultWithNameTemplate(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{
				{
					NameTemplate: "confignametemplate",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "confignametemplate", ctx.Config.Milestones[0].NameTemplate)
}

func TestDefaultWithoutGitRepo(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{{}},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	require.EqualError(t, Pipe{}.Default(ctx), "current folder is not a git repository")
	require.Empty(t, ctx.Config.Milestones[0].Repo.String())
}

func TestDefaultWithoutGitRepoOrigin(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{{}},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	testlib.GitInit(t)
	require.EqualError(t, Pipe{}.Default(ctx), "no remote configured to list refs from")
	require.Empty(t, ctx.Config.Milestones[0].Repo.String())
}

func TestDefaultWithoutGitRepoSnapshot(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{{}},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Snapshot = true
	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Milestones[0].Repo.String())
}

func TestDefaultWithoutNameTemplate(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{{}},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "{{ .Tag }}", ctx.Config.Milestones[0].NameTemplate)
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPublishCloseDisabled(t *testing.T) {
	ctx := context.New(config.Project{
		Milestones: []config.Milestone{
			{
				Close: false,
			},
		},
	})
	client := &DummyClient{}
	testlib.AssertSkipped(t, doPublish(ctx, client))
	require.Equal(t, "", client.ClosedMilestone)
}

func TestPublishCloseEnabled(t *testing.T) {
	ctx := context.New(config.Project{
		Milestones: []config.Milestone{
			{
				Close:        true,
				NameTemplate: defaultNameTemplate,
				Repo: config.Repo{
					Name:  "configrepo",
					Owner: "configowner",
				},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.0.0"
	client := &DummyClient{}
	require.NoError(t, doPublish(ctx, client))
	require.Equal(t, "v1.0.0", client.ClosedMilestone)
}

func TestPublishCloseError(t *testing.T) {
	config := config.Project{
		Milestones: []config.Milestone{
			{
				Close:        true,
				NameTemplate: defaultNameTemplate,
				Repo: config.Repo{
					Name:  "configrepo",
					Owner: "configowner",
				},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "v1.0.0"
	client := &DummyClient{
		FailToCloseMilestone: true,
	}
	require.NoError(t, doPublish(ctx, client))
	require.Equal(t, "", client.ClosedMilestone)
}

func TestPublishCloseFailOnError(t *testing.T) {
	config := config.Project{
		Milestones: []config.Milestone{
			{
				Close:        true,
				FailOnError:  true,
				NameTemplate: defaultNameTemplate,
				Repo: config.Repo{
					Name:  "configrepo",
					Owner: "configowner",
				},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "v1.0.0"
	client := &DummyClient{
		FailToCloseMilestone: true,
	}
	require.Error(t, doPublish(ctx, client))
	require.Equal(t, "", client.ClosedMilestone)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Milestones: []config.Milestone{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

type DummyClient struct {
	ClosedMilestone      string
	FailToCloseMilestone bool
}

func (c *DummyClient) CloseMilestone(ctx *context.Context, repo client.Repo, title string) error {
	if c.FailToCloseMilestone {
		return errors.New("milestone failed")
	}

	c.ClosedMilestone = title

	return nil
}

func (c *DummyClient) CreateRelease(ctx *context.Context, body string) (string, error) {
	return "", nil
}

func (c *DummyClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return "", nil
}

func (c *DummyClient) GetDefaultBranch(ctx *context.Context, repo client.Repo) (string, error) {
	return "", errors.New("DummyClient does not yet implement GetDefaultBranch")
}

func (c *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo client.Repo, content []byte, path, msg string) error {
	return nil
}

func (c *DummyClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) error {
	return nil
}

func (c *DummyClient) Changelog(ctx *context.Context, repo client.Repo, prev, current string) (string, error) {
	return "", errors.New("not implemented")
}
