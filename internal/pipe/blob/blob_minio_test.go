package blob

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
)

const (
	minioUser     = "minio"
	minioPwd      = "miniostorage"
	containerName = "goreleaserTestMinio"
)

var listen string

func TestMain(m *testing.M) {
	if !testlib.InPath("docker") || testlib.IsWindows() || !testlib.IsDockerRunning() {
		// there's no minio windows image
		m.Run()
		return
	}
	prepareEnv()

	requireNoErr := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	pool := testlib.MustDockerPool(log.Default())
	testlib.MustKillContainer(log.Default(), containerName)

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       containerName,
		Repository: "minio/minio",
		Tag:        "latest",
		Env: []string{
			"MINIO_ROOT_USER=" + minioUser,
			"MINIO_ROOT_PASSWORD=" + minioPwd,
		},
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Cmd:          []string{"server", "/data", "--console-address", ":9001"},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	requireNoErr(err)
	requireNoErr(pool.Retry(func() error {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/minio/health/ready", resource.GetPort("9000/tcp")))
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}))
	listen = "localhost:" + resource.GetPort("9000/tcp")

	m.Run()

	requireNoErr(pool.Purge(resource))
}

func TestMinioUpload(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "minio image not available for windows")
	name := "basic"
	directory := t.TempDir()
	srcpath := filepath.Join(directory, "source.tar.gz")
	tgzpath := filepath.Join(directory, "bin.tar.gz")
	debpath := filepath.Join(directory, "bin.deb")
	checkpath := filepath.Join(directory, "check.txt")
	metapath := filepath.Join(directory, "metadata.json")
	sigpath := filepath.Join(directory, "f.sig")
	certpath := filepath.Join(directory, "f.pem")
	require.NoError(t, os.WriteFile(checkpath, []byte("fake checksums"), 0o744))
	require.NoError(t, os.WriteFile(metapath, []byte(`{"fake":true}`), 0o744))
	require.NoError(t, os.WriteFile(srcpath, []byte("fake\nsrc"), 0o744))
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	require.NoError(t, os.WriteFile(sigpath, []byte("fake\nsig"), 0o744))
	require.NoError(t, os.WriteFile(certpath, []byte("fake\ncert"), 0o744))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        directory,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider:           "s3",
				Bucket:             name,
				Region:             "us-east",
				Endpoint:           "http://" + listen,
				IDs:                []string{"foo", "bar"},
				CacheControl:       []string{"max-age=9999"},
				ContentDisposition: "inline",
				IncludeMeta:        true,
				ExtraFiles: []config.ExtraFile{
					{
						Glob: "./testdata/*.golden",
					},
				},
			},
		},
	}, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Metadata,
		Name: "metadata.json",
		Path: metapath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Checksum,
		Name: "checksum.txt",
		Path: checkpath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Signature,
		Name: "checksum.txt.sig",
		Path: sigpath,
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Certificate,
		Name: "checksum.pem",
		Path: certpath,
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: "source.tar.gz",
		Path: srcpath,
		Extra: map[string]any{
			artifact.ExtraFormat: "tar.gz",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})

	setupBucket(t, testlib.MustDockerPool(t), name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))

	require.ElementsMatch(t, getFiles(t, ctx, ctx.Config.Blobs[0]), []string{
		"testupload/v1.0.0/bin.deb",
		"testupload/v1.0.0/bin.tar.gz",
		"testupload/v1.0.0/metadata.json",
		"testupload/v1.0.0/checksum.txt",
		"testupload/v1.0.0/checksum.txt.sig",
		"testupload/v1.0.0/checksum.pem",
		"testupload/v1.0.0/source.tar.gz",
		"testupload/v1.0.0/file.golden",
	})
}

func TestMinioUploadCustomBucketID(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "minio image not available for windows")
	name := "fromenv"
	directory := t.TempDir()
	tgzpath := filepath.Join(directory, "bin.tar.gz")
	debpath := filepath.Join(directory, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	// Set custom BUCKET_ID env variable.
	t.Setenv("BUCKET_ID", name)
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        directory,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider: "s3",
				Bucket:   "{{.Env.BUCKET_ID}}",
				Endpoint: "http://" + listen,
			},
		},
	}, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	setupBucket(t, testlib.MustDockerPool(t), name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadExtraFilesOnly(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "minio image not available for windows")
	name := "only-extra-files"
	directory := t.TempDir()
	tgzpath := filepath.Join(directory, "bin.tar.gz")
	debpath := filepath.Join(directory, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        directory,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider:       "s3",
				Bucket:         name,
				Endpoint:       "http://" + listen,
				IncludeMeta:    true,
				ExtraFilesOnly: true,
				ExtraFiles: []config.ExtraFile{
					{
						Glob: "./testdata/*.golden",
					},
				},
			},
		},
	}, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	setupBucket(t, testlib.MustDockerPool(t), name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))

	require.ElementsMatch(t, getFiles(t, ctx, ctx.Config.Blobs[0]), []string{
		"testupload/v1.0.0/file.golden",
	})
}

func TestMinioUploadRootDirectory(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "minio image not available for windows")
	name := "rootdir"
	directory := t.TempDir()
	tgzpath := filepath.Join(directory, "bin.tar.gz")
	debpath := filepath.Join(directory, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        directory,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider:  "s3",
				Bucket:    name,
				Directory: "/",
				Endpoint:  "http://" + listen,
			},
		},
	}, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	setupBucket(t, testlib.MustDockerPool(t), name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadInvalidCustomBucketID(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "minio image not available for windows")
	directory := t.TempDir()
	tgzpath := filepath.Join(directory, "bin.tar.gz")
	debpath := filepath.Join(directory, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        directory,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider: "s3",
				Bucket:   "{{.Bad}}",
				Endpoint: "http://" + listen,
			},
		},
	}, testctx.WithCurrentTag("v1.1.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Error(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadSkip(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "minio image not available for windows")
	name := "basic"
	directory := t.TempDir()
	debpath := filepath.Join(directory, "bin.deb")
	tgzpath := filepath.Join(directory, "bin.tar.gz")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))

	buildCtx := func(uploadID string) *context.Context {
		ctx := testctx.NewWithCfg(
			config.Project{
				Dist:        directory,
				ProjectName: "testupload",
				Blobs: []config.Blob{
					{
						Provider: "s3",
						Bucket:   name,
						Region:   "us-east",
						Endpoint: "http://" + listen,
						IDs:      []string{"foo"},
						Disable:  `{{ eq .Env.UPLOAD_ID "foo" }}`,
					},
					{
						Provider: "s3",
						Bucket:   name,
						Region:   "us-east",
						Endpoint: "http://" + listen,
						Disable:  `{{ eq .Env.UPLOAD_ID "bar" }}`,
						IDs:      []string{"bar"},
					},
				},
			},
			testctx.WithCurrentTag("v1.0.0"),
			testctx.WithEnv(map[string]string{
				"UPLOAD_ID": uploadID,
			}),
		)
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableArchive,
			Name: "bin.tar.gz",
			Path: tgzpath,
			Extra: map[string]any{
				artifact.ExtraID: "foo",
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.LinuxPackage,
			Name: "bin.deb",
			Path: debpath,
			Extra: map[string]any{
				artifact.ExtraID: "bar",
			},
		})
		return ctx
	}

	setupBucket(t, testlib.MustDockerPool(t), name)

	t.Run("upload only foo", func(t *testing.T) {
		ctx := buildCtx("foo")
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
		require.Subset(t, getFiles(t, ctx, ctx.Config.Blobs[0]), []string{
			"testupload/v1.0.0/bin.deb",
		})
	})

	t.Run("upload only bar", func(t *testing.T) {
		ctx := buildCtx("bar")
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
		require.Subset(t, getFiles(t, ctx, ctx.Config.Blobs[0]), []string{
			"testupload/v1.0.0/bin.tar.gz",
		})
	})

	t.Run("invalid tmpl", func(t *testing.T) {
		ctx := buildCtx("none")
		ctx.Config.Blobs = []config.Blob{{
			Provider: "s3",
			Bucket:   name,
			Endpoint: "http://" + listen,
			Disable:  `{{ .Env.NOME }}`,
		}}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
	})
}

func prepareEnv() {
	os.Setenv("AWS_ACCESS_KEY_ID", minioUser)
	os.Setenv("AWS_SECRET_ACCESS_KEY", minioPwd)
	os.Setenv("AWS_REGION", "us-east-1")
}

func setupBucket(tb testing.TB, pool *dockertest.Pool, name string) {
	tb.Helper()

	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "minio/mc",
		Links:      []string{containerName},
		Env:        []string{fmt.Sprintf("MC_HOST_local=http://%s:%s@%s:9000", minioUser, minioPwd, containerName)},
		Cmd:        []string{"mb", "local/" + name},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	require.NoError(tb, err)
	require.NoError(tb, pool.Retry(func() error {
		if _, ok := pool.ContainerByName(res.Container.Name); ok {
			return fmt.Errorf("still running: %s", res.Container.Name)
		}
		return nil
	}))
}

func getFiles(t *testing.T, ctx *context.Context, cfg config.Blob) []string {
	t.Helper()
	url, err := urlFor(ctx, cfg)
	require.NoError(t, err)
	conn, err := blob.OpenBucket(ctx, url)
	require.NoError(t, err)
	defer conn.Close()
	iter := conn.List(nil)
	var files []string
	for {
		file, err := iter.Next(ctx)
		if err != nil && err == io.EOF {
			break
		}
		require.NoError(t, err)
		files = append(files, file.Key)
	}
	return files
}
