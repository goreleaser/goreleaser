package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	for _, v := range []string{
		"",
		"1.2.3",
		"1.2.3-rc.1+build.5",
		"1.2.3-SNAPSHOT-abc1234",
		"0.0.0_next",
	} {
		require.NoError(t, Check(v), v)
	}
	for _, v := range []string{
		`1.0.0"; system("id"); x = "`,
		`1.0.0#{system("id")}`,
		"1.0.0;id",
		"1.0.0$(id)",
		"1.0.0`id`",
		"1.0.0 ${builtins.currentTime}",
		"1.0.0\nid",
		"feature/foo",
		"0.2.0%(echo id)",
		"0.2.0%{lua:print(1)}",
	} {
		require.ErrorContains(t, Check(v), "invalid characters", v)
	}
}
