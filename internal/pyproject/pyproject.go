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
			Name     string
			Version  string
			Packages []any
		}
	}
}

func (p PyProject) IsPoetry() bool {
	return len(p.Tool.Poetry.Packages) > 0
}

// Name returns the project name.
func (p PyProject) Name() string {
	return normalizeName(p.Project.Name)
}

func normalizeName(name string) string {
	var b strings.Builder
	previousWasSeparator := false
	for _, r := range strings.ToLower(name) {
		switch r {
		case '-', '_', '.':
			if !previousWasSeparator {
				b.WriteRune('_')
			}
			previousWasSeparator = true
		default:
			b.WriteRune(r)
			previousWasSeparator = false
		}
	}
	return b.String()
}

// Open opens and parses a pyproject.toml file.
func Open(name string) (PyProject, error) {
	var proj PyProject
	data, err := os.ReadFile(name)
	if err != nil {
		return proj, err
	}
	if err := toml.Unmarshal(data, &proj); err != nil {
		return proj, err
	}
	if proj.Project.Name == "" {
		proj.Project.Name = proj.Tool.Poetry.Name
	}
	if proj.Project.Version == "" {
		proj.Project.Version = proj.Tool.Poetry.Version
	}
	return proj, nil
}
