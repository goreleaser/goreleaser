package node

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
