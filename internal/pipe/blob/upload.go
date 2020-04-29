package blob

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
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
func doUpload(ctx *context.Context, conf config.Blob, up uploader) error {
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

	var g = semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(filter).List() {
		artifact := artifact
		g.Go(func() error {
			// TODO: replace this with ?prefix=folder on the bucket url
			data, err := getData(ctx, conf, artifact.Path)
			if err != nil {
				return err
			}

			if err := up.Upload(ctx, bucketURL, filepath.Join(folder, artifact.Name), data); err != nil {
				switch {
				case errorContains(err, "NoSuchBucket", "ContainerNotFound", "notFound"):
					return errors.Wrapf(err, "provided bucket does not exist: %s", bucketURL)
				case errorContains(err, "NoCredentialProviders"):
					return errors.Wrapf(err, "check credentials and access to bucket: %s", bucketURL)
				case errorContains(err, "InvalidAccessKeyId"):
					return errors.Wrap(err, "aws access key id you provided does not exist in our records")
				case errorContains(err, "AuthenticationFailed"):
					return errors.Wrap(err, "azure storage key you provided is not valid")
				case errorContains(err, "invalid_grant"):
					return errors.Wrap(err, "google app credentials you provided is not valid")
				case errorContains(err, "no such host"):
					return errors.Wrap(err, "azure storage account you provided is not valid")
				case errorContains(err, "ServiceCode=ResourceNotFound"):
					return errors.Wrapf(err, "missing azure storage key for provided bucket %s", bucketURL)
				default:
					return errors.Wrap(err, "failed to write to bucket")
				}
			}
			return err
		})
	}
	return g.Wait()
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
	Upload(ctx *context.Context, url, path string, data []byte) error
}

// skipUploader is used when --skip-upload is set and will just log
// things without really doing anything
type skipUploader struct{}

func (u skipUploader) Upload(_ *context.Context, url, path string, _ []byte) error {
	log.WithFields(log.Fields{
		"bucket": url,
		"path":   path,
	}).Warn("doUpload skipped because skip-publish is set")
	return nil
}

// productionUploader actually do upload to
type productionUploader struct{}

func (u productionUploader) Upload(ctx *context.Context, url, path string, data []byte) (err error) {
	log.WithFields(log.Fields{
		"bucket": url,
		"path":   path,
	}).Info("uploading")

	// TODO: its not so great that we open one connection for each file
	conn, err := blob.OpenBucket(ctx, url)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := conn.Close(); err == nil {
			err = cerr
		}
	}()
	w, err := conn.NewWriter(ctx, path, nil)
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
