package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestPipeDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeWithoutIDsThenDoesNotFilter(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	srcfile, err := os.Create(filepath.Join(folder, "source.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, srcfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())
	filteredtarfile, err := os.Create(filepath.Join(folder, "filtered.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, filteredtarfile.Close())
	filtereddebfile, err := os.Create(filepath.Join(folder, "filtered.deb"))
	require.NoError(t, err)
	require.NoError(t, filtereddebfile.Close())

	config := config.Project{
		Dist: folder,
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
		Extra: map[string]interface{}{
			"ID": "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
		Extra: map[string]interface{}{
			"ID": "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "filtered.tar.gz",
		Path: filteredtarfile.Name(),
		Extra: map[string]interface{}{
			"ID": "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "filtered.deb",
		Path: filtereddebfile.Name(),
		Extra: map[string]interface{}{
			"ID": "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: "source.tar.gz",
		Path: srcfile.Name(),
		Extra: map[string]interface{}{
			"Format": "tar.gz",
		},
	})
	client := &client.Mock{}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.Contains(t, client.UploadedFileNames, "source.tar.gz")
	require.Contains(t, client.UploadedFileNames, "bin.deb")
	require.Contains(t, client.UploadedFileNames, "bin.tar.gz")
	require.Contains(t, client.UploadedFileNames, "filtered.deb")
	require.Contains(t, client.UploadedFileNames, "filtered.tar.gz")
}

func TestRunPipeWithIDsThenFilters(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())
	filteredtarfile, err := os.Create(filepath.Join(folder, "filtered.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, filteredtarfile.Close())
	filtereddebfile, err := os.Create(filepath.Join(folder, "filtered.deb"))
	require.NoError(t, err)
	require.NoError(t, filtereddebfile.Close())

	config := config.Project{
		Dist: folder,
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			IDs: []string{"foo"},
			ExtraFiles: []config.ExtraFile{
				{Glob: "./testdata/**/*"},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
		Extra: map[string]interface{}{
			"ID": "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
		Extra: map[string]interface{}{
			"ID": "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "filtered.tar.gz",
		Path: filteredtarfile.Name(),
		Extra: map[string]interface{}{
			"ID": "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "filtered.deb",
		Path: filtereddebfile.Name(),
		Extra: map[string]interface{}{
			"ID": "bar",
		},
	})
	client := &client.Mock{}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.Contains(t, client.UploadedFileNames, "bin.deb")
	require.Contains(t, client.UploadedFileNames, "bin.tar.gz")
	require.Contains(t, client.UploadedFileNames, "f1")
	require.NotContains(t, client.UploadedFileNames, "filtered.deb")
	require.NotContains(t, client.UploadedFileNames, "filtered.tar.gz")
}

func TestRunPipeReleaseCreationFailed(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	client := &client.Mock{
		FailToCreateRelease: true,
	}
	require.Error(t, doPublish(ctx, client))
	require.False(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
}

func TestRunPipeWithFileThatDontExist(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: "/nope/nope/nope",
	})
	client := &client.Mock{}
	require.Error(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
}

func TestRunPipeUploadFailure(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	client := &client.Mock{
		FailToUpload: true,
	}
	require.EqualError(t, doPublish(ctx, client), "failed to upload bin.tar.gz after 1 tries: upload failed")
	require.True(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
}

func TestRunPipeExtraFileNotFound(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			ExtraFiles: []config.ExtraFile{
				{Glob: "./testdata/f1.txt"},
				{Glob: "./nope"},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	client := &client.Mock{}
	require.EqualError(t, doPublish(ctx, client), "globbing failed for pattern ./nope: matching \"./nope\": file does not exist")
	require.True(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
}

func TestRunPipeExtraOverride(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			ExtraFiles: []config.ExtraFile{
				{Glob: "./testdata/**/*"},
				{Glob: "./testdata/upload_same_name/f1"},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	client := &client.Mock{}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.Contains(t, client.UploadedFileNames, "f1")
	require.True(t, strings.HasSuffix(client.UploadedFilePaths["f1"], "testdata/upload_same_name/f1"))
}

func TestRunPipeUploadRetry(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	client := &client.Mock{
		FailFirstUpload: true,
	}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitHub
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultWithGitlab(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitlab.com:gitlabowner/gitlabrepo.git")

	ctx := context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitLab
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "gitlabrepo", ctx.Config.Release.GitLab.Name)
	require.Equal(t, "gitlabowner", ctx.Config.Release.GitLab.Owner)
}

func TestDefaultWithGitea(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitea.example.com:giteaowner/gitearepo.git")

	ctx := context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitea
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "gitearepo", ctx.Config.Release.Gitea.Name)
	require.Equal(t, "giteaowner", ctx.Config.Release.Gitea.Owner)
}

func TestDefaultPreReleaseAuto(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	t.Run("auto-release", func(t *testing.T) {
		ctx := context.New(config.Project{
			Release: config.Release{
				Prerelease: "auto",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, false, ctx.PreRelease)
	})

	t.Run("auto-rc", func(t *testing.T) {
		ctx := context.New(config.Project{
			Release: config.Release{
				Prerelease: "auto",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major:      1,
			Minor:      0,
			Patch:      0,
			Prerelease: "rc1",
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, true, ctx.PreRelease)
	})

	t.Run("auto-rc-github-setup", func(t *testing.T) {
		ctx := context.New(config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name:  "foo",
					Owner: "foo",
				},
				Prerelease: "auto",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major:      1,
			Minor:      0,
			Patch:      0,
			Prerelease: "rc1",
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, true, ctx.PreRelease)
	})
}

func TestDefaultPipeDisabled(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := context.New(config.Project{
		Release: config.Release{
			Disable: true,
		},
	})
	ctx.TokenType = context.TokenTypeGitHub
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultFilled(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name:  "foo",
					Owner: "bar",
				},
			},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Release.GitHub.Name)
	require.Equal(t, "bar", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultNotAGitRepo(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	require.EqualError(t, Pipe{}.Default(ctx), "current folder is not a git repository")
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutOrigin(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	testlib.GitInit(t)
	require.EqualError(t, Pipe{}.Default(ctx), "no remote configured to list refs from")
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultNotAGitRepoSnapshot(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Snapshot = true
	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutRemote(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	require.Error(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultMultipleReleasesDefined(t *testing.T) {
	ctx := context.New(config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "githubName",
				Name:  "githubName",
			},
			GitLab: config.Repo{
				Owner: "gitlabOwner",
				Name:  "gitlabName",
			},
			Gitea: config.Repo{
				Owner: "giteaOwner",
				Name:  "giteaName",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), ErrMultipleReleases.Error())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Release: config.Release{
				Disable: true,
			},
		})
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(context.New(config.Project{})))
	})
}
