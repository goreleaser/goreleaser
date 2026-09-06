package blob

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
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
			require.EqualError(t, handleError(errors.New(k), "someurl"), v)
		})
	}
}

func TestDefaultsNoConfig(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Blobs: []config.Blob{{}},
	})

	require.EqualError(t, Pipe{}.Default(ctx), errorString)
}

func TestDefaultsNoBucket(t *testing.T) {
	errorString := "bucket or provider cannot be empty"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Blobs: []config.Blob{
			{
				Bucket: "goreleaser-bucket",
			},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), errorString)
}

func TestDefaults(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Blobs: []config.Blob{
			{
				Bucket:             "foo",
				Provider:           "azblob",
				IDs:                []string{"foo", "bar"},
				ContentDisposition: "inline",
			},
			{
				Bucket:   "foobar2",
				Provider: "gcs",
			},
			{
				Bucket:             "foobar",
				Provider:           "gcs",
				ContentDisposition: "-",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, []config.Blob{
		{
			Bucket:             "foo",
			Provider:           "azblob",
			Directory:          "{{ .ProjectName }}/{{ .Tag }}",
			IDs:                []string{"foo", "bar"},
			ContentDisposition: "inline",
		},
		{
			Bucket:             "foobar2",
			Provider:           "gcs",
			Directory:          "{{ .ProjectName }}/{{ .Tag }}",
			ContentDisposition: "attachment;filename={{.Filename}}",
		},
		{
			Bucket:             "foobar",
			Provider:           "gcs",
			Directory:          "{{ .ProjectName }}/{{ .Tag }}",
			ContentDisposition: "",
		},
	}, ctx.Config.Blobs)
}

func TestDefaultsWithProvider(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:     "foo",
			Provider:   "s3",
			Region:     "us-west-1",
			Directory:  "foo",
			Endpoint:   "s3.foobar.com",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo?disable_https=true&endpoint=s3.foobar.com&region=us-west-1&s3ForcePathStyle=true", url)
	})

	t.Run("s3 with some opts", func(t *testing.T) {
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:     "foo",
			Provider:   "s3",
			Region:     "us-west-1",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo?disable_https=true&region=us-west-1", url)
	})

	t.Run("gs with opts", func(t *testing.T) {
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:     "foo",
			Provider:   "gs",
			Region:     "us-west-1",
			Directory:  "foo",
			Endpoint:   "s3.foobar.com",
			DisableSSL: true,
		})
		require.NoError(t, err)
		require.Equal(t, "gs://foo", url)
	})

	t.Run("s3 force path style without endpoint", func(t *testing.T) {
		for _, forcePathStyle := range []bool{true, false} {
			t.Run(strconv.FormatBool(forcePathStyle), func(t *testing.T) {
				url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
					Bucket:           "foo",
					Provider:         "s3",
					Region:           "us-west-1",
					S3ForcePathStyle: &forcePathStyle,
				})
				require.NoError(t, err)
				require.Equal(
					t,
					"s3://foo?region=us-west-1&s3ForcePathStyle="+strconv.FormatBool(forcePathStyle),
					url,
				)
			})
		}
	})

	t.Run("s3 force path style false with endpoint", func(t *testing.T) {
		forcePathStyle := false
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:           "foo",
			Provider:         "s3",
			Endpoint:         "s3.foobar.com",
			S3ForcePathStyle: &forcePathStyle,
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo?endpoint=s3.foobar.com&s3ForcePathStyle=false", url)
	})

	t.Run("s3 no opts", func(t *testing.T) {
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:   "foo",
			Provider: "s3",
		})
		require.NoError(t, err)
		require.Equal(t, "s3://foo", url)
	})

	t.Run("gs no opts", func(t *testing.T) {
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:   "foo",
			Provider: "gs",
		})
		require.NoError(t, err)
		require.Equal(t, "gs://foo", url)
	})

	t.Run("template errors", func(t *testing.T) {
		t.Run("provider", func(t *testing.T) {
			_, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
				Provider: "{{ .Nope }}",
			})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("bucket", func(t *testing.T) {
			_, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
				Bucket:   "{{ .Nope }}",
				Provider: "gs",
			})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("endpoint", func(t *testing.T) {
			_, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
				Bucket:   "foobar",
				Endpoint: "{{.Env.NOPE}}",
				Provider: "s3",
			})
			testlib.RequireTemplateError(t, err)
		})
		t.Run("region", func(t *testing.T) {
			_, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
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
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Blobs: []config.Blob{{}},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestGetDataAWSKMSPlaintextLimit(t *testing.T) {
	const awsKMSLimit = 4096

	for name, tt := range map[string]struct {
		size         int
		wantData     []byte
		wantRequests int64
		wantErr      string
	}{
		"accepts 4096 bytes": {
			size:         awsKMSLimit,
			wantData:     []byte("ciphertext"),
			wantRequests: 1,
		},
		"rejects 4097 bytes before kms": {
			size:         awsKMSLimit + 1,
			wantData:     bytes.Repeat([]byte("a"), awsKMSLimit+1),
			wantRequests: 0,
			wantErr:      "failed to encrypt with kms: awskms encryption supports files up to 4096 bytes, got 4097 bytes",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/x-amz-json-1.1")

				var input struct {
					Plaintext []byte `json:"Plaintext"`
				}
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if len(input.Plaintext) > awsKMSLimit {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprint(w, `{"__type":"ValidationException","message":"plaintext too large"}`)
					return
				}

				_, _ = fmt.Fprintf(w, `{"CiphertextBlob":%q,"KeyId":"alias/my-key"}`,
					base64.StdEncoding.EncodeToString([]byte("ciphertext")))
			}))
			t.Cleanup(server.Close)

			file := filepath.Join(t.TempDir(), "artifact")
			require.NoError(t, os.WriteFile(file, bytes.Repeat([]byte("a"), tt.size), 0o644))

			data, err := getData(testctx.Wrap(t.Context()), config.Blob{
				KMSKey: "awskms://alias/my-key?region=us-east-1&anonymous=true&hostname_immutable=true&endpoint=" + url.QueryEscape(server.URL),
			}, file)

			require.Equal(t, tt.wantData, data)
			require.Equal(t, tt.wantRequests, requests.Load())
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}
