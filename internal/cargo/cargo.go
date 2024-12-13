package cargo

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Cargo struct {
	Package struct {
		Name string
	}
	Workspace struct {
		Members []string
	}
}

func Open(path string) (Cargo, error) {
	var cargo Cargo
	bts, err := os.ReadFile(path)
	if err != nil {
		return cargo, err
	}
	err = toml.Unmarshal(bts, &cargo)
	return cargo, err
}
