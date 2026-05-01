package nodesea

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
)

// UserSEAConfigFile is the filename goreleaser looks up in the build
// directory for user-supplied sea-config.json fields. Goreleaser owns
// `output`, `executable`, `main`, `useCodeCache`, and `useSnapshot` —
// any user-set values for those keys are overridden.
const UserSEAConfigFile = "sea-config.json"

// BuildOptions configures BuildViaBuildSEA. Every field except
// BuildDir and CodeSignID is required.
type BuildOptions struct {
	// BuildToolNode is the absolute path to a Node binary that can
	// drive `--build-sea`, as returned by BuildToolNode.
	BuildToolNode string

	// Target identifies the per-target Node release this SEA is built
	// for (e.g. "linux-x64"). Determines container format and whether
	// the result needs ad-hoc Mach-O signing.
	Target nodedist.Target

	// Version is the per-target Node release version (e.g. "v22.20.0").
	// Used to download the cached target Node binary that becomes the
	// SEA `executable`.
	Version string

	// MainJS is the absolute path to the user's entrypoint JS file,
	// written into sea-config.json's `main` field.
	MainJS string

	// OutPath is where the resulting SEA binary will be written
	// atomically with executable permissions.
	OutPath string

	// BuildDir is the user's project directory. If it contains a
	// sea-config.json file, that file is merged with goreleaser-owned
	// fields before being passed to `node --build-sea`. Optional.
	BuildDir string

	// CodeSignID is the ad-hoc CMS signing identifier applied to
	// darwin outputs by quill. Empty → derived from filepath.Base(OutPath).
	CodeSignID string
}

// BuildViaBuildSEA produces a Single Executable Application at
// opts.OutPath by invoking `<opts.BuildToolNode> --build-sea
// sea-config.json`, where sea-config.json points `executable` at the
// cached per-target Node binary downloaded for opts.Version+opts.Target.
//
// If a sea-config.json exists in opts.BuildDir, its user-tunable
// fields are merged into the rendered config (relative `assets` paths
// are resolved against opts.BuildDir so they keep working from the
// scratch directory). Goreleaser-owned fields (`output`, `executable`,
// `main`, `useCodeCache`, `useSnapshot`) always win.
//
// On darwin targets the resulting Mach-O is ad-hoc CMS-signed via
// quill (pure-Go) before it lands at OutPath, so the macOS kernel will
// exec the binary on Apple Silicon without further action. Real
// Developer ID signing and notarization are layered on top via the
// signs: and notarize: pipes — quill strips the ad-hoc signature
// before re-signing.
//
// OutPath is written atomically: --build-sea generates into a sibling
// tempfile, signing happens in place on the temp, then a rename
// promotes the temp to OutPath.
func BuildViaBuildSEA(ctx context.Context, opts BuildOptions) error {
	if err := validateBuildOptions(opts); err != nil {
		return err
	}

	cacheDir, err := nodedist.CacheDir()
	if err != nil {
		return err
	}
	targetNode, err := nodedist.Download(ctx, cacheDir, opts.Version, opts.Target)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(opts.OutPath), 0o755); err != nil {
		return err
	}

	scratch, err := os.MkdirTemp(filepath.Dir(opts.OutPath), ".buildsea-*")
	if err != nil {
		return fmt.Errorf("nodesea: scratch dir: %w", err)
	}
	defer os.RemoveAll(scratch)

	tmpOut := filepath.Join(scratch, filepath.Base(opts.OutPath)+".tmp")
	cfgPath := filepath.Join(scratch, "sea-config.json")
	cfg, err := buildSEAConfigJSON(opts, targetNode, tmpOut)
	if err != nil {
		return err
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, cfgBytes, 0o600); err != nil {
		return fmt.Errorf("nodesea: write sea-config.json: %w", err)
	}

	if err := runBuildSEA(ctx, opts.BuildToolNode, cfgPath); err != nil {
		return err
	}

	if FormatFor(opts.Target.Goos()) == FormatMachO {
		id := opts.CodeSignID
		if id == "" {
			base := filepath.Base(opts.OutPath)
			id = strings.TrimSuffix(base, filepath.Ext(base))
		}
		if err := signMachO(tmpOut, id); err != nil {
			return err
		}
	}

	if err := os.Chmod(tmpOut, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpOut, opts.OutPath); err != nil {
		return fmt.Errorf("nodesea: rename %s -> %s: %w", tmpOut, opts.OutPath, err)
	}
	return nil
}

// runBuildSEA is the executor for `<node> --build-sea <config>`. It is
// a package-level variable so tests can record argv and stub out the
// real subprocess.
//
//nolint:gochecknoglobals
var runBuildSEA = func(ctx context.Context, nodePath, cfgPath string) error {
	cmd := exec.CommandContext(ctx, nodePath, "--build-sea", cfgPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nodesea: %s --build-sea %s: %w (output: %s)",
			nodePath, cfgPath, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func validateBuildOptions(opts BuildOptions) error {
	var errs []string
	if opts.BuildToolNode == "" {
		errs = append(errs, "BuildToolNode is required")
	}
	if opts.Version == "" {
		errs = append(errs, "Version is required")
	}
	if opts.MainJS == "" {
		errs = append(errs, "MainJS is required")
	}
	if opts.OutPath == "" {
		errs = append(errs, "OutPath is required")
	}
	if FormatFor(opts.Target.Goos()) == 0 {
		errs = append(errs, fmt.Sprintf("target %q has no SEA injector", opts.Target))
	}
	if len(errs) > 0 {
		return fmt.Errorf("nodesea: invalid BuildOptions: %s", strings.Join(errs, "; "))
	}
	return nil
}

// buildSEAConfigJSON renders the sea-config.json contents goreleaser
// will hand to `node --build-sea`. Starts from the user's
// sea-config.json in opts.BuildDir (if any), then forces the
// goreleaser-owned fields and rewrites relative `assets` paths to be
// absolute relative to opts.BuildDir so they survive the move into
// the scratch directory.
func buildSEAConfigJSON(opts BuildOptions, targetNode, output string) (map[string]any, error) {
	cfg, err := loadUserSEAConfig(opts.BuildDir)
	if err != nil {
		return nil, err
	}

	// Goreleaser-owned fields — always overwrite whatever the user
	// might have set, since these point at internals (cache paths,
	// scratch tempfiles, etc.).
	cfg["main"] = opts.MainJS
	cfg["output"] = output
	cfg["executable"] = targetNode
	cfg["useCodeCache"] = false
	cfg["useSnapshot"] = false

	rewriteAssetPaths(cfg, opts.BuildDir)
	return cfg, nil
}

// loadUserSEAConfig reads <buildDir>/sea-config.json into a generic
// map. Returns an empty (non-nil) map when buildDir is empty or the
// file does not exist.
func loadUserSEAConfig(buildDir string) (map[string]any, error) {
	if buildDir == "" {
		return map[string]any{}, nil
	}
	path := filepath.Join(buildDir, UserSEAConfigFile)
	bts, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("nodesea: read %s: %w", path, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(bts, &cfg); err != nil {
		return nil, fmt.Errorf("nodesea: parse %s: %w", path, err)
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
	if !ok || len(assets) == 0 || buildDir == "" {
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
