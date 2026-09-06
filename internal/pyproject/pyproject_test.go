package pyproject

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	proj, err := Open("./testdata/pyproject.toml")
	require.NoError(t, err)
	require.Equal(t, "python-test", proj.Project.Name)
	require.Equal(t, "0.1.0", proj.Project.Version)
	require.False(t, proj.IsPoetry())
}

func TestOpenLegacyPoetry(t *testing.T) {
	proj, err := Open("../builders/poetry/testdata/pyproject.toml")
	require.NoError(t, err)
	require.Equal(t, "testdata", proj.Project.Name)
	require.Equal(t, "0.1.0", proj.Project.Version)
}

func TestOpenProjectMetadataPrecedence(t *testing.T) {
	proj, err := Open("./testdata/project-and-poetry-pyproject.toml")
	require.NoError(t, err)
	require.Equal(t, "project-name", proj.Project.Name)
	require.Equal(t, "1.2.3", proj.Project.Version)
}

func TestOpenError(t *testing.T) {
	_, err := Open("./testdata/nope.toml")
	require.Error(t, err)
}

func TestName(t *testing.T) {
	proj, err := Open("./testdata/pyproject.toml")
	require.NoError(t, err)
	require.Equal(t, "python_test", proj.Name())
}

func TestNormalizeName(t *testing.T) {
	for name, expected := range map[string]string{
		"My..Pkg":          "my_pkg",
		"my-pkg":           "my_pkg",
		"my__pkg":          "my_pkg",
		"my-_.pkg":         "my_pkg",
		"python-test":      "python_test",
		"already_normal":   "already_normal",
		"package.with.dot": "package_with_dot",
	} {
		t.Run(name, func(t *testing.T) {
			var proj PyProject
			proj.Project.Name = name
			require.Equal(t, expected, proj.Name())
		})
	}
}

func TestIsPoetry(t *testing.T) {
	proj, err := Open("./testdata/poetry-pyproject.toml")
	require.NoError(t, err)
	require.True(t, proj.IsPoetry())
}
