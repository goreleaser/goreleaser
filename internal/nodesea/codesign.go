package nodesea

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrCodeSignUnavailable is returned by adHocSignMachO when codesign(1)
// is not present on PATH (e.g. cross-compiling a darwin SEA from a
// linux host). Callers may treat this as a soft failure: the produced
// binary is well-formed but kernel-rejected on macOS until re-signed.
var ErrCodeSignUnavailable = errors.New("nodesea: codesign(1) not available; output left unsigned")

// codeSignBinary names the executable used to ad-hoc sign Mach-O
// outputs. Variable so tests can stub or skip it.
//
//nolint:gochecknoglobals
var codeSignBinary = "codesign"

// runCodeSign is the executor for codesign(1). Variable so tests can
// record argv without invoking the real binary.
//
//nolint:gochecknoglobals
var runCodeSign = func(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, codeSignBinary, args...)
	return cmd.CombinedOutput()
}

// adHocSignMachO applies an ad-hoc signature to the Mach-O at path via
// `codesign --sign - --force --identifier <id> <path>`.
//
// We use codesign(1) (Apple's signer) because `node --build-sea`
// produces a Mach-O with a placeholder LC_CODE_SIGNATURE load command
// that points at end-of-file with no signature bytes appended; an
// in-process signer that does not recognize that placeholder will
// corrupt the layout. codesign(1) handles this correctly.
//
// When codesign(1) is not on PATH (typical on non-darwin build hosts
// cross-compiling for darwin), this returns ErrCodeSignUnavailable
// without modifying path. Callers that need a runnable binary on macOS
// must re-sign via goreleaser's signs: pipe on a darwin runner.
func adHocSignMachO(ctx context.Context, path, id string) error {
	if _, err := exec.LookPath(codeSignBinary); err != nil {
		return ErrCodeSignUnavailable
	}
	out, err := runCodeSign(ctx, "--sign", "-", "--force", "--identifier", id, path)
	if err != nil {
		return fmt.Errorf("codesign %s: %w (output: %s)", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}
