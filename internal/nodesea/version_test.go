package nodesea

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveVersion(t *testing.T) {
	t.Run("from package.json engines.node pinned", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":"25.5.0"}}`), 0o644))
		v, err := ResolveVersion(dir)
		require.NoError(t, err)
		require.Equal(t, "v25.5.0", v)
	})

	t.Run("from package.json engines.node range", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":">=25 <26"}}`), 0o644))
		v, err := ResolveVersion(dir)
		require.NoError(t, err)
		// Resolved version comes from the embedded release index; just
		// assert it is a v25.x release.
		require.True(t, strings.HasPrefix(v, "v25."), "got %q", v)
	})

	t.Run("nothing set", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ResolveVersion(dir)
		require.Error(t, err)
		require.True(t, errors.Is(err, errNoVersion))
	})

	t.Run("range with no match", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":"^99"}}`), 0o644))
		_, err := ResolveVersion(dir)
		require.Error(t, err)
	})
}
