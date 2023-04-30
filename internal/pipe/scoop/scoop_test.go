package scoop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
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

	ctx := testctx.NewWithCfg(
		config.Project{ProjectName: "barr"},
		testctx.GitHubTokenType,
	)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Scoop.Name)
	require.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Scoop.CommitMessageTemplate)
}

func Test_doRun(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	type args struct {
		ctx    *context.Context
		client *client.Mock
	}

	type asserter func(testing.TB, args)
	type errChecker func(testing.TB, error)
	shouldErr := func(msg string) errChecker {
		return func(tb testing.TB, err error) {
			tb.Helper()
			require.Error(tb, err)
			require.EqualError(tb, err, msg)
		}
	}
	noAssertions := func(tb testing.TB, _ args) {
		tb.Helper()
	}
	shouldNotErr := func(tb testing.TB, err error) {
		tb.Helper()
		require.NoError(tb, err)
	}

	tests := []struct {
		name               string
		args               args
		artifacts          []artifact.Artifact
		assertRunError     errChecker
		assertPublishError errChecker
		assert             asserter
	}{
		{
			"valid public github",
			args{
				testctx.NewWithCfg(
					config.Project{
						Dist:        t.TempDir(),
						ProjectName: "run-pipe",
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
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldNotErr,
			func(tb testing.TB, a args) {
				tb.Helper()
				require.Equal(tb, "scoops/run-pipe.json", a.client.Path)
			},
		},
		{
			"git_remote",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "git-run-pipe",
						Dist:        t.TempDir(),
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Name:   "test",
								Branch: "main",
								Git: config.GitRepoRef{
									URL:        testlib.GitMakeBareRepository(t),
									PrivateKey: testlib.MakeNewSSHKey(t, keygen.Ed25519, ""),
								},
							},
							Folder:      "scoops",
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldNotErr,
			func(tb testing.TB, a args) {
				tb.Helper()
				content := testlib.CatFileFromBareRepository(
					tb,
					a.ctx.Config.Scoop.Bucket.Git.URL,
					"scoops/git-run-pipe.json",
				)
				golden.RequireEqualJSON(tb, content)
			},
		},
		{
			"wrap in directory",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{
					Name:    "foo_1.0.1_windows_amd64.tar.gz",
					Goos:    "windows",
					Goarch:  "amd64",
					Goamd64: "v1",
					Path:    file,
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
				testctx.NewWithCfg(
					config.Project{
						GitHubURLs:  config.GitHubURLs{Download: "https://api.custom.github.enterprise.com"},
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldNotErr,
			func(tb testing.TB, a args) {
				tb.Helper()
				require.Equal(tb, "run-pipe.json", a.client.Path)
			},
		},
		{
			"valid public gitlab",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://gitlab.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{
					Name:    "foo_1.0.1_windows_amd64.tar.gz",
					Goos:    "windows",
					Goarch:  "amd64",
					Goamd64: "v1",
					Path:    file,
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
				testctx.NewWithCfg(
					config.Project{
						GitHubURLs:  config.GitHubURLs{Download: "https://api.custom.gitlab.enterprise.com"},
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://gitlab.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{
					Name:    "foo_1.0.1_windows_amd64.tar.gz",
					Goos:    "windows",
					Goarch:  "amd64",
					Goamd64: "v1",
					Path:    file,
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
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{},
			shouldErr(ErrNoWindows{"v1"}.Error()),
			shouldNotErr,
			noAssertions,
		},
		{
			"is prerelease and skip upload set to auto",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
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
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1-pre.1"),
					testctx.WithVersion("1.0.1-pre.1"),
					testctx.WithSemver(1, 0, 0, "pre.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
				{Name: "foo_1.0.1-pre.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("release is prerelease"),
			noAssertions,
		},
		{
			"skip upload set to true",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
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
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
				{Name: "foo_1.0.1-pre.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("scoop.skip_upload is true"),
			noAssertions,
		},
		{
			"release is disabled",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
						Release: config.Release{
							Disable: "true",
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
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
			shouldErr("release is disabled"),
			noAssertions,
		},
		{
			"no archive",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{},
			shouldErr(ErrNoWindows{"v1"}.Error()),
			shouldNotErr,
			noAssertions,
		},
		{
			"invalid ref tmpl",
			args{
				testctx.NewWithCfg(
					config.Project{
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "{{ .Env.aaaaaa }}",
								Name:  "test",
							},
							Folder:      "scoops",
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
			},
			shouldNotErr,
			testlib.RequireTemplateError,
			noAssertions,
		},
		{
			"ref templ",
			args{
				testctx.NewWithCfg(
					config.Project{
						Env:         []string{"FOO=test", "BRANCH=main"},
						ProjectName: "run-pipe",
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner:  "{{ .Env.FOO }}",
								Name:   "{{ .Env.FOO }}",
								Branch: "{{ .Env.BRANCH }}",
							},
							Folder:      "scoops",
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					testctx.GitHubTokenType,
					testctx.WithCurrentTag("v1.0.1"),
					testctx.WithVersion("1.0.1"),
				),
				client.NewMock(),
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Goamd64: "v1", Path: file},
			},
			shouldNotErr,
			shouldNotErr,
			func(tb testing.TB, a args) {
				tb.Helper()
				require.Equal(tb, "scoops/run-pipe.json", a.client.Path)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx
			for _, a := range tt.artifacts {
				a.Type = artifact.UploadableArchive
				ctx.Artifacts.Add(&a)
			}
			require.NoError(t, Pipe{}.Default(ctx))

			tt.assertRunError(t, doRun(ctx, tt.args.client))
			tt.assertPublishError(t, doPublish(ctx, tt.args.client))
			tt.assert(t, tt.args)
		})
	}
}

func TestRunPipePullRequest(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Scoop: config.Scoop{
				Name:        "foo",
				Homepage:    "https://goreleaser.com",
				Description: "Fake desc",
				Bucket: config.RepoRef{
					Owner:  "foo",
					Name:   "bar",
					Branch: "update-{{.Version}}",
					PullRequest: config.PullRequest{
						Enabled: true,
					},
				},
			},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
	)
	path := filepath.Join(folder, "dist/foo_windows_amd64/foo.exe")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_windows_amd64.tar.gz",
		Path:   path,
		Goos:   "windows",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
			artifact.ExtraBinary: "foo",
		},
	})

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	client := client.NewMock()
	require.NoError(t, doRun(ctx, client))
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedFile)
	require.True(t, client.OpenedPullRequest)
	golden.RequireEqualJSON(t, []byte(client.Content))
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
			testctx.NewWithCfg(
				config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					ProjectName: "run-pipe",
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
				testctx.GitHubTokenType,
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithVersion("1.0.1"),
			),
		},
		{
			"pre-post-install",
			testctx.NewWithCfg(
				config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					ProjectName: "run-pipe",
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
				testctx.GitHubTokenType,
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithVersion("1.0.1"),
			),
		},
		{
			"url template",
			testctx.NewWithCfg(
				config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					ProjectName: "run-pipe",
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
				testctx.WithCurrentTag("v1.0.1"),
				testctx.GitHubTokenType,
				testctx.WithVersion("1.0.1"),
			),
		},
		{
			"gitlab url template",
			testctx.NewWithCfg(
				config.Project{
					GitLabURLs: config.GitLabURLs{
						Download: "https://gitlab.com",
					},
					ProjectName: "run-pipe",
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
				testctx.GitHubTokenType,
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithVersion("1.0.1"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := tt.ctx
			err := Pipe{}.Default(ctx)
			require.NoError(t, err)

			cl, err := client.New(ctx)
			require.NoError(t, err)
			require.NoError(t, Pipe{}.Default(ctx))

			mf, err := dataFor(ctx, cl, []*artifact.Artifact{
				{
					Name:    "foo_1.0.1_windows_amd64.tar.gz",
					Goos:    "windows",
					Goarch:  "amd64",
					Goamd64: "v1",
					Path:    file,
					Extra: map[string]interface{}{
						artifact.ExtraBuilds: []*artifact.Artifact{
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
						artifact.ExtraBuilds: []*artifact.Artifact{
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
						artifact.ExtraBuilds: []*artifact.Artifact{
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
	ctx := testctx.NewWithCfg(
		config.Project{
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
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)

	path := filepath.Join(folder, "bin.tar.gz")

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin.tar.gz",
		Path:    path,
		Goos:    "windows",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "ignored.tar.gz",
		Path:    path,
		Goos:    "windows",
		Goarch:  "amd64",
		Goamd64: "v3",
		Type:    artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
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

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
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

	ctx := testctx.NewWithCfg(
		config.Project{
			GitLabURLs: config.GitLabURLs{
				Download: "https://gitlab.com",
			},
			ProjectName: "run-pipe",
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
		testctx.GitHubTokenType,
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)

	require.NoError(t, Pipe{}.Default(ctx))
	cl, err := client.New(ctx)
	require.NoError(t, err)
	mf, err := dataFor(ctx, cl, []*artifact.Artifact{
		{
			Name:    "foo_1.0.1_windows_amd64.tar.gz",
			Goos:    "windows",
			Goarch:  "amd64",
			Goamd64: "v1",
			Path:    file,
			Extra: map[string]interface{}{
				artifact.ExtraWrappedIn: "foo_1.0.1_windows_amd64",
				artifact.ExtraBuilds: []*artifact.Artifact{
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
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Scoop: config.Scoop{
				Bucket: config.RepoRef{
					Name: "a",
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
