package publish

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/pipe/docker"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
)

// Pipe that sets the defaults
type Pipe struct{}

func (Pipe) String() string {
	return "publishing artifacts"
}

type Publisher interface {
	fmt.Stringer

	// Default sets the configuration defaults
	Publish(ctx *context.Context) error
}

var publishers = []Publisher{
	docker.Pipe{},
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	for _, publisher := range publishers {
		if err := publisher.Publish(ctx); err != nil {
			return errors.Wrapf(err, "failed to publish artifacts for %s", publisher.String())
		}
	}
	return nil
}
