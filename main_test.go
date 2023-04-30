package main

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	for name, tt := range map[string]struct {
		version, commit, date, builtBy string
	}{
		"all empty": {},
		"complete": {
			version: "1.2.3",
			date:    "12/12/12",
			commit:  "aaaa",
			builtBy: "me",
		},
		"only version": {
			version: "1.2.3",
		},
		"version and date": {
			version: "1.2.3",
			date:    "12/12/12",
		},
		"version, date, built by": {
			version: "1.2.3",
			date:    "12/12/12",
			builtBy: "me",
		},
	} {
		tt := tt
		t.Run(name, func(t *testing.T) {
			v := buildVersion(tt.version, tt.commit, tt.date, tt.builtBy)
			out, err := v.JSONString()
			require.NoError(t, err)

			golden.RequireEqualJSON(t, []byte(out))
		})
	}
}
