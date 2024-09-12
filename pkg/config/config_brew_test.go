package config

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/yaml"
	"github.com/stretchr/testify/require"
)

func TestBrewDependencies_justString(t *testing.T) {
	var actual Homebrew
	err := yaml.UnmarshalStrict([]byte(`dependencies: ['xclip']`), &actual)
	require.NoError(t, err)
	require.Equal(t, Homebrew{
		Dependencies: []HomebrewDependency{
			{Name: "xclip"},
		},
	}, actual)
}

func TestBrewDependencies_full(t *testing.T) {
	var actual Homebrew
	err := yaml.UnmarshalStrict([]byte(`dependencies:
- name: xclip
  os: linux
  version: '~> v1'
  type: optional`), &actual)
	require.NoError(t, err)
	require.Equal(t, Homebrew{
		Dependencies: []HomebrewDependency{
			{
				Name:    "xclip",
				Type:    "optional",
				Version: "~> v1",
				OS:      "linux",
			},
		},
	}, actual)
}
