package tmpl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

func FuzzTemplateApplier(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		if data == "" {
			return
		}

		ctx := testctx.NewWithCfg(
			config.Project{
				ProjectName: "proj",
				Release: config.Release{
					Draft: true,
				},
			},
			testctx.WithVersion("1.2.3"),
			testctx.WithGitInfo(context.GitInfo{
				PreviousTag: "v1.2.2",
				CurrentTag:  "v1.2.3",
				Branch:      "test-branch",
				Commit:      "commit",
				FullCommit:  "fullcommit",
				ShortCommit: "shortcommit",
				TagSubject:  "awesome release",
				TagContents: "awesome release\n\nanother line",
				TagBody:     "another line",
			}),
			testctx.WithEnv(map[string]string{
				"FOO":       "bar",
				"MULTILINE": "something with\nmultiple lines",
			}),
		)

		tpl := New(ctx).WithArtifact(
			&artifact.Artifact{
				Name:      "binary-name",
				Path:      "/tmp/foo.exe",
				Goarch:    "amd64",
				Goos:      "linux",
				Goarm:     "6",
				Goamd64:   "v3",
				Goarm64:   "v8.0",
				Go386:     "sse2",
				Gomips:    "softfloat",
				Goppc64:   "power8",
				Goriscv64: "rva22u64",
				Target:    "linux_amd64",
				Extra: map[string]any{
					artifact.ExtraBinary: "binary",
					artifact.ExtraExt:    ".exe",
				},
			},
		)

		result, err := tpl.Apply(data)

		// Validation: if there's no template syntax, it should return the input unchanged
		if !strings.Contains(data, "{{") && !strings.Contains(data, "}}") {
			if err != nil {
				t.Errorf("Expected no error for non-template input, got: %v", err)
			}
			// For non-template inputs, result should match input (possibly with whitespace changes)
			if strings.TrimSpace(result) != strings.TrimSpace(data) {
				// Only report as an issue if it's not about a missing key error
				if err == nil || !strings.Contains(err.Error(), "map has no entry for key") {
					t.Errorf("Expected result to match input for non-template. Input: %q, Result: %q", data, result)
				}
			}
		}

		// Validation: result should never be nil unless error occurred
		if result == "" && err == nil && data != "" {
			// Check if the template contains valid fields
			if strings.Contains(data, "{{") {
				t.Errorf("Empty result with no error for template: %q", data)
			}
		}
	})
}

func FuzzTemplateBool(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		// Skip empty or too long inputs
		if data == "" || len(data) > 10000 {
			return
		}

		ctx := testctx.NewWithCfg(
			config.Project{
				Env: []string{
					"FOO=true",
					"BAR=false",
					"EMPTY=",
					"NUMERIC=123",
					"SPECIAL=!@#$%^&*()",
				},
			},
		)

		tpl := New(ctx)
		result, err := tpl.Bool(data)

		// Validation: always check that either a result is returned or an error is handled gracefully
		if err != nil && !strings.Contains(err.Error(), "map has no entry for key") {
			// Only templates with invalid syntax should cause errors
			if !strings.Contains(data, "{{") {
				t.Errorf("Unexpected error for non-template boolean input: %v", err)
			}
		}

		// For valid non-template inputs, we should get predictable results
		if !strings.Contains(data, "{{") {
			// Empty or whitespace-only should be false
			if strings.TrimSpace(data) == "" {
				if result != false {
					t.Errorf("Expected false for empty input, got true")
				}
			} else if strings.EqualFold(strings.TrimSpace(data), "true") {
				// Explicit "true" values should be true
				if result != true {
					t.Errorf("Expected true for 'true' input, got false")
				}
			}
		}
	})
}

func FuzzTemplateApplyEnvOnly(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		// Skip empty or too long inputs
		if len(data) > 10000 {
			return
		}

		ctx := testctx.NewWithCfg(
			config.Project{
				Env: []string{
					"FOO=bar",
					"TEST=value",
					"EMPTY=",
				},
			},
		)

		tpl := New(ctx)
		result, err := tpl.ApplySingleEnvOnly(data)

		// Validations for ApplySingleEnvOnly
		if data == "" {
			if err != nil || result != "" {
				t.Errorf("Expected empty result and no error for empty input")
			}
			return
		}

		trimmedData := strings.TrimSpace(data)
		if trimmedData == "" {
			if err != nil || result != "" {
				t.Errorf("Expected empty result and no error for whitespace-only input")
			}
			return
		}

		// Check if it's a valid env-only template
		isValidEnvTemplate := envOnlyRe.MatchString(data)

		// If it matches the pattern but returns an error, validate the error type
		if isValidEnvTemplate && err != nil {
			// Should only error if the env variable doesn't exist
			if !strings.Contains(err.Error(), "map has no entry for key") {
				t.Errorf("Valid env template %q caused unexpected error: %v", data, err)
			}
		}

		// If it doesn't match the pattern but doesn't error, that's unexpected
		if !isValidEnvTemplate && err == nil && result != "" {
			t.Errorf("Expected error for invalid env-only template %q, but got result %q", data, result)
		}

		// If it matches the pattern and doesn't error, result should be valid
		if isValidEnvTemplate && err == nil {
			if result == "" && !strings.Contains(data, "EMPTY") {
				t.Errorf("Valid env template %q returned empty result unexpectedly", data)
			}
		}
	})
}

func FuzzTemplateSlice(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		// Skip too long inputs
		if len(data) > 10000 {
			return
		}

		ctx := testctx.New()
		tpl := New(ctx)

		input := []string{data}
		result, err := tpl.Slice(input)

		// Validate slice behavior
		if err != nil {
			// Check if error is expected based on input
			if !strings.Contains(data, "{{") && !strings.Contains(err.Error(), "unexpected") {
				t.Errorf("Unexpected error for non-template slice input %q: %v", data, err)
			}
		} else {
			// If no error, result should have one element
			if len(result) != 1 {
				t.Errorf("Expected result slice to have 1 element, got %d", len(result))
			}

			// For non-template inputs, result should match input
			if !strings.Contains(data, "{{") {
				if strings.TrimSpace(result[0]) != strings.TrimSpace(data) {
					t.Errorf("Expected result to match input for non-template slice. Input: %q, Result: %q", data, result[0])
				}
			}
		}
	})
}

func FuzzTemplateWithBuildOptions(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		// Skip empty or too long inputs
		if len(data) > 10000 {
			return
		}

		ctx := testctx.New()

		target := &buildTarget{
			Target:  "linux_amd64",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v3",
			Goarm:   "6",
			Gomips:  "softfloat",
		}

		tpl := New(ctx).WithBuildOptions(build.Options{
			Name:   "test",
			Path:   "./test",
			Ext:    ".test",
			Target: target,
		})

		result, err := tpl.Apply(data)

		// Validation for build options templates
		if err != nil && !strings.Contains(err.Error(), "map has no entry for key") {
			// Only templates with invalid syntax should cause non-key errors
			if !strings.Contains(data, "{{") {
				t.Errorf("Unexpected error for non-template build options input %q: %v", data, err)
			}
		}

		// For non-template inputs, result should match input (possibly with whitespace changes)
		if !strings.Contains(data, "{{") && !strings.Contains(data, "}}") {
			if err != nil {
				t.Errorf("Expected no error for non-template build options input: %v", err)
			}
		}

		// Check that result is never nil unless error occurred
		if result == "" && err == nil && data != "" {
			if strings.Contains(data, "{{") {
				t.Errorf("Empty result with no error for template: %q", data)
			}
		}
	})
}

func FuzzTemplateChecksums(f *testing.F) {
	// Add corpus entries for common checksum patterns
	f.Add("{{ sha256 .ArtifactPath }}")
	f.Add("{{ md5 .ArtifactPath }}")
	f.Add("{{ sha1 .ArtifactPath }}")
	f.Add("{{ blake2b .ArtifactPath }}")

	f.Fuzz(func(t *testing.T, data string) {
		// Skip empty or too long inputs
		if len(data) > 10000 {
			return
		}

		// Create a temporary file for checksum testing
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile")
		content := "test content for checksum"

		if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
			t.Skipf("Could not create temporary file: %v", err)
		}

		ctx := testctx.New()
		artifact := &artifact.Artifact{
			Path: tmpFile,
		}

		tpl := New(ctx).WithArtifact(artifact)

		// Check for known checksum patterns in the template
		checksumFunctions := []string{
			"sha256", "md5", "sha1", "blake2b", "blake2s",
			"sha224", "sha384", "sha512", "sha3_224", "sha3_256",
			"sha3_384", "sha3_512", "crc32",
		}

		containsValidChecksumFunc := false
		for _, fn := range checksumFunctions {
			// Check for valid function call patterns with parentheses
			if strings.Contains(data, fn+"(") {
				containsValidChecksumFunc = true
				break
			}
		}

		// Apply template regardless of content to test robustness
		result, err := tpl.Apply(data)

		// If it's a checksum template and succeeded, validate the output
		if containsValidChecksumFunc && err == nil && result != "" {
			// All checksums should be hex strings of specific lengths
			if !isValidHex(result) {
				t.Errorf("Checksum function returned non-hex result: %q", result)
			}
		}

		// If there's an error, it should be a template execution error, not a panic
		if err != nil && !strings.Contains(err.Error(), "map has no entry for key") {
			// Skip expected errors from missing keys
			if containsValidChecksumFunc && !strings.Contains(err.Error(), "wrong type") {
				// For valid checksum templates, we expect either success or specific template errors
				t.Logf("Template %q caused error: %v", data, err)
			}
		}
	})
}

func FuzzTemplateSemverOperations(f *testing.F) {
	// Add corpus entries for common semver operations
	f.Add("{{ .Tag | incmajor }}")
	f.Add("{{ .Version | incminor }}")
	f.Add("{{ .Tag | incpatch }}")

	f.Fuzz(func(t *testing.T, data string) {
		// Skip too long inputs
		if len(data) > 10000 {
			return
		}

		ctx := testctx.New()
		ctx.Semver = context.Semver{
			Major:      1,
			Minor:      2,
			Patch:      3,
			Prerelease: "beta1",
		}
		ctx.Version = "1.2.3-beta1"
		ctx.Git.CurrentTag = "v1.2.3-beta1"

		tpl := New(ctx)
		result, err := tpl.Apply(data)

		// If we're operating on semver fields, check the results make sense
		semverOps := []string{"incmajor", "incminor", "incpatch"}
		for _, op := range semverOps {
			if strings.Contains(data, op) {
				if err != nil && !strings.Contains(err.Error(), "map has no entry for key") {
					// For invalid templates, we expect template errors, not panics
					if strings.Contains(data, "{{") && strings.Contains(data, "}}") {
						t.Logf("Semver operation template %q caused error: %v", data, err)
					}
				}
				// If we get a result and it contains version operations,
				// check that it looks like a version
				if result != "" && err == nil {
					// Could be a version prefixed with v or not
					version := strings.TrimPrefix(result, "v")

					// Just validate that if we have incmajor, the result contains something like a version
					// The exact validation is complex due to templating possibilities
					if strings.Contains(data, "incmajor") && !strings.Contains(version, ".0.0") {
						// Major increment should contain .0.0 pattern
						// Skip if it's just a formatting error or missing key error
						if strings.Contains(data, "Tag") || strings.Contains(data, "Version") {
							t.Logf("IncMajor result %q might not follow expected pattern for %q", result, data)
						}
					}
				}
			}
		}
	})
}

// Helper functions for validations
func isValidHex(s string) bool {
	// Check if string contains only valid hex characters
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	// Check for reasonable length (most checksums are 32-128 characters)
	return len(s) >= 8 && len(s) <= 128
}

type buildTarget struct {
	Target  string
	Goos    string
	Goarch  string
	Goamd64 string
	Goarm   string
	Gomips  string
}

func (t *buildTarget) String() string { return t.Target }

func (t *buildTarget) Fields() map[string]string {
	return map[string]string{
		"target": t.Target,
		"os":     t.Goos,
		"arch":   t.Goarch,
		"amd64":  t.Goamd64,
		"arm":    t.Goarm,
		"mips":   t.Gomips,
	}
}
