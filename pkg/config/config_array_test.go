package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

type Unmarshaled struct {
	Strings StringArray `yaml:",omitempty"`
	Flags   FlagArray   `yaml:",omitempty"`
}

type yamlUnmarshalTestCase struct {
	yaml     string
	expected Unmarshaled
	err      string
}

var stringArrayTests = []yamlUnmarshalTestCase{
	{
		"",
		Unmarshaled{},
		"",
	},
	{
		"strings: []",
		Unmarshaled{
			Strings: StringArray{},
		},
		"",
	},
	{
		"strings: [one two, three]",
		Unmarshaled{
			Strings: StringArray{"one two", "three"},
		},
		"",
	},
	{
		"strings: one two",
		Unmarshaled{
			Strings: StringArray{"one two"},
		},
		"",
	},
	{
		"strings: {key: val}",
		Unmarshaled{},
		"yaml: unmarshal errors:\n  line 1: cannot unmarshal !!map into string",
	},
}

var flagArrayTests = []yamlUnmarshalTestCase{
	{
		"",
		Unmarshaled{},
		"",
	},
	{
		"flags: []",
		Unmarshaled{
			Flags: FlagArray{},
		},
		"",
	},
	{
		"flags: [one two, three]",
		Unmarshaled{
			Flags: FlagArray{"one two", "three"},
		},
		"",
	},
	{
		"flags: one two",
		Unmarshaled{
			Flags: FlagArray{"one", "two"},
		},
		"",
	},
	{
		"flags: {key: val}",
		Unmarshaled{},
		"yaml: unmarshal errors:\n  line 1: cannot unmarshal !!map into string",
	},
}

func TestStringArray(t *testing.T) {
	for _, testCase := range stringArrayTests {
		var actual Unmarshaled

		err := yaml.UnmarshalStrict([]byte(testCase.yaml), &actual)
		if testCase.err == "" {
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, actual)
		} else {
			assert.EqualError(t, err, testCase.err)
		}
	}
}

// func TestStringArrayFailure(t *testing.T) {
// var source = `
// strings:
// key: val
// `

// var actual Unmarshaled
// err := yaml.UnmarshalStrict([]byte(source), &actual)
// // assert.EqualError(t, err, )
// }

func TestFlagArray(t *testing.T) {
	for _, testCase := range flagArrayTests {
		var actual Unmarshaled

		err := yaml.UnmarshalStrict([]byte(testCase.yaml), &actual)
		if testCase.err == "" {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, testCase.err)
		}
		assert.Equal(t, testCase.expected, actual)
	}
}
