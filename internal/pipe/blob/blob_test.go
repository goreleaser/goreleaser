package blob

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoBlob(t *testing.T) {
	testlib.AssertSkipped(t, Pipe{}.Publish(context.New(config.Project{})))
}

func TestDefaultsNoConfig(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{},
		},
	})
	assert.EqualError(Pipe{}.Default(ctx), errorString)
}

func TestDefaultsNoBucket(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Provider: "azblob",
			},
		},
	})
	assert.EqualError(Pipe{}.Default(ctx), errorString)
}

func TestDefaultsNoProvider(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Bucket: "goreleaser-bucket",
			},
		},
	})
	assert.EqualError(Pipe{}.Default(ctx), errorString)
}

func TestDefaults(t *testing.T) {
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "azblob",
			},
		},
	})
	setEnvVariables()
	assert.NoError(Pipe{}.Default(ctx))
	assert.Equal([]config.Blob{{
		Bucket:   "foo",
		Provider: "azblob",
		Folder:   "{{ .ProjectName }}/{{ .Tag }}",
	}}, ctx.Config.Blobs)
}

func TestDefaultsWithProvider(t *testing.T) {
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "azblob",
			},
			{
				Bucket:   "foo",
				Provider: "s3",
			},
			{
				Bucket:   "foo",
				Provider: "gs",
			},
		},
	})

	setEnvVariables()
	assert.Nil(Pipe{}.Default(ctx))
}

func TestDefaultsWithInvalidProvider(t *testing.T) {
	var assert = assert.New(t)

	// This is invalid provider, meaning not registred with GO CDK
	invalidProvider := "bar"
	errorString := fmt.Sprintf("unknown provider [%v],currently supported providers: [azblob, gs, s3]", invalidProvider)
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: invalidProvider,
			},
		},
	})

	setEnvVariables()
	assert.EqualError(Pipe{}.Default(ctx), errorString)
}

func TestDefaultsWithMissingEnv(t *testing.T) {
	var assert = assert.New(t)

	errorString := "missing AZURE_STORAGE_ACCOUNT,AZURE_STORAGE_KEY"
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "azblob",
			},
		},
	})

	os.Unsetenv("AZURE_STORAGE_ACCOUNT")
	os.Unsetenv("AZURE_STORAGE_KEY")

	assert.EqualError(Pipe{}.Default(ctx), errorString)
}

func TestPipe_Publish(t *testing.T) {
	gcloudCredentials, _ := filepath.Abs("./testdata/credentials.json")

	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	assert.NoError(t, ioutil.WriteFile(tgzpath, []byte("fake\ntargz"), 0744))
	assert.NoError(t, ioutil.WriteFile(debpath, []byte("fake\ndeb"), 0744))

	// Azure Blob Context Without ENV
	var azblobctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "azblob",
			},
		},
	})

	azblobctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	azblobctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	azblobctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	// Azure Blob Context Without ENV
	var azblobctx1 = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload1",
		Blobs: []config.Blob{
			{
				Bucket:   "foobar",
				Provider: "azblob",
			},
		},
	})

	azblobctx1.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	azblobctx1.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	azblobctx1.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	// Google Cloud Storage Context
	var gsctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "gs",
			},
		},
	})

	gsctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}

	gsctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	azblobctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	// AWS S3 Context
	var s3ctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "s3",
			},
		},
	})

	s3ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}

	gsctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	s3ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debpath,
	})

	type args struct {
		ctx *context.Context
	}
	tests := []struct {
		name          string
		args          args
		env           map[string]string
		wantErr       bool
		wantErrString string
	}{
		{
			name:          "Azure Blob Bucket test Publish(StorageAccount)",
			args:          args{azblobctx},
			env:           map[string]string{"AZURE_STORAGE_ACCOUNT": "hjsdhjsdhs", "AZURE_STORAGE_KEY": "eHCSajxLvl94l36gIMlzZ/oW2O0rYYK+cVn5jNT2"},
			wantErr:       false,
			wantErrString: "azure storage account you provided is not valid",
		},
		{
			name:          "Google Cloud Storage Bucket Test Publish",
			args:          args{gsctx},
			env:           map[string]string{"GOOGLE_APPLICATION_CREDENTIALS": gcloudCredentials},
			wantErr:       false,
			wantErrString: "google app credentials you provided is not valid",
		},
		{
			name:          "AWS S3 Bucket Test Publish",
			args:          args{s3ctx},
			env:           map[string]string{"AWS_ACCESS_KEY": "WPXKJC7CZQCFPKY5727N", "AWS_SECRET_KEY": "eHCSajxLvl94l36gIMlzZ/oW2O0rYYK+cVn5jNT2", "AWS_REGION": "us-east-1"},
			wantErr:       false,
			wantErrString: "aws access key id you provided does not exist in our records",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pipe{}
			setEnv(tt.env)

			if err := p.Publish(tt.args.ctx); (err != nil) != tt.wantErr {
				if err.Error() != tt.wantErrString {
					t.Errorf("Pipe.Publish() error = %v, wantErr %v", err, tt.wantErrString)
				}
			}
			os.Clearenv()
		})
	}
}

// Fake secret ENV VARIABLES use to authenticate against cloud provider
func setEnvVariables() {
	os.Setenv("AWS_ACCESS_KEY", "WPXKJC7CZQCFPKY5727N")
	os.Setenv("AWS_SECRET_KEY", "eHCSajxLvl94l36gIMlzZ/oW2O0rYYK+cVn5jNT2")
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AZURE_STORAGE_ACCOUNT", "goreleaser")
	os.Setenv("AZURE_STORAGE_KEY", "eHCSajxLvl94l36gIMlzZ/oW2O0rYYK+cVn5jNT2")
	gcloudCredentials, _ := filepath.Abs("./testdata/credentials.json")
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", gcloudCredentials)
}

func setEnv(env map[string]string) {

	for k, v := range env {
		os.Setenv(k, v)

	}
}
