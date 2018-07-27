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
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoS3(t *testing.T) {
	assert.NoError(t, Pipe{}.Run(context.New(config.Project{})))
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

func TestSkipPublish(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	require.NoError(t, err)
	artifactPath := filepath.Join(folder, "foo.tar.gz")
	require.NoError(t, ioutil.WriteFile(artifactPath, []byte("fake\ntargz"), 0744))
	var ctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		S3: []config.S3{
			{
				Bucket:   "test",
				Endpoint: "http://fake.s3.example",
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "foo.tar.gz",
		Path: artifactPath,
	})
	ctx.SkipPublish = true
	require.NoError(t, Pipe{}.Default(ctx))
	err = Pipe{}.Run(ctx)
	assert.True(t, pipeline.IsSkip(err))
	assert.EqualError(t, err, pipeline.ErrSkipPublishEnabled.Error())
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
	assert.NoError(t, Pipe{}.Run(ctx))
}

func setCredentials(t *testing.T) {
	// this comes from the testdata/config/config.json file - not real aws keys
	os.Setenv("AWS_ACCESS_KEY_ID", "IWA0WZQ1QJ2I8I1ALW64")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "zcK4QQegvYwVGJaBm2E6k20WRLIQZqHOKOPP2npT")
	os.Setenv("AWS_REGION", "us-east-1")
}

func start(t *testing.T) {
	dir, err := os.Getwd()
	assert.NoError(t, err)
	log.Info("wd: " + dir)
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
