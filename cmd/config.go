package cmd

import (
	"errors"
	"io/fs"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

func loadConfig(path string) (config.Project, error) {
	p, _, err := loadConfigCheck(path)
	return p, err
}

func loadConfigCheck(path string) (config.Project, string, error) {
	if path == "-" {
		log.Info("loading config from stdin")
		p, err := config.LoadReader(os.Stdin)
		return p, path, err
	}
	if path != "" {
		p, err := config.Load(path)
		return p, path, err
	}
	for _, f := range [4]string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		proj, err := config.Load(f)
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return proj, f, err
	}
	// the user didn't specify a config file and the known possible file names
	// don't exist, so, return an empty config and a nil err.
	log.Warn("could not find a configuration file, using defaults...")
	return config.Project{}, "", nil
}
