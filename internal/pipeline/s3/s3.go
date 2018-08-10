// Package s3 provides a Pipe that push artifacts to s3/minio
package s3

import (
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
		if s3.Bucket == "" {
			continue
		}
		if s3.Folder == "" {
			s3.Folder = "{{ .ProjectName }}/{{ .Tag }}"
		}
		if s3.Region == "" {
			s3.Region = "us-east-1"
		}
		if s3.ACL == "" {
			s3.ACL = "private"
		}
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipeline.ErrSkipPublishEnabled
	}
	var g = semerrgroup.New(ctx.Parallelism)
	for _, conf := range ctx.Config.S3 {
		conf := conf
		g.Go(func() error {
			return upload(ctx, conf)
		})
	}
	return g.Wait()
}

func upload(ctx *context.Context, conf config.S3) error {
	var awsConfig = aws.NewConfig()
	// TODO: add a test for this
	if conf.Endpoint != "" {
		awsConfig.Endpoint = aws.String(conf.Endpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
	}
	awsConfig.Credentials = credentials.NewChainCredentials([]credentials.Provider{
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{
			Profile: conf.Profile,
		},
	})

	_, err := awsConfig.Credentials.Get()
	var sess *session.Session
	if err == nil {
		sess = session.Must(session.NewSession(awsConfig))
	} else {
		// Specify profile and assume an IAM role with MFA prompting for token code on stdin
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			SharedConfigState:       session.SharedConfigEnable,
			Profile:                 conf.Profile,
		}))
	}

	svc := s3.New(sess, &aws.Config{
		Region: aws.String(conf.Region),
	})
	folder, err := tmpl.New(ctx).Apply(conf.Folder)
	if err != nil {
		return err
	}

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
				ACL:    aws.String(conf.ACL),
			})
			return err
		})
	}
	return g.Wait()
}
