package rust

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Cargo struct {
	Workspace struct {
		Members []string
	}
}

func parseCargo(path string) (Cargo, error) {
	var cargo Cargo
	bts, err := os.ReadFile(path)
	if err != nil {
		return cargo, err
	}
	err = toml.Unmarshal(bts, &cargo)
	return cargo, err
}
