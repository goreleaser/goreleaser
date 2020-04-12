package before

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	for _, tc := range [][]string{
		nil,
		{},
		{"go version"},
		{"go version", "go list"},
		{`bash -c "go version; echo \"lala spaces and such\""`},
	} {
		ctx := context.New(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		require.NoError(t, Pipe{}.Run(ctx))
	}
}

func TestRunPipeInvalidCommand(t *testing.T) {
	for _, tc := range [][]string{
		{`bash -c "echo \"unterminated command\"`},
	} {
		ctx := context.New(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		require.Error(t, Pipe{}.Run(ctx))
	}
}

func TestRunPipeFail(t *testing.T) {
	for _, tc := range [][]string{
		{"go tool foobar"},
	} {
		ctx := context.New(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		require.Error(t, Pipe{}.Run(ctx))
	}
}

func TestRunWithEnv(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	require.NoError(t, os.Remove(f.Name()))
	defer os.Remove(f.Name())
	require.NoError(t, Pipe{}.Run(context.New(
		config.Project{
			Env: []string{
				"TEST_FILE=" + f.Name(),
			},
			Before: config.Before{
				Hooks: []string{"touch {{ .Env.TEST_FILE }}"},
			},
		},
	)))
	require.FileExists(t, f.Name())
}

func TestInvalidTemplate(t *testing.T) {
	require.EqualError(t, Pipe{}.Run(context.New(
		config.Project{
			Before: config.Before{
				Hooks: []string{"touch {{ .fasdsd }"},
			},
		},
	)), `template: tmpl:1: unexpected "}" in operand`)
}
