// Package s3 provides a Pipe that push artifacts to s3/minio
package s3

import (
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/nametemplate"
	"golang.org/x/sync/errgroup"
)

// Pipe for Artifactory
type Pipe struct{}

// String returns the description of the pipe
func (Pipe) String() string {
	return "releasing to s3"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.S3 {
		s3 := &ctx.Config.S3[i]
		if s3.Folder == "" {
			s3.Folder = "{{ .ProjectName }}/{{ .Tag }}"
		}
		if s3.Region == "" {
			s3.Region = "us-east-1"
		}
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group
	sem := make(chan bool, ctx.Parallelism)
	for _, conf := range ctx.Config.S3 {
		conf := conf
		sem <- true
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return upload(ctx, conf)
		})
	}
	return g.Wait()
}

func upload(ctx *context.Context, conf config.S3) error {
	var awsConfig = aws.Config{}
	if conf.Endpoint != "" {
		awsConfig.Endpoint = aws.String(conf.Endpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
	}
	sess := session.Must(session.NewSession(&awsConfig))
	svc := s3.New(sess, &aws.Config{
		Region: aws.String(conf.Region),
	})
	folder, err := nametemplate.Apply(ctx, "s3", conf.Folder)
	if err != nil {
		return err
	}

	var g errgroup.Group
	sem := make(chan bool, ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
			artifact.ByType(artifact.Checksum),
			artifact.ByType(artifact.Signature),
			artifact.ByType(artifact.LinuxPackage),
		),
	).List() {
		sem <- true
		artifact := artifact
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			f, err := os.Open(artifact.Path)
			if err != nil {
				return err
			}
			log.WithFields(log.Fields{
				"bucket":   conf.Bucket,
				"folder":   folder,
				"artifact": artifact.Name,
			}).Info("uploading")
			_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
				Bucket: aws.String(conf.Bucket),
				Key:    aws.String(filepath.Join(folder, artifact.Name)),
				Body:   f,
			})
			return err
		})
	}
	return g.Wait()
}
