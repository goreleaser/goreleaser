package blob

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	gocdk "gocloud.dev/blob"

	// Import the blob packages we want to be able to open.
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// OpenBucket is the interface that wraps the BucketConnect and UploadBucket method
type OpenBucket interface {
	Connect(ctx *context.Context, bucketURL string) (*gocdk.Bucket, error)
	Upload(ctx *context.Context, conf config.Blob, folder string) error
}

// Bucket is object which holds connection for Go Bucker Provider
type Bucket struct {
	BucketConn *gocdk.Bucket
}

// returns openbucket connection for list of providers
func newOpenBucket() OpenBucket {
	return Bucket{}
}

// Connect makes connection with provider
func (b Bucket) Connect(ctx *context.Context, bucketURL string) (*gocdk.Bucket, error) {
	bucketConnection, err := gocdk.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, err
	}
	return bucketConnection, nil
}

// Upload takes connection initilized from newOpenBucket to upload goreleaser artifacts
// Takes goreleaser context(which includes artificats) and bucketURL for upload destination (gs://gorelease-bucket)
func (b Bucket) Upload(ctx *context.Context, conf config.Blob, folder string) error {
	var bucketURL = fmt.Sprintf("%s://%s", conf.Provider, conf.Bucket)

	// Get the openbucket connection for specific provider
	openbucketConn, err := b.Connect(ctx, bucketURL)
	if err != nil {
		return err
	}
	defer openbucketConn.Close()

	var filter = artifact.Or(
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableBinary),
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
			// Prepare artifact for upload.
			data, err := ioutil.ReadFile(artifact.Path)
			if err != nil {
				return err
			}
			log.WithFields(log.Fields{
				"provider": bucketURL,
				"artifact": artifact.Name,
			}).Info("uploading")

			w, err := openbucketConn.NewWriter(ctx, filepath.Join(folder, artifact.Path), nil)
			if err != nil {
				return fmt.Errorf("failed to obtain writer: %s", err)
			}
			_, err = w.Write(data)
			if err != nil {
				if errorContains(err, "NoSuchBucket", "ContainerNotFound", "notFound") {
					return fmt.Errorf("(%v) provided bucket does not exist", bucketURL)
				}
				return fmt.Errorf("failed to write to bucket : %s", err)
			}
			if err = w.Close(); err != nil {
				switch {
				case errorContains(err, "InvalidAccessKeyId"):
					return fmt.Errorf("aws access key id you provided does not exist in our records")
				case errorContains(err, "AuthenticationFailed"):
					return fmt.Errorf("azure storage key you provided is not valid")
				case errorContains(err, "invalid_grant"):
					return fmt.Errorf("google app credentials you provided is not valid")
				case errorContains(err, "no such host"):
					return fmt.Errorf("azure storage account you provided is not valid")
				case errorContains(err, "NoSuchBucket", "ContainerNotFound", "notFound"):
					return fmt.Errorf("(%v) provided bucket does not exist", bucketURL)
				default:
					return fmt.Errorf("failed to close Bucket writer: %s", err)
				}
			}
			return err
		})
	}
	return g.Wait()
}
