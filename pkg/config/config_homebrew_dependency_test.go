package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalHomebrewDependency(t *testing.T) {
	t.Run("string arr", func(t *testing.T) {
		conf := `
brews:
- name: foo
  dependencies:
  - foo
  - bar
`
		buf := strings.NewReader(conf)
		prop, err := LoadReader(buf)

		require.NoError(t, err)
		require.Equal(t, []HomebrewDependency{
			{
				Name: "foo",
			}, {
				Name: "bar",
			},
		}, prop.Brews[0].Dependencies)
	})

	t.Run("mixed", func(t *testing.T) {
		conf := `
brews:
- name: foo
  dependencies:
  - name: foo
  - bar
  - name: foobar
    type: optional
`
		buf := strings.NewReader(conf)
		prop, err := LoadReader(buf)

		require.NoError(t, err)
		require.Equal(t, []HomebrewDependency{
			{
				Name: "foo",
			}, {
				Name: "bar",
			}, {
				Name: "foobar",
				Type: "optional",
			},
		}, prop.Brews[0].Dependencies)
	})

	t.Run("mixed", func(t *testing.T) {
		conf := `
brews:
- name: foo
  dependencies:
  - name: foo
  - namer: bar
  - asdda
`
		buf := strings.NewReader(conf)
		_, err := LoadReader(buf)

		require.EqualError(t, err, "yaml: unmarshal errors:\n  line 6: field namer not found in type config.homebrewDependency")
	})
}
