package finalconfig

import (
	"io/ioutil"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	yaml "gopkg.in/yaml.v2"
)

type Pipe struct {
}

func (Pipe) String() string {
	return "writing the final config file to dist folder"
}

func (Pipe) Run(ctx *context.Context) (err error) {
	var path = filepath.Join(ctx.Config.Dist, "config.yaml")
	bts, err := yaml.Marshal(ctx.Config)
	if err != nil {
		return err
	}
	log.WithField("path", path).Info("writting")
	return ioutil.WriteFile(path, bts, 0644)
}
