package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/quill/quill"
)

func createSEAConfig(tpl *tmpl.Template, build config.Build, name, targetNode, output string) error {
	main, err := tpl.Apply(build.Main)
	if err != nil {
		return fmt.Errorf("node: template main: %w", err)
	}
	mainPath := filepath.Join(build.Dir, main)
	if _, err := os.Stat(mainPath); err != nil {
		return fmt.Errorf("node: main %q not found in %q: %w", main, build.Dir, err)
	}
	absMainPath, err := filepath.Abs(mainPath)
	if err != nil {
		return fmt.Errorf("node: abs main %q: %w", mainPath, err)
	}

	absBuildDir, err := filepath.Abs(build.Dir)
	if err != nil {
		return fmt.Errorf("node: abs build dir %q: %w", build.Dir, err)
	}

	cfg, err := buildSEAConfigJSON(
		absBuildDir,
		absMainPath,
		targetNode,
		output,
	)
	if err != nil {
		return err
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(name, cfgBytes, 0o600); err != nil {
		return fmt.Errorf("node: write sea-config.json: %w", err)
	}
	return nil
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
	if !ok {
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
