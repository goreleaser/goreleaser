// Package packagejson provides NodeJS/Bun package.json parsing.
package packagejson

import (
	"encoding/json"
	"os"
	"strings"
)

type Package struct {
	Name    string  `json:"name"`
	Module  string  `json:"module"`
	Type    string  `json:"type"`
	Engines Engines `json:"engines"`
	Scripts Scripts `json:"scripts"`

	DevDependencies map[string]string `json:"devDependencies"` //nolint:tagliatelle
}

// Engines mirrors the `engines` map in package.json. Only the fields
// goreleaser cares about are surfaced.
type Engines struct {
	Node string `json:"node"`
}

// NodeRange returns the trimmed engines.node value. Empty when unset.
func (e Engines) NodeRange() string {
	return strings.TrimSpace(e.Node)
}

// Scripts mirrors the `scripts` map in package.json. Only the fields
// goreleaser cares about are surfaced.
type Scripts struct {
	Build string `json:"build"`
}

// HasBuild reports whether `scripts.build` is set to a non-empty
// command.
func (s Scripts) HasBuild() bool {
	return strings.TrimSpace(s.Build) != ""
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

// OpenOrEmpty parses the file at name and returns the result, or a
// zero Package when the file does not exist. Other errors (parse,
// permission, etc.) are returned unchanged.
func OpenOrEmpty(name string) (Package, error) {
	pkg, err := Open(name)
	if err != nil && os.IsNotExist(err) {
		return Package{}, nil
	}
	return pkg, err
}
