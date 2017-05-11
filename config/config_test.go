package config

import (
        "testing"
	"strings"

        "github.com/stretchr/testify/assert"
)

func TestRepo(t *testing.T) {
	var assert = assert.New(t)
	r := Repo{"goreleaser", "godownloader"}
	assert.Equal("goreleaser/godownloader", r.String(), "not equal")
}

func TestLoadReader(t *testing.T) {
	var conf =`
homepage: &homepage http://goreleaser.github.io
fpm:
  homepage: *homepage
`
        var assert = assert.New(t)
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	assert.Nil(err)
	assert.Equal("http://goreleaser.github.io", prop.FPM.Homepage, "yaml did not load correctly")
}
