package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlobBuilds(t *testing.T) {
	testCases := []struct {
		desc string
		in   []Build
		glob map[string]struct {
			matches []string
			err     error
		}
		exp []Build
		err error
	}{
		{
			desc: "bypass all builds",
			in: []Build{
				{
					Main: "./cmd/aaa/main.go",
				},
				{
					Main: "./cmd/bbb/main.go",
				},
			},
			exp: []Build{
				{
					Main: "./cmd/aaa/main.go",
				},
				{
					Main: "./cmd/bbb/main.go",
				},
			},
			err: nil,
		},
		{
			desc: "basic expand 1",
			in: []Build{
				{
					Main: "./cmd/aaa/main.go",
				},
				{
					Dir:  "",
					Glob: "./tools/*/main.go",
				},
			},
			glob: map[string]struct {
				matches []string
				err     error
			}{
				"./tools/*/main.go": {
					[]string{"./tools/bbb/main.go", "./tools/ccc/main.go"},
					nil,
				},
			},
			exp: []Build{
				{
					Main: "./cmd/aaa/main.go",
				},
				{
					Glob:   "./tools/*/main.go",
					ID:     "./tools/bbb/main.go",
					Dir:    "",
					Main:   "./tools/bbb/main.go",
					Binary: "bbb",
				},
				{
					Glob:   "./tools/*/main.go",
					ID:     "./tools/ccc/main.go",
					Dir:    "",
					Main:   "./tools/ccc/main.go",
					Binary: "ccc",
				},
			},
			err: nil,
		},
		{
			desc: "complex expand 2 - relative open dir and lost prefix",
			in: []Build{
				{
					Main: "./cmd/aaa/main.go",
				},
				{
					Dir:  "./tools",
					Glob: "./tools/*/main.go",
				},
			},
			glob: map[string]struct {
				matches []string
				err     error
			}{
				"./tools/*/main.go": {
					[]string{"tools/bbb/main.go", "tools/ccc/main.go"},
					nil,
				},
			},
			exp: []Build{
				{
					Main: "./cmd/aaa/main.go",
				},
				{
					Glob:   "./tools/*/main.go",
					ID:     "tools/bbb/main.go",
					Dir:    "./tools",
					Main:   "./bbb/main.go",
					Binary: "bbb",
				},
				{
					Glob:   "./tools/*/main.go",
					ID:     "tools/ccc/main.go",
					Dir:    "./tools",
					Main:   "./ccc/main.go",
					Binary: "ccc",
				},
			},
			err: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, err := globBuilds(tC.in, func(glob string) ([]string, error) {
				s, ok := tC.glob[glob]
				require.True(t, ok)
				return s.matches, s.err
			})

			require.Equal(t, tC.exp, out)
			require.Equal(t, tC.err, err)
		})
	}
}
