package blob

// this is pretty much copied from the s3 pipe to ensure both work the same way
// only differences are that it sets `blobs` instead of `s3` on test cases and
// the test setup and teardown

import (
	"io/ioutil"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinioUpload(t *testing.T) {
	var listen = randomListen(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	assert.NoError(t, ioutil.WriteFile(tgzpath, []byte("fake\ntargz"), 0744))
	assert.NoError(t, ioutil.WriteFile(debpath, []byte("fake\ndeb"), 0744))
	var ctx = context.New(config.Project{
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
	var name = "test_upload"
	defer stop(t, name)
	start(t, name, listen)
	prepareEnv(t, listen)
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadCustomBucketID(t *testing.T) {
	var listen = randomListen(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	assert.NoError(t, ioutil.WriteFile(tgzpath, []byte("fake\ntargz"), 0744))
	assert.NoError(t, ioutil.WriteFile(debpath, []byte("fake\ndeb"), 0744))
	// Set custom BUCKET_ID env variable.
	err = os.Setenv("BUCKET_ID", "test")
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
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
	var name = "custom_bucket_id"
	defer stop(t, name)
	start(t, name, listen)
	prepareEnv(t, listen)
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.NoError(t, Pipe{}.Publish(ctx))
}

func TestMinioUploadInvalidCustomBucketID(t *testing.T) {
	var listen = randomListen(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	assert.NoError(t, ioutil.WriteFile(tgzpath, []byte("fake\ntargz"), 0744))
	assert.NoError(t, ioutil.WriteFile(debpath, []byte("fake\ndeb"), 0744))
	// Set custom BUCKET_ID env variable.
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
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
	var name = "invalid_bucket_id"
	defer stop(t, name)
	start(t, name, listen)
	prepareEnv(t, listen)
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Error(t, Pipe{}.Publish(ctx))
}

func randomListen(t *testing.T) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	listener.Close()
	return listener.Addr().String()
}

func prepareEnv(t *testing.T, listen string) {
	os.Setenv("AWS_ACCESS_KEY_ID", "minio")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "miniostorage")
	os.Setenv("AWS_REGION", "us-east-1")
}

func start(t *testing.T, name, listen string) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	removeTestData(t)

	if out, err := exec.Command(
		"docker", "run", "-d", "--rm",
		"-v", filepath.Join(wd, "testdata/data")+":/data",
		"--name", name,
		"-p", listen+":9000",
		"-e", "MINIO_ACCESS_KEY=minio",
		"-e", "MINIO_SECRET_KEY=miniostorage",
		"--health-interval", "1s",
		"minio/minio:RELEASE.2019-05-14T23-57-45Z",
		"server", "/data",
	).CombinedOutput(); err != nil {
		t.Fatalf("failed to start minio: %s", string(out))
	}

	for range time.Tick(time.Second) {
		out, err := exec.Command("docker", "inspect", "--format='{{json .State.Health}}'", name).CombinedOutput()
		if err != nil {
			t.Fatalf("failed to check minio status: %s", string(out))
		}
		if strings.Contains(string(out), `"Status":"healthy"`) {
			t.Log("minio is healthy")
			break
		}
		t.Log("waiting for minio to be healthy")
	}
}

func stop(t *testing.T, name string) {
	if out, err := exec.Command("docker", "stop", name).CombinedOutput(); err != nil {
		t.Fatalf("failed to stop minio: %s", string(out))
	}
	removeTestData(t)
}

func removeTestData(t *testing.T) {
	_ = os.RemoveAll("./testdata/data/test/testupload") // dont care if it fails
}
