package node

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// userSEAConfigFile is the filename goreleaser looks up in the build
// directory for user-supplied sea-config.json fields. Goreleaser owns
// `output`, `executable`, `main`, `useCodeCache`, and `useSnapshot` —
// any user-set values for those keys are overridden.
const userSEAConfigFile = "sea-config.json"

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
// resolved against buildDir so they keep working from the scratch
// directory). Goreleaser-owned fields (`output`, `executable`,
// `main`, `useCodeCache`, `useSnapshot`) always win.
//
// On darwin targets the resulting Mach-O is ad-hoc CMS-signed via
// quill (pure-Go) before it lands at outPath, so the macOS kernel
// will exec the binary on Apple Silicon without further action. Real
// Developer ID signing and notarization are layered on top via the
// signs: and notarize: pipes — quill strips the ad-hoc signature
// before re-signing.
//
// outPath is written atomically: --build-sea generates into a sibling
// tempfile, signing happens in place on the temp, then a rename
// promotes the temp to outPath.
func buildSEA(ctx context.Context, target Target, buildDir, mainPath, outPath string) error {
	if !target.IsSupported() {
		return fmt.Errorf("node: unsupported target %q", target)
	}

	version, err := resolveVersion(buildDir)
	if err != nil {
		return fmt.Errorf("node: resolve node version: %w", err)
	}

	targetNode, err := downloadTargetNode(ctx, version, target)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	scratch, err := os.MkdirTemp(filepath.Dir(outPath), ".buildsea-*")
	if err != nil {
		return fmt.Errorf("node: scratch dir: %w", err)
	}
	defer os.RemoveAll(scratch)

	tmpOut := filepath.Join(scratch, filepath.Base(outPath)+".tmp")
	cfgPath := filepath.Join(scratch, "sea-config.json")
	cfg, err := buildSEAConfigJSON(buildDir, mainPath, targetNode, tmpOut)
	if err != nil {
		return err
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, cfgBytes, 0o600); err != nil {
		return fmt.Errorf("node: write sea-config.json: %w", err)
	}

	if err := runBuildSEA(ctx, cfgPath); err != nil {
		return err
	}

	if target.Goos() == "darwin" {
		base := filepath.Base(outPath)
		id := strings.TrimSuffix(base, filepath.Ext(base))
		if err := signMachO(tmpOut, id); err != nil {
			return err
		}
	}

	if err := os.Chmod(tmpOut, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpOut, outPath); err != nil {
		return fmt.Errorf("node: rename %s -> %s: %w", tmpOut, outPath, err)
	}
	return nil
}

// runBuildSEA is the executor for `node --build-sea <config>`. It is a
// package-level variable so tests can record argv and stub out the real
// subprocess. Uses whatever `node` is on PATH; surfaces the underlying
// failure (including missing binary or unsupported flag) verbatim.
//
//nolint:gochecknoglobals
var runBuildSEA = func(ctx context.Context, cfgPath string) error {
	cmd := exec.CommandContext(ctx, "node", "--build-sea", cfgPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("node: node --build-sea %s: %w (output: %s)",
			cfgPath, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// downloadTargetNode resolves to downloadHostBinary in production;
// tests swap it to short-circuit the network.
//
//nolint:gochecknoglobals
var downloadTargetNode = downloadHostBinary

// buildSEAConfigJSON renders the sea-config.json contents goreleaser
// will hand to `node --build-sea`. Starts from the user's
// sea-config.json in buildDir (if any), then forces the
// goreleaser-owned fields and rewrites relative `assets` paths to be
// absolute relative to buildDir so they survive the move into the
// scratch directory.
func buildSEAConfigJSON(buildDir, mainPath, targetNode, output string) (map[string]any, error) {
	cfg, err := loadUserSEAConfig(buildDir)
	if err != nil {
		return nil, err
	}

	// Goreleaser-owned fields — always overwrite whatever the user
	// might have set, since these point at internals (cache paths,
	// scratch tempfiles, etc.).
	cfg["main"] = mainPath
	cfg["output"] = output
	cfg["executable"] = targetNode
	cfg["useCodeCache"] = false
	cfg["useSnapshot"] = false

	rewriteAssetPaths(cfg, buildDir)
	return cfg, nil
}

// loadUserSEAConfig reads <buildDir>/sea-config.json into a generic
// map. Returns an empty (non-nil) map when the file does not exist.
func loadUserSEAConfig(buildDir string) (map[string]any, error) {
	path := filepath.Join(buildDir, userSEAConfigFile)
	bts, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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
// goreleaser writes the merged config into a scratch dir, so relative
// user paths would otherwise break.
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
