package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateManifester(t *testing.T) {
	tests := []struct {
		use       string
		wantError string
	}{
		{use: "docker"},
		{use: "buildx", wantError: "docker manifest: invalid use: buildx, valid options are [docker]"},
	}

	for _, tt := range tests {
		t.Run(tt.use, func(t *testing.T) {
			err := validateManifester(tt.use)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
