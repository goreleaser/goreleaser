// Package blob provides a Pipe that push artifacts to blob supported by Go CDK
package blob

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/context"

	// Import the blob packages we want to be able to open.
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// Pipe for Artifactory
type Pipe struct {
	bucket OpenBucket
}

// String returns the description of the pipe
func (Pipe) String() string {
	return "Blob"
}

var (
	openbucket = newOpenBucket()
)

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Blob {
		blob := &ctx.Config.Blob[i]
		if blob.Bucket == "" {
			continue
		}
		if blob.Provider == "" {
			blob.Provider = "azblob"
		}
		// Validation before opening connection to bucket
		// gocdk also does this validation but doing it in advance for better error handling
		// as currently, go cdk does not throw error if AZURE_STORAGE_KEY is missing.
		err := checkProvider(blob.Provider)
		if err != nil {
			return err
		}
	}
	return nil
}

// Publish to specified blob bucket url
func (Pipe) Publish(ctx *context.Context) error {
	if len(ctx.Config.Blob) == 0 {
		return pipe.Skip("Blob section is not configured")
	}
	// Openning connectiong to the list of buckets

	var g = semerrgroup.New(ctx.Parallelism)
	for _, conf := range ctx.Config.Blob {
		conf := conf
		g.Go(func() error {
			return openbucket.UploadBucket(ctx, fmt.Sprintf("%s://%s", conf.Provider, conf.Bucket))
		})
	}
	return g.Wait()
}
