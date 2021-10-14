package blob

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	listener.Close()
	listen = listener.Addr().String()

	cleanup, err := start(listen)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	prepareEnv()

	code := m.Run()
	if err := cleanup(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(code)
}

func TestMinioUpload(t *testing.T) {
	name := "basic"
	folder := t.TempDir()
	srcpath := filepath.Join(folder, "source.tar.gz")
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	checkpath := filepath.Join(folder, "check.txt")
	require.NoError(t, os.WriteFile(checkpath, []byte("fake checksums"), 0o744))
	require.NoError(t, os.WriteFile(srcpath, []byte("fake\nsrc"), 0o744))
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider: "s3",
				Bucket:   name,
				Region:   "us-east",
				Endpoint: "http://" + listen,
				IDs:      []string{"foo", "bar"},
				ExtraFiles: []config.ExtraFile{
					{
						Glob: "./testdata/*.golden",
					},
				},
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Checksum,
		Name: "checksum.txt",
		Path: checkpath,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: "source.tar.gz",
		Path: srcpath,
		Extra: map[string]interface{}{
			artifact.ExtraFormat: "tar.gz",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
		Extra: map[string]interface{}{
			artifact.ExtraID: "bar",
		},
	})

	setupBucket(t, name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))

	require.Subset(t, getFiles(t, ctx, ctx.Config.Blobs[0]), []string{
		"testupload/v1.0.0/bin.deb",
		"testupload/v1.0.0/bin.tar.gz",
		"testupload/v1.0.0/checksum.txt",
		"testupload/v1.0.0/source.tar.gz",
		"testupload/v1.0.0/file.golden",
	})
}

func TestMinioUploadCustomBucketID(t *testing.T) {
	name := "fromenv"
	folder := t.TempDir()
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	// Set custom BUCKET_ID env variable.
	require.NoError(t, os.Setenv("BUCKET_ID", name))
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider: "s3",
				Bucket:   "{{.Env.BUCKET_ID}}",
				Endpoint: "http://" + listen,
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
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

	setupBucket(t, name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadRootFolder(t *testing.T) {
	name := "rootdir"
	folder := t.TempDir()
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider: "s3",
				Bucket:   name,
				Folder:   "/",
				Endpoint: "http://" + listen,
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
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

	setupBucket(t, name)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadInvalidCustomBucketID(t *testing.T) {
	folder := t.TempDir()
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Provider: "s3",
				Bucket:   "{{.Bad}}",
				Endpoint: "http://" + listen,
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.1.0"}
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

func prepareEnv() {
	os.Setenv("AWS_ACCESS_KEY_ID", minioUser)
	os.Setenv("AWS_SECRET_ACCESS_KEY", minioPwd)
	os.Setenv("AWS_REGION", "us-east-1")
}

func start(listen string) (func() error, error) {
	data := filepath.Join(os.TempDir(), containerName)

	fn := func() error {
		if out, err := exec.Command("docker", "stop", containerName).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to stop minio: %s: %w", out, err)
		}
		if err := os.RemoveAll(data); err != nil {
			log.Println("failed to remove", data)
		}
		return nil
	}

	// stop container if it is running (likely from previous test)
	_, _ = exec.Command("docker", "stop", containerName).CombinedOutput()

	if out, err := exec.Command(
		"docker", "run", "-d", "--rm",
		"-v", data+":/data",
		"--name", containerName,
		"-p", listen+":9000",
		"-e", "MINIO_ROOT_USER="+minioUser,
		"-e", "MINIO_ROOT_PASSWORD="+minioPwd,
		"--health-interval", "1s",
		"--health-cmd=curl --silent --fail http://localhost:9000/minio/health/ready || exit 1",
		"minio/minio",
		"server", "/data", "--console-address", ":9001",
	).CombinedOutput(); err != nil {
		return fn, fmt.Errorf("failed to start minio: %s: %w", out, err)
	}

	for range time.Tick(time.Second) {
		out, err := exec.Command("docker", "inspect", "--format='{{json .State.Health}}'", containerName).CombinedOutput()
		if err != nil {
			return fn, fmt.Errorf("failed to check minio status: %s: %w", string(out), err)
		}
		if strings.Contains(string(out), `"Status":"healthy"`) {
			log.Println("minio is healthy")
			break
		}
		log.Println("waiting for minio to be healthy")
	}

	return fn, nil
}

func setupBucket(tb testing.TB, name string) {
	tb.Helper()
	mc(tb, "mc mb local/"+name)
	tb.Cleanup(func() {
		mc(tb, "mc rb --force local/"+name)
	})
}

func mc(tb testing.TB, cmd string) {
	tb.Helper()

	if out, err := exec.Command(
		"docker", "run", "--rm",
		"--link", containerName,
		"--entrypoint", "sh",
		"minio/mc",
		"-c", fmt.Sprintf(
			"mc config host add local http://%s:9000 %s %s; %s",
			containerName, minioUser, minioPwd, cmd,
		),
	).CombinedOutput(); err != nil {
		tb.Fatalf("failed to create test bucket: %s", string(out))
	}
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
