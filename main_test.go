package main

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	os := fmt.Sprintf("\ngoos: %s\ngoarch: %s", runtime.GOOS, runtime.GOARCH)
	for name, tt := range map[string]struct {
		version, commit, date, builtBy string
		out                            string
	}{
		"all empty": {
			out: os + website,
		},
		"complete": {
			version: "1.2.3",
			date:    "12/12/12",
			commit:  "aaaa",
			builtBy: "me",
			out:     "1.2.3\ncommit: aaaa\nbuilt at: 12/12/12\nbuilt by: me" + os + website,
		},
		"only version": {
			version: "1.2.3",
			out:     "1.2.3" + os + website,
		},
		"version and date": {
			version: "1.2.3",
			date:    "12/12/12",
			out:     "1.2.3\nbuilt at: 12/12/12" + os + website,
		},
		"version, date, built by": {
			version: "1.2.3",
			date:    "12/12/12",
			builtBy: "me",
			out:     "1.2.3\nbuilt at: 12/12/12\nbuilt by: me" + os + website,
		},
	} {
		tt := tt
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.out, buildVersion(tt.version, tt.commit, tt.date, tt.builtBy))
		})
	}
}
