package nodesea

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTargetNodeVersion(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		for _, v := range []string{
			"v22.20.0", "22.20.0", "v22.20.5", "v22.99.0",
			"v24.6.0", "v24.6.1", "v24.10.0",
			"v25.0.0", "v25.5.0", "v25.9.0", "v26.0.0",
		} {
			t.Run(v, func(t *testing.T) {
				require.NoError(t, ValidateTargetNodeVersion(v))
			})
		}
	})
	t.Run("rejected", func(t *testing.T) {
		for _, v := range []string{
			"v18.20.0",
			"v20.18.0",
			"v22.0.0", "v22.19.0",
			"v23.0.0", "v23.5.0",
			"v24.0.0", "v24.5.0",
		} {
			t.Run(v, func(t *testing.T) {
				err := ValidateTargetNodeVersion(v)
				require.Error(t, err)
				require.Contains(t, err.Error(), "v22.20.0 / v24.6.0 / v25.0.0")
			})
		}
	})
	t.Run("malformed", func(t *testing.T) {
		err := ValidateTargetNodeVersion("not-a-version")
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "parse"))
	})
	t.Run("empty", func(t *testing.T) {
		require.Error(t, ValidateTargetNodeVersion(""))
		require.Error(t, ValidateTargetNodeVersion("   "))
	})
}
