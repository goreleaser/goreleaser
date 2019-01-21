package s3

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoS3(t *testing.T) {
	testlib.AssertSkipped(t, Pipe{}.Publish(context.New(config.Project{})))
}

func TestDefaultsNoS3(t *testing.T) {
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		S3: []config.S3{
			{},
		},
	})
	assert.NoError(Pipe{}.Default(ctx))
	assert.Equal([]config.S3{{}}, ctx.Config.S3)
}

func TestDefaults(t *testing.T) {
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		S3: []config.S3{
			{
				Bucket: "foo",
			},
		},
	})
	assert.NoError(Pipe{}.Default(ctx))
	assert.Equal([]config.S3{{
		Bucket: "foo",
		Region: "us-east-1",
		Folder: "{{ .ProjectName }}/{{ .Tag }}",
		ACL:    "private",
	}}, ctx.Config.S3)
}

func TestUpload(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	assert.NoError(t, ioutil.WriteFile(tgzpath, []byte("fake\ntargz"), 0744))
	assert.NoError(t, ioutil.WriteFile(debpath, []byte("fake\ndeb"), 0744))
	var ctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		S3: []config.S3{
			{
				Bucket:   "test",
				Endpoint: "http://localhost:9000",
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})
	start(t)
	defer stop(t)
	setCredentials(t)
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.NoError(t, Pipe{}.Publish(ctx))
}

func TestUploadCustomBucketID(t *testing.T) {
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
		S3: []config.S3{
			{
				Bucket:   "{{.Env.BUCKET_ID}}",
				Endpoint: "http://localhost:9000",
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})
	start(t)
	defer stop(t)
	setCredentials(t)
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.NoError(t, Pipe{}.Publish(ctx))
}

func TestUploadInvalidCustomBucketID(t *testing.T) {
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
		S3: []config.S3{
			{
				Bucket:   "{{.Bad}}",
				Endpoint: "http://localhost:9000",
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})
	start(t)
	defer stop(t)
	setCredentials(t)
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Error(t, Pipe{}.Publish(ctx))
}

func setCredentials(t *testing.T) {
	// this comes from the testdata/config/config.json file - not real aws keys
	os.Setenv("AWS_ACCESS_KEY_ID", "WPXKJC7CZQCFPKY5727N")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "eHCSajxLvl94l36gIMlzZ/oW2O0rYYK+cVn5jNT2")
	os.Setenv("AWS_REGION", "us-east-1")
}

func start(t *testing.T) {
	dir, err := os.Getwd()
	assert.NoError(t, err)
	if out, err := exec.Command(
		"docker", "run", "-d", "--rm",
		"--name", "minio",
		"-p", "9000:9000",
		"-v", dir+"/testdata/data:/data",
		"-v", dir+"/testdata/config:/root/.minio",
		"minio/minio:RELEASE.2018-06-09T03-43-35Z",
		"server", "/data",
	).CombinedOutput(); err != nil {
		log.WithError(err).Errorf("failed to start minio: %s", string(out))
		t.FailNow()
	}

	for range time.Tick(time.Second) {
		out, err := exec.Command("docker", "inspect", "--format='{{json .State.Health}}'", "minio").CombinedOutput()
		if err != nil {
			log.WithError(err).Errorf("failed to check minio status: %s", string(out))
			t.FailNow()
		}
		if strings.Contains(string(out), `"Status":"healthy"`) {
			log.Info("minio is healthy")
			break
		}
		log.Info("waiting for minio to be healthy")
	}
}

func stop(t *testing.T) {
	if out, err := exec.Command("docker", "stop", "minio").CombinedOutput(); err != nil {
		log.WithError(err).Errorf("failed to stop minio: %s", string(out))
		t.FailNow()
	}
}
