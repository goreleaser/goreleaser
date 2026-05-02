package node

import (
	"fmt"

	"github.com/goreleaser/quill/quill"
)

// signMachO ad-hoc signs the Mach-O at path with identifier id using
// quill (pure-Go Mach-O signer). Works on any host OS — no codesign(1)
// dependency, so cross-compiling darwin SEAs from linux/windows hosts
// produces a kernel-loadable binary.
//
// `node --build-sea` leaves a placeholder LC_CODE_SIGNATURE pointing at
// end-of-file with no signature bytes appended. quill's signSingleBinary
// calls RemoveSigningContent before signing, so the placeholder is
// stripped and replaced with a valid ad-hoc CMS superblob.
//
// Ad-hoc only — no developer cert involved. Users with a Developer ID
// can layer real signing on top via the signs: pipe; quill removes the
// ad-hoc signature first there too.
func signMachO(path, id string) error {
	cfg, err := quill.NewSigningConfigFromPEMs(path, "", "", "", false)
	if err != nil {
		return fmt.Errorf("node: quill config for %s: %w", path, err)
	}
	cfg.WithIdentity(id)
	if err := quill.Sign(*cfg); err != nil {
		return fmt.Errorf("node: ad-hoc sign %s: %w", path, err)
	}
	return nil
}
