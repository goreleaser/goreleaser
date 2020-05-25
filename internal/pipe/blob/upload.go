package blob

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	"gocloud.dev/secrets"

	// Import the blob packages we want to be able to open.
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"

	// import the secrets packages we want to be able to open:
	_ "gocloud.dev/secrets/awskms"
	_ "gocloud.dev/secrets/azurekeyvault"
	_ "gocloud.dev/secrets/gcpkms"
)

func urlFor(ctx *context.Context, conf config.Blob) (string, error) {
	bucket, err := tmpl.New(ctx).Apply(conf.Bucket)
	if err != nil {
		return "", err
	}

	bucketURL := fmt.Sprintf("%s://%s", conf.Provider, bucket)

	if conf.Provider != "s3" {
		return bucketURL, nil
	}

	var query = url.Values{}
	if conf.Endpoint != "" {
		query.Add("endpoint", conf.Endpoint)
		query.Add("s3ForcePathStyle", "true")
	}
	if conf.Region != "" {
		query.Add("region", conf.Region)
	}
	if conf.DisableSSL {
		query.Add("disableSSL", "true")
	}

	if len(query) > 0 {
		bucketURL = bucketURL + "?" + query.Encode()
	}

	return bucketURL, nil
}

// Takes goreleaser context(which includes artificats) and bucketURL for
// upload to destination (eg: gs://gorelease-bucket) using the given uploader
// implementation
func doUpload(ctx *context.Context, conf config.Blob) error {
	folder, err := tmpl.New(ctx).Apply(conf.Folder)
	if err != nil {
		return err
	}

	bucketURL, err := urlFor(ctx, conf)
	if err != nil {
		return err
	}

	var filter = artifact.Or(
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableBinary),
		artifact.ByType(artifact.UploadableSourceArchive),
		artifact.ByType(artifact.Checksum),
		artifact.ByType(artifact.Signature),
		artifact.ByType(artifact.LinuxPackage),
	)
	if len(conf.IDs) > 0 {
		filter = artifact.And(filter, artifact.ByIDs(conf.IDs...))
	}

	var up = newUploader(ctx)
	if err := up.Open(ctx, bucketURL); err != nil {
		return handleError(err, bucketURL)
	}
	defer up.Close()

	var g = semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(filter).List() {
		artifact := artifact
		g.Go(func() error {
			// TODO: replace this with ?prefix=folder on the bucket url
			var dataFile = artifact.Path
			var uploadFile = path.Join(folder, artifact.Name)

			err := uploadData(ctx, conf, up, dataFile, uploadFile, bucketURL)

			return err
		})
	}

	files, err := extrafiles.Find(conf.ExtraFiles)
	if err != nil {
		return err
	}
	for name, fullpath := range files {
		name := name
		fullpath := fullpath
		g.Go(func() error {
			var uploadFile = path.Join(folder, name)

			err := uploadData(ctx, conf, up, fullpath, uploadFile, bucketURL)

			return err
		})
	}

	return g.Wait()
}

func uploadData(ctx *context.Context, conf config.Blob, up uploader, dataFile, uploadFile, bucketURL string) error {
	data, err := getData(ctx, conf, dataFile)
	if err != nil {
		return err
	}

	err = up.Upload(ctx, uploadFile, data)
	if err != nil {
		return handleError(err, bucketURL)
	}
	return err
}

func handleError(err error, url string) error {
	switch {
	case errorContains(err, "NoSuchBucket", "ContainerNotFound", "notFound"):
		return errors.Wrapf(err, "provided bucket does not exist: %s", url)
	case errorContains(err, "NoCredentialProviders"):
		return errors.Wrapf(err, "check credentials and access to bucket: %s", url)
	case errorContains(err, "InvalidAccessKeyId"):
		return errors.Wrap(err, "aws access key id you provided does not exist in our records")
	case errorContains(err, "AuthenticationFailed"):
		return errors.Wrap(err, "azure storage key you provided is not valid")
	case errorContains(err, "invalid_grant"):
		return errors.Wrap(err, "google app credentials you provided is not valid")
	case errorContains(err, "no such host"):
		return errors.Wrap(err, "azure storage account you provided is not valid")
	case errorContains(err, "ServiceCode=ResourceNotFound"):
		return errors.Wrapf(err, "missing azure storage key for provided bucket %s", url)
	default:
		return errors.Wrap(err, "failed to write to bucket")
	}
}

func newUploader(ctx *context.Context) uploader {
	if ctx.SkipPublish {
		return &skipUploader{}
	}
	return &productionUploader{}
}

func getData(ctx *context.Context, conf config.Blob, path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return data, errors.Wrapf(err, "failed to open file %s", path)
	}
	if conf.KMSKey == "" {
		return data, nil
	}
	keeper, err := secrets.OpenKeeper(ctx, conf.KMSKey)
	if err != nil {
		return data, errors.Wrapf(err, "failed to open kms %s", conf.KMSKey)
	}
	defer keeper.Close()
	data, err = keeper.Encrypt(ctx, data)
	if err != nil {
		return data, errors.Wrap(err, "failed to encrypt with kms")
	}
	return data, err
}

// uploader implements upload
type uploader interface {
	io.Closer
	Open(ctx *context.Context, url string) error
	Upload(ctx *context.Context, path string, data []byte) error
}

// skipUploader is used when --skip-upload is set and will just log
// things without really doing anything
type skipUploader struct{}

func (u *skipUploader) Close() error                            { return nil }
func (u *skipUploader) Open(_ *context.Context, _ string) error { return nil }

func (u *skipUploader) Upload(_ *context.Context, path string, _ []byte) error {
	log.WithField("path", path).Warn("upload skipped because skip-publish is set")
	return nil
}

// productionUploader actually do upload to
type productionUploader struct {
	bucket *blob.Bucket
}

func (u *productionUploader) Close() error {
	if u.bucket == nil {
		return nil
	}
	return u.bucket.Close()
}
func (u *productionUploader) Open(ctx *context.Context, bucket string) error {
	log.WithFields(log.Fields{
		"bucket": bucket,
	}).Debug("uploading")

	conn, err := blob.OpenBucket(ctx, bucket)
	if err != nil {
		return err
	}
	u.bucket = conn
	return nil
}

func (u *productionUploader) Upload(ctx *context.Context, path string, data []byte) (err error) {
	log.WithField("path", path).Info("uploading")

	w, err := u.bucket.NewWriter(ctx, path, nil)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := w.Close(); err == nil {
			err = cerr
		}
	}()
	_, err = w.Write(data)
	return
}
