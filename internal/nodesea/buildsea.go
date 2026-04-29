package nodesea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SEAConfig is the user-supplied subset of sea-config.json fields
// accepted by goreleaser. The fields goreleaser owns semantically —
// `output`, `executable`, `useCodeCache`, `useSnapshot` — are not
// representable here on purpose. See Node's documentation:
// https://nodejs.org/api/single-executable-applications.html#generating-single-executable-preparation-blobs
type SEAConfig struct {
	// Assets is a map of asset name → file path, baked into the SEA
	// blob and accessible at runtime via sea.getAsset().
	Assets map[string]string

	// ExecArgv is a list of Node CLI flags baked into the binary
	// (e.g. ["--max-old-space-size=4096"]).
	ExecArgv []string

	// DisableExperimentalSEAWarning controls Node's runtime warning
	// about SEA being experimental. nil → goreleaser default (true).
	// Set explicitly to surface the warning.
	DisableExperimentalSEAWarning *bool

	// MainFormat selects the module system used to evaluate the main
	// entrypoint: "commonjs" (default) or "module".
	MainFormat string
}

// BuildOptions configures BuildViaBuildSEA. Every field except
// SEAConfig and CodeSignID is required.
type BuildOptions struct {
	// BuildToolNode is the absolute path to a Node binary that can
	// drive `--build-sea`, as returned by BuildToolNode.
	BuildToolNode string

	// Target identifies the per-target Node release this SEA is built
	// for (e.g. "linux-x64"). Determines container format and whether
	// the result needs ad-hoc Mach-O signing.
	Target Target

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

	// SEAConfig carries user-tunable sea-config.json fields. See
	// SEAConfig for the whitelisted set.
	SEAConfig SEAConfig

	// CodeSignID is the ad-hoc CMS signing identifier applied to
	// darwin outputs. Empty → derived from filepath.Base(OutPath).
	CodeSignID string
}

// BuildViaBuildSEA produces a Single Executable Application at
// opts.OutPath by invoking `<opts.BuildToolNode> --build-sea
// sea-config.json`, where sea-config.json points `executable` at the
// cached per-target Node binary downloaded for opts.Version+opts.Target.
//
// On darwin targets the resulting Mach-O is ad-hoc CMS-signed via
// codesign(1) before it lands at OutPath. When codesign(1) is not on
// PATH (typical when cross-compiling for darwin from non-darwin hosts),
// the binary is left unsigned: it is well-formed but the macOS kernel
// will refuse to exec it until it is re-signed (e.g. via goreleaser's
// signs: pipe on a darwin runner).
//
// OutPath is written atomically: --build-sea generates into a sibling
// tempfile, signing happens in place on the temp, then a rename
// promotes the temp to OutPath.
func BuildViaBuildSEA(ctx context.Context, opts BuildOptions) error {
	if err := validateBuildOptions(opts); err != nil {
		return err
	}

	cacheDir, err := CacheDir()
	if err != nil {
		return err
	}
	targetNode, err := downloadHost(ctx, cacheDir, opts.Version, opts.Target)
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
	cfg := buildSEAConfigJSON(opts, targetNode, tmpOut)
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
		if err := adHocSignMachO(ctx, tmpOut, id); err != nil && !errors.Is(err, ErrCodeSignUnavailable) {
			return fmt.Errorf("nodesea: ad-hoc sign: %w", err)
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
// will hand to `node --build-sea`. Goreleaser-owned fields (output,
// executable, useCodeCache, useSnapshot) are always set explicitly.
// User-provided whitelisted fields are appended only when non-zero.
func buildSEAConfigJSON(opts BuildOptions, targetNode, output string) map[string]any {
	cfg := map[string]any{
		"main":         opts.MainJS,
		"output":       output,
		"executable":   targetNode,
		"useCodeCache": false,
		"useSnapshot":  false,
	}

	disable := true
	if opts.SEAConfig.DisableExperimentalSEAWarning != nil {
		disable = *opts.SEAConfig.DisableExperimentalSEAWarning
	}
	cfg["disableExperimentalSEAWarning"] = disable

	if len(opts.SEAConfig.Assets) > 0 {
		cfg["assets"] = opts.SEAConfig.Assets
	}
	if len(opts.SEAConfig.ExecArgv) > 0 {
		cfg["execArgv"] = opts.SEAConfig.ExecArgv
	}
	if opts.SEAConfig.MainFormat != "" {
		cfg["mainFormat"] = opts.SEAConfig.MainFormat
	}
	return cfg
}
