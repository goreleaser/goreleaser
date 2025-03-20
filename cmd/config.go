package cmd

import (
	"errors"
	"io/fs"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

var proExplain = `Your configuration is for GoReleaser Pro.
You are currently using GoReleaser OSS, so all the Pro-only features will be ignored.
Use GoReleaser Pro to enable all the features.`

func loadConfig(strict bool, path string) (config.Project, error) {
	p, path, err := loadConfigCheck(path)
	if err == nil {
		log.WithField("path", path).Debug("using configuration")
	}
	if errors.Is(err, config.ErrProConfig) {
		if strict {
			return p, err
		}
		log.WithField("explanation", proExplain).
			Warnf(
				"%s %s",
				logext.Warning("your configuration specifies"),
				logext.Keyword("pro: true"),
			)
		return p, nil
	}
	return p, err
}

func loadConfigCheck(path string) (config.Project, string, error) {
	if path == "-" {
		p, err := config.LoadReader(os.Stdin)
		return p, path, err
	}
	if path != "" {
		p, err := config.Load(path)
		return p, path, err
	}
	for _, f := range [6]string{
		".config/goreleaser.yml",
		".config/goreleaser.yaml",
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
	log.Warn(logext.Warning("could not find a configuration file, using defaults..."))
	return config.Project{}, "", nil
}
