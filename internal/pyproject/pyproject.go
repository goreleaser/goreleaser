// Package pyproject provides a way to parse a pyproject.toml file.
package pyproject

import (
	"os"

	"github.com/BurntSushi/toml"
)

// PyProject represents a pyproject.toml file.
type PyProject struct {
	Project struct {
		Name           string
		Version        string
		RequiresPython string `toml:"requires-python"`
	}
}

// Open opens and parses a pyproject.toml file.
func Open(name string) (PyProject, error) {
	var proj PyProject
	data, err := os.ReadFile(name)
	if err != nil {
		return proj, err
	}
	err = toml.Unmarshal(data, &proj)
	return proj, err
}
