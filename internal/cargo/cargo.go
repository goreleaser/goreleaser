// Package cargo provides Rust's Cargo.toml parsing.
package cargo

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Cargo a parsed Cargo.toml.
type Cargo struct {
	Package struct {
		Name string
	}
	Workspace struct {
		Members []string
	}
}

// Open and parse the given file name.
func Open(name string) (Cargo, error) {
	var cargo Cargo
	bts, err := os.ReadFile(name)
	if err != nil {
		return cargo, err
	}
	err = toml.Unmarshal(bts, &cargo)
	return cargo, err
}
