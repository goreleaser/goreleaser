// Package pyproject provides a way to parse a pyproject.toml file.
package pyproject

import (
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// PyProject represents a pyproject.toml file.
type PyProject struct {
	Project struct {
		Name    string
		Version string
	}
	Tool struct {
		Poetry struct {
			Packages []any
		}
	}
}

func (p PyProject) IsPoetry() bool {
	return len(p.Tool.Poetry.Packages) > 0
}

// Name returns the project name.
func (p PyProject) Name() string {
	return strings.ReplaceAll(p.Project.Name, "-", "_")
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
