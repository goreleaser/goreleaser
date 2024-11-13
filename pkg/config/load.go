package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

// SearchError will happen when the build.Glob multiplication
// meet an error while looking for matching pattern.
type SearchError struct {
	id      string
	pattern string
	err     error
}

func (e SearchError) Error() string {
	return fmt.Sprintf(
		"matching files search failed for %s with %s, please check your %s definition",
		logext.Keyword(fmt.Sprintf("pattern: %s", e.pattern)),
		logext.Keyword(fmt.Sprintf("err: %v", e.err)),
		logext.Keyword(fmt.Sprintf("build id: %s", e.id)),
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

	config.Builds, err = globBuilds(config.Builds, filepath.Glob)
	return config, err
}

// The globBuilds rebuild config.Builds
// Those builds which has no build.Main but build.Glob
// would be replicated\multiplied with matching pathes.
// New builds would have ID equal to matching path,
// main would be set to matching path with cut dir prefix,
// and binary would be set either
// to file name (if match not points to main.go - example match "./tools/cmd_delete.go" set binary "cmd_delete"),
// or base directory (example: match "./cmd/lake/main.go" set binary "lake")
func globBuilds(builds []Build, glob func(string) ([]string, error)) ([]Build, error) {
	a := []Build{}
	for _, build := range builds {
		if build.Glob == "" || build.Main != "" {
			a = append(a, build)
			continue
		}

		separator := string(filepath.Separator)
		dot_sep := "." + separator
		is_relative := strings.HasPrefix(build.Glob, dot_sep)

		// Ideally we have to respect build.Dir,
		// but unfortunatelly filepath.Join(build.Dir, build.Glob)
		// could end up having altered result
		// So as at the moment the build.Glob in config has to
		// account the build.Dir in its definition
		matches, err := glob(build.Glob)
		if err != nil {
			return a, SearchError{build.ID, build.Glob, err}
		}

		for _, match := range matches {
			clone := build // do we need here deep copy?
			clone.ID = match

			// recover relative prefix if it got missing
			if is_relative && !strings.HasPrefix(match, dot_sep) {
				match = dot_sep + match
			}

			dir := build.Dir
			if dir != "" && !strings.HasSuffix(dir, separator) {
				dir = dir + separator
			}
			main, ok := strings.CutPrefix(match, dir)
			if !ok {
				return a, SearchError{build.ID, build.Glob, fmt.Errorf("match(%s) must have build.Dir(%s) prefix", match, build.Dir)}
			}
			if is_relative && !strings.HasPrefix(main, dot_sep) {
				main = dot_sep + main
			}

			binary, found := strings.CutSuffix(main, separator+"main.go")
			if !found {
				binary, _ = strings.CutSuffix(filepath.Base(main), ".go")
			}
			binary = filepath.Base(binary)

			clone.Main = main
			clone.Binary = binary
			a = append(a, clone)
		}
	}
	return a, nil
}
