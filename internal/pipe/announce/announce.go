package announce

import (
	"context"
	"fmt"
)

// Pipe that announces releases on social media platforms
type Pipe struct{}

func (Pipe) String() string {
	return "announcing"
}

// Announcer should be implemented by pipes that want to announce a release.
type Announcer interface {
	fmt.Stringer

	Announce(ctx *context.Context) error
}
