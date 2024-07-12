package config

import (
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

func (e VersionError) Error() string {
	return fmt.Sprintf(
		"only configurations files on %s are supported, yours is %s, please update your configuration",
		logext.Keyword("version: 2"),
		logext.Keyword(fmt.Sprintf("version: %d", e.current)),
	)
}

// Load config file.
func Load(file string) (config Project, err error) {
	f, err := os.Open(file) // #nosec
	if err != nil {
		return
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
	return config, err
}
