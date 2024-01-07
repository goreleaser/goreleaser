package blob

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestErrors(t *testing.T) {
	for k, v := range map[string]string{
		"NoSuchBucket":                 "provided bucket does not exist: someurl: NoSuchBucket",
		"ContainerNotFound":            "provided bucket does not exist: someurl: ContainerNotFound",
		"notFound":                     "provided bucket does not exist: someurl: notFound",
		"NoCredentialProviders":        "check credentials and access to bucket: someurl: NoCredentialProviders",
		"InvalidAccessKeyId":           "aws access key id you provided does not exist in our records: InvalidAccessKeyId",
		"AuthenticationFailed":         "azure storage key you provided is not valid: AuthenticationFailed",
		"invalid_grant":                "google app credentials you provided is not valid: invalid_grant",
		"no such host":                 "azure storage account you provided is not valid: no such host",
		"ServiceCode=ResourceNotFound": "missing azure storage key for provided bucket someurl: ServiceCode=ResourceNotFound",
		"other":                        "failed to write to bucket: other",
	} {
		t.Run(k, func(t *testing.T) {
			require.EqualError(t, handleError(fmt.Errorf(k), "someurl"), v)
		})
	}
}

func TestDefaultsNoConfig(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	ctx := testctx.NewWithCfg(config.Project{
		Blobs: []config.Blob{{}},
	})
	require.EqualError(t, Pipe{}.Default(ctx), errorString)
}

func TestDefaultsNoBucket(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	ctx := testctx.NewWithCfg(config.Project{
		Blobs: []config.Blob{
			{
				Provider: "azblob",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), errorString)
}

func TestDefaultsNoProvider(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	ctx := testctx.NewWithCfg(config.Project{
		Blobs: []config.Blob{
			{
				Bucket: "goreleaser-bucket",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), errorString)
}

func TestDefaults(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Blobs: []config.Blob{
			{
				Bucket:             "foo",
				Provider:           "azblob",
				IDs:                []string{"foo", "bar"},
				ContentDisposition: "inline",
			},
			{
				Bucket:   "foobar",
				Provider: "gcs",
			},
			{
				Bucket:        "deprecated",
				Provider:      "s3",
				Folder:        "static",
				OldDisableSSL: true,
				OldKMSKey:     "fake",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, []config.Blob{
		{
			Bucket:             "foo",
			Provider:           "azblob",
			Folder:             "{{ .ProjectName }}/{{ .Tag }}",
			IDs:                []string{"foo", "bar"},
			ContentDisposition: "inline",
		},
		{
			Bucket:             "foobar",
			Provider:           "gcs",
			Folder:             "{{ .ProjectName }}/{{ .Tag }}",
			ContentDisposition: "attachment;filename={{.Filename}}",
		},
		{
			Bucket:             "deprecated",
			Provider:           "s3",
			Folder:             "static",
			OldDisableSSL:      true,
			DisableSSL:         true,
			OldKMSKey:          "fake",
			KMSKey:             "fake",
			ContentDisposition: "attachment;filename={{.Filename}}",
		},
	}, ctx.Config.Blobs)
}

func TestDefaultsWithProvider(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	require.NoError(t, Pipe{}.Default(ctx))
}

func TestURL(t *testing.T) {
	t.Run("s3 with opts", func(t *testing.T) {
		url, err := urlFor(testctx.New(), config.Blob{
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
		url, err := urlFor(testctx.New(), config.Blob{
			Bucket:     "foo",
			Provider:   "s3",
			Region:     "us-west-1",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo?disableSSL=true&region=us-west-1", url)
	})

	t.Run("gs with opts", func(t *testing.T) {
		url, err := urlFor(testctx.New(), config.Blob{
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
		url, err := urlFor(testctx.New(), config.Blob{
			Bucket:   "foo",
			Provider: "s3",
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo", url)
	})

	t.Run("gs no opts", func(t *testing.T) {
		url, err := urlFor(testctx.New(), config.Blob{
			Bucket:   "foo",
			Provider: "gs",
		})
		require.NoError(t, err)
		require.Equal(t, "gs://foo", url)
	})

	t.Run("template errors", func(t *testing.T) {
		t.Run("provider", func(t *testing.T) {
			_, err := urlFor(testctx.New(), config.Blob{
				Provider: "{{ .Nope }}",
			})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("bucket", func(t *testing.T) {
			_, err := urlFor(testctx.New(), config.Blob{
				Bucket:   "{{ .Nope }}",
				Provider: "gs",
			})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("endpoint", func(t *testing.T) {
			_, err := urlFor(testctx.New(), config.Blob{
				Bucket:   "foobar",
				Endpoint: "{{.Env.NOPE}}",
				Provider: "s3",
			})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("region", func(t *testing.T) {
			_, err := urlFor(testctx.New(), config.Blob{
				Bucket:   "foobar",
				Region:   "{{.Env.NOPE}}",
				Provider: "s3",
			})
			testlib.RequireTemplateError(t, err)
		})
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Blobs: []config.Blob{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
