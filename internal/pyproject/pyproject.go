package pyproject

import (
	"os"

	"github.com/BurntSushi/toml"
)

type PyProject struct {
	Project struct {
		Name           string
		Version        string
		RequiresPython string `toml:"requires-python"`
	}
}

func Open(name string) (PyProject, error) {
	var proj PyProject
	data, err := os.ReadFile(name)
	if err != nil {
		return proj, err
	}
	err = toml.Unmarshal(data, &proj)
	return proj, err
}
