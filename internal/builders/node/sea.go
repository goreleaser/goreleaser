package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
	"github.com/goreleaser/quill/quill"
)

// buildSEA produces a Single Executable Application at outPath for
// target by invoking `node --build-sea sea-config.json`, where
// sea-config.json points `executable` at the per-target Node binary
// downloaded for the version declared in <buildDir>/package.json's
// engines.node. mainPath is the absolute path to the user's
// entrypoint JS file. The `node` binary on PATH is used as-is; if it
// cannot drive `--build-sea` the underlying command failure is
// returned to the caller.
//
// If a sea-config.json exists in buildDir, its user-tunable fields
// are merged into the rendered config (relative `assets` paths are
// resolved against buildDir so they keep working from the dist
// directory). Goreleaser-owned fields (`output`, `executable`,
// `main`) always win.
//
// On darwin targets the resulting Mach-O is ad-hoc CMS-signed via
// quill (pure-Go) before goreleaser is done, so the macOS kernel will
// exec the binary on Apple Silicon without further action. Real
// Developer ID signing and notarization are layered on top via the
// signs: and notarize: pipes — quill strips the ad-hoc signature
// before re-signing.
func buildSEA(ctx context.Context, target Target, tool, buildDir, mainPath, outPath string) error {
	version, err := resolveVersion(buildDir)
	if err != nil {
		return fmt.Errorf("node: resolve node version: %w", err)
	}

	targetNode, err := downloadHostBinary(ctx, version, target)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	cfg, err := buildSEAConfigJSON(buildDir, mainPath, targetNode, outPath)
	if err != nil {
		return err
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(filepath.Dir(outPath), "sea-config.json")
	if err := os.WriteFile(cfgPath, cfgBytes, 0o600); err != nil {
		return fmt.Errorf("node: write sea-config.json: %w", err)
	}

	if err := base.Exec(ctx, []string{tool, "--build-sea", cfgPath}, nil, ""); err != nil {
		return err
	}

	if target.Goos() == "darwin" {
		if err := signMachO(outPath, filepath.Base(outPath)); err != nil {
			return err
		}
	}

	return os.Chmod(outPath, 0o755)
}

// buildSEAConfigJSON renders the sea-config.json contents goreleaser
// will hand to `node --build-sea`. Starts from the user's
// sea-config.json in buildDir (if any), then forces the
// goreleaser-owned fields and rewrites relative `assets` paths to be
// absolute relative to buildDir so they survive the move into the
// dist directory.
func buildSEAConfigJSON(buildDir, mainPath, targetNode, output string) (map[string]any, error) {
	cfg, err := loadUserSEAConfig(buildDir)
	if err != nil {
		return nil, err
	}

	cfg["main"] = mainPath
	cfg["output"] = output
	cfg["executable"] = targetNode

	rewriteAssetPaths(cfg, buildDir)
	return cfg, nil
}

// loadUserSEAConfig reads <buildDir>/sea-config.json into a generic
// map. Returns an empty (non-nil) map when the file does not exist.
func loadUserSEAConfig(buildDir string) (map[string]any, error) {
	path := filepath.Join(buildDir, "sea-config.json")
	bts, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("node: read %s: %w", path, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(bts, &cfg); err != nil {
		return nil, fmt.Errorf("node: parse %s: %w", path, err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

// rewriteAssetPaths converts relative asset values in cfg["assets"]
// into absolute paths anchored at buildDir. Node resolves `assets`
// paths relative to the directory containing sea-config.json, but
// goreleaser writes the merged config into the dist directory, so
// relative user paths would otherwise break.
func rewriteAssetPaths(cfg map[string]any, buildDir string) {
	assets, ok := cfg["assets"].(map[string]any)
	if !ok || len(assets) == 0 {
		return
	}
	for name, v := range assets {
		p, ok := v.(string)
		if !ok || filepath.IsAbs(p) {
			continue
		}
		assets[name] = filepath.Join(buildDir, p)
	}
}

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
