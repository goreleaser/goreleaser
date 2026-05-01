package nodesea

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignMachO_RecordsArgs(t *testing.T) {
	prev := signMachO
	t.Cleanup(func() { signMachO = prev })

	var gotPath, gotID string
	signMachO = func(path, id string) error {
		gotPath, gotID = path, id
		return nil
	}

	require.NoError(t, signMachO("/path/to/bin", "my.bundle.id"))
	require.Equal(t, "/path/to/bin", gotPath)
	require.Equal(t, "my.bundle.id", gotID)
}

func TestSignMachO_PropagatesError(t *testing.T) {
	prev := signMachO
	t.Cleanup(func() { signMachO = prev })

	signMachO = func(string, string) error {
		return errors.New("quill: bad bag o' bits")
	}

	err := signMachO("/p", "id")
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad bag o' bits")
}

// TestSignMachO_RealQuill exercises the live quill signer on a tiny
// fake non-Mach-O file to confirm the wiring fails with a real quill
// error (not a panic) when the input isn't a valid Mach-O. Ad-hoc
// signing of a real Mach-O is covered by quill's own test suite.
func TestSignMachO_RealQuill(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "not-a-macho")
	require.NoError(t, os.WriteFile(bin, []byte("not a mach-o file"), 0o644))

	err := signMachO(bin, "id")
	require.Error(t, err)
}
