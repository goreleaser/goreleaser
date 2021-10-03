package scoop

import (
	ctx "context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)

	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			ProjectName: "barr",
			Builds: []config.Build{
				{
					Binary: "foo",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
				},
				{
					Binary: "bar",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
					Ignore: []config.IgnoredBuild{
						{Goos: "darwin", Goarch: "amd64"},
					},
				},
				{
					Binary: "foobar",
					Goos:   []string{"linux"},
					Goarch: []string{"amd64"},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Scoop.Name)
	require.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Scoop.CommitMessageTemplate)
}

func Test_doRun(t *testing.T) {
	folder := testlib.Mktmp(t)
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	type args struct {
		ctx    *context.Context
		client *DummyClient
	}

	type asserter func(*testing.T, args)
	type errChecker func(*testing.T, error)
	shouldErr := func(msg string) errChecker {
		return func(t *testing.T, err error) {
			t.Helper()
			require.Error(t, err)
			require.EqualError(t, err, msg)
		}
	}
	noAssertions := func(t *testing.T, _ args) {
		t.Helper()
	}
	shouldNotErr := func(t *testing.T, err error) {
		t.Helper()
		require.NoError(t, err)
	}

	tests := []struct {
		name               string
		args               args
		artifacts          []*artifact.Artifact
		assertRunError     errChecker
		assertPublishError errChecker
		assert             asserter
	}{
		{
			"valid public github",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Folder:      "scoops",
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldNotErr,
			func(t *testing.T, a args) {
				t.Helper()
				require.Equal(t, "scoops/run-pipe.json", a.client.Path)
			},
		},
		{
			"wrap in directory",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz", WrapInDirectory: "true"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"Wrap": "foo_1.0.1_windows_amd64",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"Wrap": "foo_1.0.1_windows_386",
					},
				},
			},
			shouldNotErr,
			shouldNotErr,
			noAssertions,
		},
		{
			"valid enterprise github",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						GitHubURLs: config.GitHubURLs{Download: "https://api.custom.github.enterprise.com"},
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldNotErr,
			func(t *testing.T, a args) {
				t.Helper()
				require.Equal(t, "run-pipe.json", a.client.Path)
			},
		},
		{
			"valid public gitlab",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitLab,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitLab: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://gitlab.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
				},
			},
			shouldNotErr,
			shouldNotErr,
			noAssertions,
		},
		{
			"valid enterprise gitlab",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitLab,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						GitHubURLs: config.GitHubURLs{Download: "https://api.custom.gitlab.enterprise.com"},
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://gitlab.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
				},
			},
			shouldNotErr,
			shouldNotErr,
			noAssertions,
		},
		{
			"no windows build",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test"},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_linux_amd64.tar.gz", Goos: "linux", Goarch: "amd64"},
				{Name: "foo_1.0.1_linux_386.tar.gz", Goos: "linux", Goarch: "386"},
			},
			shouldErr("scoop requires a windows build"),
			shouldNotErr,
			noAssertions,
		},
		{
			"is draft",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							Draft: true,
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("release is marked as draft"),
			noAssertions,
		},
		{
			"is prerelease and skip upload set to auto",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1-pre.1",
					},
					Semver: context.Semver{
						Major:      1,
						Minor:      0,
						Patch:      1,
						Prerelease: "-pre.1",
					},
					Version:   "1.0.1-pre.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							SkipUpload: "auto",
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1-pre.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("release is prerelease"),
			noAssertions,
		},
		{
			"skip upload set to true",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							SkipUpload: "true",
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1-pre.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("scoop.skip_upload is true"),
			noAssertions,
		},
		{
			"release is disabled",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							Disable: true,
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("release is disabled"),
			noAssertions,
		},
		{
			"no archive",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "binary"},
						},
						Release: config.Release{
							Draft: true,
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("archive format is binary"),
			shouldNotErr,
			noAssertions,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx
			for _, a := range tt.artifacts {
				ctx.Artifacts.Add(a)
			}
			require.NoError(t, Pipe{}.Default(ctx))

			tt.assertRunError(t, doRun(ctx, tt.args.client))
			tt.assertPublishError(t, doPublish(ctx, tt.args.client))
			tt.assert(t, tt.args)
		})
	}
}

func Test_buildManifest(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	tests := []struct {
		desc string
		ctx  *context.Context
	}{
		{
			"common",
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description: "A run pipe test formula",
						Homepage:    "https://github.com/goreleaser",
						Persist:     []string{"data", "config", "test.ini"},
					},
				},
			},
		},
		{
			"pre-post-install",
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description: "A run pipe test formula",
						Homepage:    "https://github.com/goreleaser",
						Persist:     []string{"data", "config", "test.ini"},
						PreInstall:  []string{"Write-Host 'Running preinstall command'"},
						PostInstall: []string{"Write-Host 'Running postinstall command'"},
					},
				},
			},
		},
		{
			"url template",
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Builds: []config.Build{
						{Binary: "test"},
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description:           "A run pipe test formula",
						Homepage:              "https://github.com/goreleaser",
						URLTemplate:           "http://github.mycompany.com/foo/bar/{{ .Tag }}/{{ .ArtifactName }}",
						CommitMessageTemplate: "chore(scoop): update {{ .ProjectName }} version {{ .Tag }}",
						Persist:               []string{"data.cfg", "etc"},
					},
				},
			},
		},
		{
			"gitlab url template",
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitLab,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitLabURLs: config.GitLabURLs{
						Download: "https://gitlab.com",
					},
					Builds: []config.Build{
						{Binary: "test"},
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description:           "A run pipe test formula",
						Homepage:              "https://gitlab.com/goreleaser",
						URLTemplate:           "http://gitlab.mycompany.com/foo/bar/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}",
						CommitMessageTemplate: "chore(scoop): update {{ .ProjectName }} version {{ .Tag }}",
						Persist:               []string{"data.cfg", "etc"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := tt.ctx
			err := Pipe{}.Default(ctx)
			require.NoError(t, err)

			cl, err := client.New(ctx)
			require.NoError(t, err)

			mf, err := dataFor(ctx, cl, []*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"Builds": []*artifact.Artifact{
							{
								Name: "foo.exe",
							},
							{
								Name: "bar.exe",
							},
						},
					},
				},
				{
					Name:   "foo_1.0.1_windows_arm.tar.gz",
					Goos:   "windows",
					Goarch: "arm",
					Path:   file,
					Extra: map[string]interface{}{
						"Builds": []*artifact.Artifact{
							{
								Name: "foo.exe",
							},
							{
								Name: "bar.exe",
							},
						},
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"Builds": []*artifact.Artifact{
							{
								Name: "foo.exe",
							},
							{
								Name: "bar.exe",
							},
						},
					},
				},
			})
			require.NoError(t, err)

			out, err := doBuildManifest(mf)
			require.NoError(t, err)

			golden.RequireEqualJSON(t, out.Bytes())
		})
	}
}

func getScoopPipeSkipCtx(folder string) (*context.Context, string) {
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Config: config.Project{
			Archives: []config.Archive{
				{Format: "tar.gz"},
			},
			Builds: []config.Build{
				{Binary: "test"},
			},
			Dist:        folder,
			ProjectName: "run-pipe",
			Scoop: config.Scoop{
				Bucket: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
				Description: "A run pipe test formula",
				Homepage:    "https://github.com/goreleaser",
				Name:        "run-pipe",
			},
		},
	}

	path := filepath.Join(folder, "bin.tar.gz")

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "windows",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			"ID":     "foo",
			"Format": "tar.gz",
		},
	})

	return ctx, path
}

func TestRunPipeScoopWithSkipUpload(t *testing.T) {
	folder := t.TempDir()
	ctx, path := getScoopPipeSkipCtx(folder)
	ctx.Config.Scoop.SkipUpload = "true"

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cli := &DummyClient{}
	require.NoError(t, doRun(ctx, cli))
	require.EqualError(t, doPublish(ctx, cli), `scoop.skip_upload is true`)

	distFile := filepath.Join(folder, ctx.Config.Scoop.Name+".json")
	_, err = os.Stat(distFile)
	require.NoError(t, err, "file should exist: "+distFile)
}

func TestWrapInDirectory(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))
	ctx := &context.Context{
		TokenType: context.TokenTypeGitLab,
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Config: config.Project{
			GitLabURLs: config.GitLabURLs{
				Download: "https://gitlab.com",
			},
			Builds: []config.Build{
				{Binary: "test"},
			},
			Dist:        ".",
			ProjectName: "run-pipe",
			Archives: []config.Archive{
				{Format: "tar.gz", WrapInDirectory: "true"},
			},
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
			Scoop: config.Scoop{
				Bucket: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
				Description:           "A run pipe test formula",
				Homepage:              "https://gitlab.com/goreleaser",
				URLTemplate:           "http://gitlab.mycompany.com/foo/bar/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}",
				CommitMessageTemplate: "chore(scoop): update {{ .ProjectName }} version {{ .Tag }}",
				Persist:               []string{"data.cfg", "etc"},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	cl, err := client.New(ctx)
	require.NoError(t, err)
	mf, err := dataFor(ctx, cl, []*artifact.Artifact{
		{
			Name:   "foo_1.0.1_windows_amd64.tar.gz",
			Goos:   "windows",
			Goarch: "amd64",
			Path:   file,
			Extra: map[string]interface{}{
				"WrappedIn": "foo_1.0.1_windows_amd64",
				"Builds": []*artifact.Artifact{
					{
						Name: "foo.exe",
					},
					{
						Name: "bar.exe",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := doBuildManifest(mf)
	require.NoError(t, err)
	golden.RequireEqualJSON(t, out.Bytes())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Scoop: config.Scoop{
				Bucket: config.RepoRef{
					Name: "a",
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

type DummyClient struct {
	CreatedFile bool
	Content     string
	Path        string
}

func (dc *DummyClient) CloseMilestone(ctx *context.Context, repo client.Repo, title string) error {
	return nil
}

func (dc *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return
}

func (dc *DummyClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return "", nil
}

func (dc *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo client.Repo, content []byte, path, msg string) (err error) {
	dc.CreatedFile = true
	dc.Content = string(content)
	dc.Path = path
	return
}

func (dc *DummyClient) GetDefaultBranch(ctx *context.Context, repo client.Repo) (string, error) {
	return "", errors.New("DummyClient does not yet implement GetDefaultBranch")
}

func (dc *DummyClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error) {
	return
}
