package nodesea

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdHocSignMachO_CodesignMissing(t *testing.T) {
	// Force lookups to miss by emptying PATH.
	t.Setenv("PATH", filepath.Join(t.TempDir(), "definitely-empty"))

	err := adHocSignMachO(t.Context(), "/some/path", "id")
	require.ErrorIs(t, err, ErrCodeSignUnavailable)
}

func TestAdHocSignMachO_RecordsArgv(t *testing.T) {
	// Stage a fake codesign on PATH so LookPath succeeds.
	bin := filepath.Join(t.TempDir(), "codesign")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", filepath.Dir(bin))

	prev := runCodeSign
	t.Cleanup(func() { runCodeSign = prev })

	var got []string
	runCodeSign = func(_ context.Context, args ...string) ([]byte, error) {
		got = args
		return nil, nil
	}

	require.NoError(t, adHocSignMachO(t.Context(), "/path/to/bin", "my.bundle.id"))
	require.Equal(t,
		[]string{"--sign", "-", "--force", "--identifier", "my.bundle.id", "/path/to/bin"},
		got)
}

func TestAdHocSignMachO_CodesignFailure(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "codesign")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", filepath.Dir(bin))

	prev := runCodeSign
	t.Cleanup(func() { runCodeSign = prev })

	runCodeSign = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("codesign: bad bag o' bits"), errors.New("exit status 1")
	}

	err := adHocSignMachO(t.Context(), "/p", "id")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrCodeSignUnavailable)
	require.Contains(t, err.Error(), "bad bag o' bits")
}

// TestBuildViaBuildSEA_Darwin_NoCodesign verifies the "cross-compile from
// linux for darwin" fallback: when codesign(1) is missing the build
// completes successfully and leaves the binary unsigned.
func TestBuildViaBuildSEA_Darwin_NoCodesign(t *testing.T) {
	t.Setenv("PATH", filepath.Join(t.TempDir(), "no-codesign-here"))

	const version = "v22.20.0"
	target := Target("darwin-arm64")
	stageTargetNode(t, version, target)

	mainPath := filepath.Join(t.TempDir(), "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))

	outPath := filepath.Join(t.TempDir(), "myapp")
	stubRunBuildSEA(t, nil)

	require.NoError(t, BuildViaBuildSEA(t.Context(), BuildOptions{
		BuildToolNode: "/fake/node",
		Target:        target,
		Version:       version,
		MainJS:        mainPath,
		OutPath:       outPath,
	}))

	require.FileExists(t, outPath)
}
