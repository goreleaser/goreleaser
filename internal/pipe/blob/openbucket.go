package blob

import (
	"fmt"
	"io/ioutil"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/context"
	gocdk "gocloud.dev/blob"

	// Import the blob packages we want to be able to open.
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// OpenBucket is the interface that wraps the BucketConnect and UploadBucket method
type OpenBucket interface {
	BucketConnect(ctx *context.Context, bucketURL string) (*gocdk.Bucket, error)
	UploadBucket(ctx *context.Context, bucketURL string) error
}

// Bucket is object which holds connection for Go Bucker Provider
type bucket struct {
	bucketConnect *gocdk.Bucket
}

// returns openbucket connection for list of providers
func newOpenBucket() OpenBucket {
	return bucket{}
}

// BucketConnect makes connection with provider
func (b bucket) BucketConnect(ctx *context.Context, bucketURL string) (*gocdk.Bucket, error) {

	bucketConnection, err := gocdk.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, err
	}
	return bucketConnection, nil
}

// UploadBucket takes connection initilized from newOpenBucket to upload goreleaser artifacts
// Takes goreleaser context(which includes artificats) and bucketURL for upload destination (gs://gorelease-bucket)
func (b bucket) UploadBucket(ctx *context.Context, bucketURL string) error {

	// Get the openbucket connection for specific provider
	openbucketConn, err := b.BucketConnect(ctx, bucketURL)
	if err != nil {
		return err
	}
	defer openbucketConn.Close()

	var g = semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
			artifact.ByType(artifact.Checksum),
			artifact.ByType(artifact.Signature),
			artifact.ByType(artifact.LinuxPackage),
		),
	).List() {
		artifact := artifact
		g.Go(func() error {
			// Prepare artifact for upload.
			data, err := ioutil.ReadFile(artifact.Path)
			if err != nil {
				return err
			}
			log.WithFields(log.Fields{
				"provider": bucketURL,
				"artifact": artifact.Name,
			}).Info("uploading")

			w, err := openbucketConn.NewWriter(ctx, artifact.Path, nil)
			if err != nil {
				return fmt.Errorf("Failed to obtain writer: %s", err)
			}
			_, err = w.Write(data)
			if err != nil {
				if errorContains(err, "NoSuchBucket", "ContainerNotFound", "notFound") {
					return fmt.Errorf("(%v) Provided bucket does not exist", bucketURL)
				}
				return fmt.Errorf("Failed to write to bucket : %s", err)
			}
			if err = w.Close(); err != nil {
				// Invalid AWS Keys
				if errorContains(err, "InvalidAccessKeyId") {
					return fmt.Errorf("The AWS Access Key Id you provided does not exist in our records")
					// Invalid AZURE_STORAGE_KEY
				} else if errorContains(err, "AuthenticationFailed") {
					return fmt.Errorf("The Azure Storage Key you provided is not valid")
					// Invalid GC Credentials
				} else if errorContains(err, "invalid_grant") {
					return fmt.Errorf("The Google App Credentials you provided is not valid")
				} else if errorContains(err, "blob.core.windows.net: no such host") {
					return fmt.Errorf("The Azure Storage Account you provided is not valid")
					// Not existing bucket
				} else if errorContains(err, "NoSuchBucket", "ContainerNotFound", "notFound") {
					return fmt.Errorf("(%v) Provided bucket does not exist", bucketURL)
				} else {
					return fmt.Errorf("Failed to close Bucket writer: %s", err)
				}
			}
			return err
		})
	}
	return g.Wait()
}
