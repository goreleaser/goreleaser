package blob

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"testing/synctest"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"

	// register the file:// provider, used to exercise the real blob writer.
	_ "gocloud.dev/blob/fileblob"
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

	t.Run("file stages next to the destination", func(t *testing.T) {
		url, err := urlFor(testctx.Wrap(t.Context()), config.Blob{
			Bucket:   "/tmp/foo",
			Provider: "file",
		})
		require.NoError(t, err)
		require.Equal(t, "file:///tmp/foo?no_tmp_dir=true", url)
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

func TestUploadDataAWSKMSEncryptsBeforeUploading(t *testing.T) {
	const awsKMSLimit = 4096

	for name, tt := range map[string]struct {
		size         int
		wantRequests int64
		wantErr      string
	}{
		"accepts 4096 bytes": {
			size:         awsKMSLimit,
			wantRequests: 1,
		},
		"rejects 4097 bytes before kms": {
			size:         awsKMSLimit + 1,
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

			bucket := memblob.OpenBucket(nil)
			t.Cleanup(func() { require.NoError(t, bucket.Close()) })

			file := filepath.Join(t.TempDir(), "artifact")
			require.NoError(t, os.WriteFile(file, bytes.Repeat([]byte("a"), tt.size), 0o644))

			err := uploadData(testctx.Wrap(t.Context()), config.Blob{
				KMSKey: "awskms://alias/my-key?region=us-east-1&anonymous=true&hostname_immutable=true&endpoint=" + url.QueryEscape(server.URL),
			}, &productionUploader{bucket: bucket}, file, "dist/artifact", "mem://")

			require.Equal(t, tt.wantRequests, requests.Load())

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				exists, err := bucket.Exists(t.Context(), "dist/artifact")
				require.NoError(t, err)
				require.False(t, exists, "nothing should be uploaded when encryption fails")
				return
			}

			require.NoError(t, err)
			got, err := bucket.ReadAll(t.Context(), "dist/artifact")
			require.NoError(t, err)
			require.Equal(t, []byte("ciphertext"), got, "the ciphertext, not the plaintext, must reach the bucket")
		})
	}
}

func TestDoUploadDoesNotStartArtifactWorkersWhenExtraFilesFail(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		errExtraFiles := errors.New("extra files failed")
		uploader := &recordingUploader{block: make(chan struct{})}
		defer close(uploader.block)

		replaceBlobUploader(t, uploader)
		previousFindExtraFiles := findExtraFiles
		findExtraFiles = func(*context.Context, []config.ExtraFile) (map[string]string, error) {
			synctest.Wait()
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
	})
}

func TestDoUploadUploadsArtifactsAndExtraFiles(t *testing.T) {
	uploader := &recordingUploader{}
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

func TestProductionUploaderUploadsFileContentsAndDetectsContentType(t *testing.T) {
	dir := t.TempDir()
	contents := map[string][]byte{
		"empty.txt":  {},
		"small.txt":  []byte("small file"),
		"script.sh":  []byte("#!/bin/sh\necho hello\n"),
		"large.bin":  bytes.Repeat([]byte("0123456789abcdef"), 1<<16),
		"uneven.bin": bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6}, 100_003),
	}

	bucket := memblob.OpenBucket(nil)
	t.Cleanup(func() { require.NoError(t, bucket.Close()) })
	up := &productionUploader{bucket: bucket}
	ctx := testctx.Wrap(t.Context())

	for name, content := range contents {
		file := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(file, content, 0o644))
		require.NoError(t, uploadData(ctx, config.Blob{}, up, file, "dist/"+name, "mem://"))

		got, err := bucket.ReadAll(t.Context(), "dist/"+name)
		require.NoError(t, err)
		require.Equal(t, content, got, name)

		attrs, err := bucket.Attributes(t.Context(), "dist/"+name)
		require.NoError(t, err)
		require.Equal(t, http.DetectContentType(content), attrs.ContentType, name)
	}
}

func TestProductionUploaderDoesNotCommitPartialUploads(t *testing.T) {
	// sizes below and above the 512-byte content-type sniff length, so both
	// the buffered and the already-open writer paths are covered.
	for _, size := range []int{100, 100_000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			bucket := memblob.OpenBucket(nil)
			t.Cleanup(func() { require.NoError(t, bucket.Close()) })

			readErr := errors.New("simulated read failure")
			err := (&productionUploader{bucket: bucket}).Upload(
				testctx.Wrap(t.Context()),
				"dist/artifact",
				io.MultiReader(
					bytes.NewReader(bytes.Repeat([]byte("a"), size)),
					iotest.ErrReader(readErr),
				),
			)

			require.ErrorIs(t, err, readErr)
			exists, err := bucket.Exists(t.Context(), "dist/artifact")
			require.NoError(t, err)
			require.False(t, exists, "a failed read must not leave a truncated object behind")
		})
	}
}

func TestUploadDataMissingFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "missing.txt")
	err := uploadData(testctx.Wrap(t.Context()), config.Blob{}, &recordingUploader{}, file, "dist/missing.txt", "mem://")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorContains(t, err, "failed to open file "+file)
}

func TestPublishAllocationDoesNotScaleWithFileSize(t *testing.T) {
	const (
		files        = 4
		destinations = 4
		smallFile    = 1 << 20
		largeFile    = 4 << 20
	)

	small := publishAllocBytes(t, files, destinations, smallFile)
	large := publishAllocBytes(t, files, destinations, largeFile)

	// Buffering every file for every destination grows allocation by
	// files*destinations*(largeFile-smallFile), ~48MiB here. Streaming copies
	// through a fixed-size buffer, so growing the files must not move the
	// total by even one large file's worth.
	require.Less(t, large, small+uint64(largeFile),
		"allocation scaled with file size: %d bytes for %dMiB files vs %d bytes for %dMiB files",
		large, largeFile>>20, small, smallFile>>20)
}

// publishAllocBytes reports how many bytes Publish allocates while uploading
// files of the given size to every destination. It uses the file:// provider so
// the real productionUploader and blob writer run, and nothing is retained by
// the bucket itself.
func publishAllocBytes(tb testing.TB, files, destinations, fileSize int) uint64 {
	tb.Helper()

	dir := tb.TempDir()
	root := tb.TempDir()
	ctx := testctx.WrapWithCfg(tb.Context(), config.Project{})
	ctx.Parallelism = files

	for i := range files {
		name := fmt.Sprintf("file%d.bin", i)
		file := filepath.Join(dir, name)
		require.NoError(tb, os.WriteFile(file, bytes.Repeat([]byte{byte(i)}, fileSize), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableArchive,
			Name: name,
			Path: file,
		})
	}
	for i := range destinations {
		bucket := filepath.Join(root, fmt.Sprintf("bucket%d", i))
		require.NoError(tb, os.MkdirAll(bucket, 0o700))
		ctx.Config.Blobs = append(ctx.Config.Blobs, config.Blob{
			Provider: "file",
			// file:// needs a slash-separated absolute path; on Windows
			// filepath gives D:\..., which is not a valid URL host.
			Bucket:    "/" + strings.TrimPrefix(filepath.ToSlash(bucket), "/"),
			Directory: "dist",
		})
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	require.NoError(tb, Pipe{}.Publish(ctx))
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc

	// guard against measuring a run that uploaded nothing.
	for i := range files {
		for j := range destinations {
			got, err := os.Stat(filepath.Join(root, fmt.Sprintf("bucket%d", j), "dist", fmt.Sprintf("file%d.bin", i)))
			require.NoError(tb, err)
			require.EqualValues(tb, fileSize, got.Size())
		}
	}
	return allocated
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

type recordingUploader struct {
	opens            atomic.Int64
	closes           atomic.Int64
	active           atomic.Int64
	closedWithActive atomic.Bool

	block chan struct{}

	mu    sync.Mutex
	files []string
}

func (u *recordingUploader) Open(*context.Context, string) error {
	u.opens.Add(1)
	return nil
}

func (u *recordingUploader) Upload(_ *context.Context, path string, _ io.Reader) error {
	u.active.Add(1)
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
