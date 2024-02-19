package config

import (
	"fmt"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/yaml"
)

// VersionError will happen if the goreleaser config file version does not
// match the current GoReleaser version.
type VersionError struct {
	current int
}

func (e VersionError) Error() string {
	return fmt.Sprintf(
		"only configurations files on %s are supported, yours is %s, please update your configuration",
		logext.Keyword("version: 1"),
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
	_ = yaml.Unmarshal(data, &versioned)
	if versioned.Version != 0 && versioned.Version != 1 {
		return config, VersionError{versioned.Version}
	}

	err = yaml.UnmarshalStrict(data, &config)
	return config, err
}
