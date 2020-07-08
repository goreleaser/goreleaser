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
	"github.com/stretchr/testify/assert"
)

func TestDefaultWithGithubConfig(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:githubowner/githubrepo.git")

	var ctx = &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{
				{
					GitHub: config.Repo{
						Name:  "configrepo",
						Owner: "configowner",
					},
				},
			},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "configrepo", ctx.Config.Milestones[0].GitHub.Name)
	assert.Equal(t, "configowner", ctx.Config.Milestones[0].GitHub.Owner)
}

func TestDefaultWithGithubRemote(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:githubowner/githubrepo.git")

	var ctx = context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitHub
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "githubrepo", ctx.Config.Milestones[0].GitHub.Name)
	assert.Equal(t, "githubowner", ctx.Config.Milestones[0].GitHub.Owner)
}

func TestDefaultWithGitlabConfig(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitlab.com:gitlabowner/gitlabrepo.git")

	var ctx = &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{
				{
					GitLab: config.Repo{
						Name:  "configrepo",
						Owner: "configowner",
					},
				},
			},
		},
	}
	ctx.TokenType = context.TokenTypeGitLab
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "configrepo", ctx.Config.Milestones[0].GitLab.Name)
	assert.Equal(t, "configowner", ctx.Config.Milestones[0].GitLab.Owner)
}

func TestDefaultWithGitlabRemote(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitlab.com:gitlabowner/gitlabrepo.git")

	var ctx = context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitLab
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "gitlabrepo", ctx.Config.Milestones[0].GitLab.Name)
	assert.Equal(t, "gitlabowner", ctx.Config.Milestones[0].GitLab.Owner)
}

func TestDefaultWithGiteaConfig(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitea.example.com:giteaowner/gitearepo.git")

	var ctx = &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{
				{
					Gitea: config.Repo{
						Name:  "configrepo",
						Owner: "configowner",
					},
				},
			},
		},
	}
	ctx.TokenType = context.TokenTypeGitea
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "configrepo", ctx.Config.Milestones[0].Gitea.Name)
	assert.Equal(t, "configowner", ctx.Config.Milestones[0].Gitea.Owner)
}

func TestDefaultWithGiteaRemote(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitea.example.com:giteaowner/gitearepo.git")

	var ctx = context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitea
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "gitearepo", ctx.Config.Milestones[0].Gitea.Name)
	assert.Equal(t, "giteaowner", ctx.Config.Milestones[0].Gitea.Owner)
}

func TestDefaultWithNameTemplate(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{
				{
					NameTemplate: "confignametemplate",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "confignametemplate", ctx.Config.Milestones[0].NameTemplate)
}

func TestDefaultWithoutGitRepo(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	assert.EqualError(t, Pipe{}.Default(ctx), "current folder is not a git repository")
	assert.Empty(t, ctx.Config.Milestones[0].GitHub.String())
}

func TestDefaultWithoutGitRepoOrigin(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	testlib.GitInit(t)
	assert.EqualError(t, Pipe{}.Default(ctx), "repository doesn't have an `origin` remote")
	assert.Empty(t, ctx.Config.Milestones[0].GitHub.String())
}

func TestDefaultWithoutGitRepoSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Snapshot = true
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.Milestones[0].GitHub.String())
}

func TestDefaultWithoutNameTemplate(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Milestones: []config.Milestone{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "{{ .Tag }}", ctx.Config.Milestones[0].NameTemplate)
}

func TestString(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestPublishCloseDisabled(t *testing.T) {
	var ctx = context.New(config.Project{
		Milestones: []config.Milestone{
			{
				Close: false,
			},
		},
	})
	client := &DummyClient{}
	testlib.AssertSkipped(t, doPublish(ctx, client))
	assert.Equal(t, "", client.ClosedMilestone)
}

func TestPublishCloseEnabled(t *testing.T) {
	var ctx = context.New(config.Project{
		Milestones: []config.Milestone{
			{
				Close: true,
				GitHub: config.Repo{
					Name:  "configrepo",
					Owner: "configowner",
				},
				NameTemplate: defaultNameTemplate,
			},
		},
	})
	ctx.Git.CurrentTag = "v1.0.0"
	client := &DummyClient{}
	assert.NoError(t, doPublish(ctx, client))
	assert.Equal(t, "v1.0.0", client.ClosedMilestone)
}

func TestPublishCloseError(t *testing.T) {
	var config = config.Project{
		Milestones: []config.Milestone{
			{
				Close: true,
				GitHub: config.Repo{
					Name:  "configrepo",
					Owner: "configowner",
				},
				NameTemplate: defaultNameTemplate,
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "v1.0.0"
	client := &DummyClient{
		FailToCloseMilestone: true,
	}
	assert.Error(t, doPublish(ctx, client))
	assert.Equal(t, "", client.ClosedMilestone)
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

func (c *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo client.Repo, content []byte, path, msg string) error {
	return nil
}

func (c *DummyClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) error {
	return nil
}
