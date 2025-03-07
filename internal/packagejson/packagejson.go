// Package packagejson provides NodeJS/Bun package.json parsing.
package packagejson

import (
	"encoding/json"
	"os"
)

type Package struct {
	Name   string `json:"name"`
	Module string `json:"module"`
	Type   string `json:"type"`

	DevDependencies map[string]string `json:"devDependencies"` //nolint:tagliatelle
}

func (p Package) IsBun() bool {
	_, ok := p.DevDependencies["@types/bun"]
	return ok
}

// Open and parse the given file name.
func Open(name string) (Package, error) {
	var pkg Package
	bts, err := os.ReadFile(name)
	if err != nil {
		return pkg, err
	}
	err = json.Unmarshal(bts, &pkg)
	return pkg, err
}
