package config

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/yaml"
)

// VersionError will happen if the goreleaser config file version does not
// match the current GoReleaser version.
type VersionError struct {
	current int
}

// ErrProConfig happens if the configuration failed to load strictly, but
// there's a 'pro: true' field in it, so we just allow anything.
var ErrProConfig = errors.New("you are using a GoReleaser Pro configuration file with GoReleaser OSS")

func (e VersionError) Error() string {
	return fmt.Sprintf(
		"only %s configuration files are supported, yours is %s, please update your configuration",
		logext.Keyword("version: 2"),
		logext.Keyword(fmt.Sprintf("version: %d", e.current)),
	)
}

// Load config file.
func Load(file string) (Project, error) {
	f, err := os.Open(file) // #nosec
	if err != nil {
		return Project{}, err
	}
	defer f.Close()
	return LoadReader(f)
}

// LoadReader config via io.Reader.
func LoadReader(fd io.Reader) (config Project, err error) {
	data, err := io.ReadAll(fd)
	if err != nil {
		return config, err
	}

	var versioned Versioned
	if err := yaml.Unmarshal(data, &versioned); err != nil {
		return config, err
	}

	validVersion := versioned.Version == 2
	if !validVersion {
		log.Warn(VersionError{versioned.Version}.Error())
	}

	err = yaml.UnmarshalStrict(data, &config)
	if err != nil && !validVersion {
		return config, VersionError{versioned.Version}
	}
	if err != nil && versioned.Pro {
		err2 := yaml.Unmarshal(data, &config)
		if err2 == nil {
			err = errors.Join(err, ErrProConfig)
		}
	}
	return config, err
}
