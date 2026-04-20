package nodesea

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/pkg/codesign"
	cstypes "github.com/blacktop/go-macho/pkg/codesign/types"
)

// AdHocSignMachO writes an ad-hoc code signature to the Mach-O file at
// path. macOS arm64 kernels refuse to exec unsigned binaries, so this is
// mandatory after [InjectMachO]. x86_64 doesn't strictly require it, but
// signing is safe and keeps the binary uniform across architectures.
//
// id is used as the code-signing identifier; if empty, the binary's
// filename (without extension) is used.
func AdHocSignMachO(path, id string) error {
	if id == "" {
		base := filepath.Base(path)
		id = strings.TrimSuffix(base, filepath.Ext(base))
	}

	f, err := macho.Open(path)
	if err != nil {
		return fmt.Errorf("nodesea: codesign: open %s: %w", path, err)
	}
	defer f.Close()

	cfg := &codesign.Config{
		ID:    id,
		Flags: cstypes.ADHOC | cstypes.LINKER_SIGNED,
	}
	if err := f.CodeSign(cfg); err != nil {
		return fmt.Errorf("nodesea: codesign: sign %s: %w", path, err)
	}
	if err := f.Save(path); err != nil {
		return fmt.Errorf("nodesea: codesign: save %s: %w", path, err)
	}
	return nil
}
