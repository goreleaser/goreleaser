package blob

// this is pretty much copied from the s3 pipe to ensure both work the same way
// only differences are that it sets `blobs` instead of `s3` on test cases and
// the test setup and teardown

import (
	"io"
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

func TestMinioUpload(t *testing.T) {
	listen := randomListen(t)
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
				Bucket:   "test",
				Region:   "us-east",
				Endpoint: "http://" + listen,
				IDs:      []string{"foo", "bar"},
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
			"Format": "tar.gz",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
		Extra: map[string]interface{}{
			"ID": "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
		Extra: map[string]interface{}{
			"ID": "bar",
		},
	})
	name := "test_upload"
	start(t, name, listen)
	prepareEnv()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))

	require.Subset(t, getFiles(t, ctx, ctx.Config.Blobs[0]), []string{
		"testupload/v1.0.0/bin.deb",
		"testupload/v1.0.0/bin.tar.gz",
		"testupload/v1.0.0/checksum.txt",
		"testupload/v1.0.0/source.tar.gz",
	})
}

func TestMinioUploadCustomBucketID(t *testing.T) {
	listen := randomListen(t)
	folder := t.TempDir()
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	require.NoError(t, os.WriteFile(tgzpath, []byte("fake\ntargz"), 0o744))
	require.NoError(t, os.WriteFile(debpath, []byte("fake\ndeb"), 0o744))
	// Set custom BUCKET_ID env variable.
	require.NoError(t, os.Setenv("BUCKET_ID", "test"))
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
	name := "custom_bucket_id"
	start(t, name, listen)
	prepareEnv()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadRootFolder(t *testing.T) {
	listen := randomListen(t)
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
				Bucket:   "test",
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
	name := "root_folder"
	start(t, name, listen)
	prepareEnv()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadInvalidCustomBucketID(t *testing.T) {
	listen := randomListen(t)
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
	name := "invalid_bucket_id"
	start(t, name, listen)
	prepareEnv()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Error(t, Pipe{}.Publish(ctx))
}

func randomListen(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	listener.Close()
	return listener.Addr().String()
}

func prepareEnv() {
	os.Setenv("AWS_ACCESS_KEY_ID", "minio")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "miniostorage")
	os.Setenv("AWS_REGION", "us-east-1")
}

func start(tb testing.TB, name, listen string) {
	tb.Helper()
	wd, err := os.Getwd()
	require.NoError(tb, err)

	removeTestData()

	tb.Cleanup(func() {
		if out, err := exec.Command("docker", "stop", name).CombinedOutput(); err != nil {
			tb.Fatalf("failed to stop minio: %s", string(out))
		}
		removeTestData()
	})

	if out, err := exec.Command(
		"docker", "run", "-d", "--rm",
		"-v", filepath.Join(wd, "testdata/data")+":/data",
		"--name", name,
		"-p", listen+":9000",
		"-e", "MINIO_ACCESS_KEY=minio",
		"-e", "MINIO_SECRET_KEY=miniostorage",
		"--health-interval", "1s",
		"--health-cmd=curl --silent --fail http://localhost:9000/minio/health/ready || exit 1",
		"minio/minio",
		"server", "/data",
	).CombinedOutput(); err != nil {
		tb.Fatalf("failed to start minio: %s", string(out))
	}

	for range time.Tick(time.Second) {
		out, err := exec.Command("docker", "inspect", "--format='{{json .State.Health}}'", name).CombinedOutput()
		if err != nil {
			tb.Fatalf("failed to check minio status: %s", string(out))
		}
		if strings.Contains(string(out), `"Status":"healthy"`) {
			tb.Log("minio is healthy")
			break
		}
		tb.Log("waiting for minio to be healthy")
	}
}

func removeTestData() {
	_ = os.RemoveAll("./testdata/data/test/testupload") // dont care if it fails
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
