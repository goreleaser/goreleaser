package blob

import (
	"errors"

	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe for blobs.
type Pipe struct{}

// String returns the description of the pipe.
func (Pipe) String() string                 { return "blobs" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Blobs) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Blobs {
		blob := &ctx.Config.Blobs[i]
		if blob.Bucket == "" || blob.Provider == "" {
			return errors.New("bucket or provider cannot be empty")
		}
		if blob.Directory == "" {
			blob.Directory = "{{ .ProjectName }}/{{ .Tag }}"
		}

		switch blob.ContentDisposition {
		case "":
			blob.ContentDisposition = "attachment;filename={{.Filename}}"
		case "-":
			blob.ContentDisposition = ""
		}
	}
	return nil
}

// Publish to specified blob bucket url.
func (Pipe) Publish(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, conf := range ctx.Config.Blobs {
		g.Go(func() error {
			b, err := tmpl.New(ctx).Bool(conf.Disable)
			if err != nil {
				return err
			}
			if b {
				return pipe.Skip("configuration is disabled")
			}
			return doUpload(ctx, conf)
		})
	}
	return g.Wait()
}
