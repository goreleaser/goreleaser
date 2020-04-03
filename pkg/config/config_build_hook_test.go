package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestBuildHook_justString(t *testing.T) {
	var actual HookConfig

	err := yaml.UnmarshalStrict([]byte(`pre: ./script.sh`), &actual)
	assert.NoError(t, err)
	assert.Equal(t, BuildHook{
		Cmd: "./script.sh",
		Env: nil,
	}, actual.Pre[0])
}

func TestBuildHook_stringCmds(t *testing.T) {
	var actual HookConfig

	err := yaml.UnmarshalStrict([]byte(`pre:
 - ./script.sh
 - second-script.sh
`), &actual)
	assert.NoError(t, err)

	assert.Equal(t, BuildHooks{
		{
			Cmd: "./script.sh",
			Env: nil,
		},
		{
			Cmd: "second-script.sh",
			Env: nil,
		},
	}, actual.Pre)
}

func TestBuildHook_complex(t *testing.T) {
	var actual HookConfig

	err := yaml.UnmarshalStrict([]byte(`pre:
 - cmd: ./script.sh
   env:
    - TEST=value
`), &actual)
	assert.NoError(t, err)
	assert.Equal(t, BuildHook{
		Cmd: "./script.sh",
		Env: []string{"TEST=value"},
	}, actual.Pre[0])
}
