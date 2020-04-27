package cmd

import (
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

func loadConfig(path string) (config.Project, error) {
	if path != "" {
		return config.Load(path)
	}
	for _, f := range [4]string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		proj, err := config.Load(f)
		if err != nil && os.IsNotExist(err) {
			continue
		}
		return proj, err
	}
	// the user didn't specify a config file and the known possible file names
	// don't exist, so, return an empty config and a nil err.
	log.Warn("could not find a config file, using defaults...")
	return config.Project{}, nil
}
