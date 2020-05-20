package blob

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
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
	var ctx = context.New(config.Project{
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "azblob",
				IDs:      []string{"foo", "bar"},
			},
			{
				Bucket:   "foobar",
				Provider: "gcs",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, []config.Blob{
		{
			Bucket:   "foo",
			Provider: "azblob",
			Folder:   "{{ .ProjectName }}/{{ .Tag }}",
			IDs:      []string{"foo", "bar"},
			Extra: config.BlobExtraFiles{
				Folder: "{{ .ProjectName }}/{{ .Tag }}",
				Path:   "/",
			},
		},
		{
			Bucket:   "foobar",
			Provider: "gcs",
			Folder:   "{{ .ProjectName }}/{{ .Tag }}",
			Extra: config.BlobExtraFiles{
				Folder: "{{ .ProjectName }}/{{ .Tag }}",
				Path:   "/",
			},
		},
	}, ctx.Config.Blobs)
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
	assert.Nil(Pipe{}.Default(ctx))
}

func TestPipe_Publish(t *testing.T) {
	pipePublish(t, config.BlobExtraFiles{})
}

func TestPipe_PublishExtraFiles(t *testing.T) {
	var extra = config.BlobExtraFiles{
		Path:  "testdata",
		Files: []string{"file.golden"},
	}
	pipePublish(t, extra)
}

func pipePublish(t *testing.T, extra config.BlobExtraFiles) {
	gcloudCredentials, _ := filepath.Abs("./testdata/credentials.json")

	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tgzpath := filepath.Join(folder, "bin.tar.gz")
	debpath := filepath.Join(folder, "bin.deb")
	assert.NoError(t, ioutil.WriteFile(tgzpath, []byte("fake\ntargz"), 0744))
	assert.NoError(t, ioutil.WriteFile(debpath, []byte("fake\ndeb"), 0744))

	// Azure Blob Context
	var azblobctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "testupload",
		Blobs: []config.Blob{
			{
				Bucket:   "foo",
				Provider: "azblob",
				Extra:    extra,
			},
		},
	})

	azblobctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	azblobctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	azblobctx.Artifacts.Add(&artifact.Artifact{
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
				Extra:    extra,
			},
		},
	})

	gsctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}

	gsctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	gsctx.Artifacts.Add(&artifact.Artifact{
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
				Extra:    extra,
			},
		},
	})
	s3ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	s3ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tgzpath,
	})
	s3ctx.Artifacts.Add(&artifact.Artifact{
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
			name:          "Azure Blob Bucket Test Publish",
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
			defer unsetEnv(tt.env)
			if err := p.Publish(tt.args.ctx); (err != nil) != tt.wantErr {
				if !strings.HasPrefix(err.Error(), tt.wantErrString) {
					t.Errorf("Pipe.Publish() error = %v, wantErr %v", err, tt.wantErrString)
				}
			}
		})
	}
}

func TestURL(t *testing.T) {
	t.Run("s3 with opts", func(t *testing.T) {
		url, err := urlFor(context.New(config.Project{}), config.Blob{
			Bucket:     "foo",
			Provider:   "s3",
			Region:     "us-west-1",
			Folder:     "foo",
			Endpoint:   "s3.foobar.com",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo?disableSSL=true&endpoint=s3.foobar.com&region=us-west-1&s3ForcePathStyle=true", url)
	})

	t.Run("s3 with some opts", func(t *testing.T) {
		url, err := urlFor(context.New(config.Project{}), config.Blob{
			Bucket:     "foo",
			Provider:   "s3",
			Region:     "us-west-1",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo?disableSSL=true&region=us-west-1", url)
	})

	t.Run("gs with opts", func(t *testing.T) {
		url, err := urlFor(context.New(config.Project{}), config.Blob{
			Bucket:     "foo",
			Provider:   "gs",
			Region:     "us-west-1",
			Folder:     "foo",
			Endpoint:   "s3.foobar.com",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "gs://foo", url)
	})

	t.Run("s3 no opts", func(t *testing.T) {
		url, err := urlFor(context.New(config.Project{}), config.Blob{
			Bucket:   "foo",
			Provider: "s3",
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo", url)
	})

	t.Run("gs no opts", func(t *testing.T) {
		url, err := urlFor(context.New(config.Project{}), config.Blob{
			Bucket:   "foo",
			Provider: "gs",
		})
		require.NoError(t, err)
		require.Equal(t, "gs://foo", url)
	})
}

func setEnv(env map[string]string) {
	for k, v := range env {
		os.Setenv(k, v)
	}
}

func unsetEnv(env map[string]string) {
	for k := range env {
		os.Unsetenv(k)
	}
}
