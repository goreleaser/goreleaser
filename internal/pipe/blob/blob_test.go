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
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
					Plaintext []byte `json:"Plaintext"` //nolint:tagliatelle // AWS KMS Encrypt wire format uses Plaintext.
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

func TestDoUploadDoesNotStartArtifactWorkersWhenExtraFilesFail(t *testing.T) {
	errExtraFiles := errors.New("extra files failed")
	uploader := newRecordingUploader()
	uploader.block = make(chan struct{})
	defer close(uploader.block)

	replaceBlobUploader(t, uploader)
	previousFindExtraFiles := findExtraFiles
	findExtraFiles = func(*context.Context, []config.ExtraFile) (map[string]string, error) {
		for range 1000 {
			runtime.Gosched()
			select {
			case <-uploader.started:
				return nil, errors.New("artifact upload started before extra files were resolved")
			default:
			}
		}
		return nil, errExtraFiles
	}
	t.Cleanup(func() { findExtraFiles = previousFindExtraFiles })

	ctx, conf := blobUploadContext(t, []string{"one.txt", "two.txt"}, nil)
	conf.ExtraFiles = []config.ExtraFile{{Glob: filepath.Join(t.TempDir(), "missing.txt")}}

	err := doUpload(ctx, conf)

	require.ErrorIs(t, err, errExtraFiles)
	require.Equal(t, int64(1), uploader.opens.Load())
	require.Equal(t, int64(1), uploader.closes.Load())
	require.Zero(t, uploader.active.Load())
	require.False(t, uploader.closedWithActive.Load())
	require.Empty(t, uploader.uploaded())
}

func TestDoUploadUploadsArtifactsAndExtraFiles(t *testing.T) {
	uploader := newRecordingUploader()
	replaceBlobUploader(t, uploader)

	ctx, conf := blobUploadContext(t, []string{"one.txt", "two.txt"}, []config.ExtraFile{{Glob: "./testdata/file.golden"}})

	require.NoError(t, doUpload(ctx, conf))
	require.Equal(t, int64(1), uploader.opens.Load())
	require.Equal(t, int64(1), uploader.closes.Load())
	require.Zero(t, uploader.active.Load())
	require.False(t, uploader.closedWithActive.Load())
	require.ElementsMatch(t, []string{
		"dist/file.golden",
		"dist/one.txt",
		"dist/two.txt",
	}, uploader.uploaded())
}

func blobUploadContext(tb testing.TB, names []string, extraFiles []config.ExtraFile) (*context.Context, config.Blob) {
	tb.Helper()

	dir := tb.TempDir()
	conf := config.Blob{
		Provider:   "test",
		Bucket:     "bucket",
		Directory:  "dist",
		ExtraFiles: extraFiles,
	}
	ctx := testctx.WrapWithCfg(tb.Context(), config.Project{})
	for _, name := range names {
		file := filepath.Join(dir, name)
		require.NoError(tb, os.WriteFile(file, []byte(name), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableArchive,
			Name: name,
			Path: file,
		})
	}
	return ctx, conf
}

func replaceBlobUploader(tb testing.TB, rec *recordingUploader) {
	tb.Helper()

	previous := newUploader
	newUploader = func(config.Blob, string) uploader {
		return rec
	}
	tb.Cleanup(func() { newUploader = previous })
}

func newRecordingUploader() *recordingUploader {
	return &recordingUploader{
		started: make(chan struct{}),
	}
}

type recordingUploader struct {
	opens            atomic.Int64
	closes           atomic.Int64
	active           atomic.Int64
	closedWithActive atomic.Bool

	startOnce sync.Once
	started   chan struct{}
	block     chan struct{}

	mu    sync.Mutex
	files []string
}

func (u *recordingUploader) Open(*context.Context, string) error {
	u.opens.Add(1)
	return nil
}

func (u *recordingUploader) Upload(_ *context.Context, path string, _ []byte) error {
	u.active.Add(1)
	u.startOnce.Do(func() { close(u.started) })
	if u.block != nil {
		<-u.block
	}
	defer u.active.Add(-1)

	u.mu.Lock()
	defer u.mu.Unlock()
	u.files = append(u.files, path)
	return nil
}

func (u *recordingUploader) Close() error {
	u.closes.Add(1)
	if u.active.Load() > 0 {
		u.closedWithActive.Store(true)
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return nil
}

func (u *recordingUploader) uploaded() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]string(nil), u.files...)
}
