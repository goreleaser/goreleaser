package pipeline

import "github.com/goreleaser/releaser/config"

type Pipe interface {
	Name() string
	Work(config config.ProjectConfig) error
}
