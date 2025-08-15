package makeself

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestMakeselfArchive(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test.run")

	// Create mock makeself script that creates a simple archive
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
# Mock makeself script for testing
# Find the output path (the argument that ends with .run, whether file exists or not)
for arg in "$@"; do
    if [[ "$arg" == *.run ]]; then
        OUTPUT_PATH="$arg"
        break
    fi
done
echo "Creating self-extracting archive: $OUTPUT_PATH"
# Create a simple executable script as output
cat > "$OUTPUT_PATH" << 'EOF'
#!/bin/bash
echo "Self-extracting archive created successfully"
exit 0
EOF
chmod +x "$OUTPUT_PATH"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	archive := New(f)
	defer archive.Close()

	// Test adding files
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	// Test error handling for non-existent file
	require.Error(t, archive.Add(config.File{
		Source:      "../testdata/nope.txt",
		Destination: "nope.txt",
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	// Verify the output file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestMakeselfArchiveWithCustomInstallScript(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-custom.run")

	// Create custom install script
	customScript := "#!/bin/bash\necho 'Custom install script executed'\n"

	// Create mock makeself script
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
# Find the output path (the argument that ends with .run, whether file exists or not)
for arg in "$@"; do
    if [[ "$arg" == *.run ]]; then
        OUTPUT_PATH="$arg"
        break
    fi
done
cat > "$OUTPUT_PATH" << 'EOF'
#!/bin/bash
echo "Archive with custom install script"
exit 0
EOF
chmod +x "$OUTPUT_PATH"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	archive := NewWithInstallScript(f, customScript)
	defer archive.Close()

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	// Verify output file exists
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestMakeselfArchiveError(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-error.run")

	// Create mock makeself script that fails
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
echo "makeself.sh error" >&2
exit 1
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	archive := New(f)

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	// Close should fail because makeself returns error
	err = archive.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "makeself failed")
}

func TestMakeselfArchiveMissingMakeself(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-missing.run")

	// Ensure makeself is not in PATH
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", "/nonexistent")

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	archive := New(f)

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	// Close should fail because makeself is not found
	err = archive.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "makeself command not found")
}

func TestMakeselfWithConfig(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-config.run")

	// Create mock makeself script
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
# Find the output path (the argument that ends with .run, whether file exists or not)
for arg in "$@"; do
    if [[ "$arg" == *.run ]]; then
        OUTPUT_PATH="$arg"
        break
    fi
done
cat > "$OUTPUT_PATH" << 'EOF'
#!/bin/bash
echo "Archive with configuration"
exit 0
EOF
chmod +x "$OUTPUT_PATH"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	cfg := MakeselfConfig{
		Label:         "Test Archive",
		InstallScript: "#!/bin/bash\necho 'Custom install'",
		Compression:   "none",
		ExtraArgs:     []string{"--notemp"},
	}

	archive := NewWithConfig(f, "", cfg)
	defer archive.Close()

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	// Verify output file exists
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestFindMakeselfCommand(t *testing.T) {
	// This test checks if findMakeselfCommand works correctly
	// In real environments, it might find makeself or makeself.sh
	cmd := findMakeselfCommand()

	// The command might be found or not, depending on the test environment
	// If found, it should be one of the expected commands
	if cmd != "" {
		require.Contains(t, []string{"makeself", "makeself.sh"}, cmd)

		// Verify the command actually exists
		_, err := exec.LookPath(cmd)
		require.NoError(t, err)
	}
}

func TestMakeselfWithLSMTemplate(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-lsm-template.run")

	// Create mock makeself script that accepts LSM flag
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
# Mock makeself script that accepts LSM flag
for arg in "$@"; do
    if [[ "$arg" == "--lsm" ]]; then
        echo "LSM flag detected"
    fi
done
# Find the output path (the argument that ends with .run, whether file exists or not)
for arg in "$@"; do
    if [[ "$arg" == *.run ]]; then
        OUTPUT_PATH="$arg"
        break
    fi
done
cat > "$OUTPUT_PATH" << 'EOF'
#!/bin/bash
echo "Archive with LSM template"
exit 0
EOF
chmod +x "$OUTPUT_PATH"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	lsmContent := `Begin4
Title: Test Software
Version: 1.0.0
Description: A test software package
Author: Test Author
Maintained-by: test@example.com
Platforms: Linux
Copying-policy: MIT
End`

	cfg := MakeselfConfig{
		Label:       "Test Archive with LSM Template",
		LSMTemplate: lsmContent,
	}

	archive := NewWithConfig(f, "", cfg)
	defer archive.Close()

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	// Verify output file exists
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestMakeselfWithLSMFile(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-lsm-file.run")

	// Create a temporary LSM file
	lsmFile := filepath.Join(tmp, "test.lsm")
	lsmContent := `Begin4
Title: External LSM Test
Version: 2.0.0
Description: Testing external LSM file
Author: External Author
Maintained-by: external@example.com
Platforms: Linux
Copying-policy: GPL
End`
	require.NoError(t, os.WriteFile(lsmFile, []byte(lsmContent), 0o644))

	// Create mock makeself script that accepts LSM flag
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
for arg in "$@"; do
    if [[ "$arg" == "--lsm" ]]; then
        echo "LSM flag detected with file: $2"
    fi
done
# Find the output path (the argument that ends with .run, whether file exists or not)
for arg in "$@"; do
    if [[ "$arg" == *.run ]]; then
        OUTPUT_PATH="$arg"
        break
    fi
done
cat > "$OUTPUT_PATH" << 'EOF'
#!/bin/bash
echo "Archive with external LSM file"
exit 0
EOF
chmod +x "$OUTPUT_PATH"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	cfg := MakeselfConfig{
		Label:   "Test Archive with External LSM",
		LSMFile: lsmFile,
	}

	archive := NewWithConfig(f, "", cfg)
	defer archive.Close()

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	// Verify output file exists
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestMakeselfWithMissingLSMFile(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-missing-lsm.run")

	// Create mock makeself script
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
echo "This should not be reached"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	cfg := MakeselfConfig{
		Label:   "Test Archive with Missing LSM",
		LSMFile: "/nonexistent/file.lsm",
	}

	archive := NewWithConfig(f, "", cfg)

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))

	// Close should fail because LSM file doesn't exist
	err = archive.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "LSM file")
	require.Contains(t, err.Error(), "not found")
}

func TestMakeselfInstallScriptPathHandling(t *testing.T) {
	tmp := t.TempDir()
	outputFile := filepath.Join(tmp, "test-script-path.run")

	// Create install script files for testing
	installScript1 := filepath.Join(tmp, "script.sh")
	scriptContent := "#!/bin/bash\necho 'Custom install script executed'"
	require.NoError(t, os.WriteFile(installScript1, []byte(scriptContent), 0o755))

	// Create mock makeself script that captures the install script argument
	mockMakeselfScript := filepath.Join(tmp, "makeself")
	mockScript := `#!/bin/bash
# Capture the install script argument (last argument)
echo "Install script arg: ${@: -1}" > "` + tmp + `/captured_args.txt"
# Find the output path (the argument that ends with .run, whether file exists or not)
for arg in "$@"; do
    if [[ "$arg" == *.run ]]; then
        OUTPUT_PATH="$arg"
        break
    fi
done
if [[ -z "$OUTPUT_PATH" ]]; then
    echo "Error: Could not find output path" >&2
    exit 1
fi
cat > "$OUTPUT_PATH" << 'EOF'
#!/bin/bash
echo "Archive with install script path test"
exit 0
EOF
chmod +x "$OUTPUT_PATH"
`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock script
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()
	os.Setenv("PATH", tmp+":"+originalPath)

	// Test 1: Install script file without ./ prefix
	t.Run("without_dot_slash", func(t *testing.T) {
		f, err := os.Create(outputFile + "_1.run")
		require.NoError(t, err)
		defer f.Close()

		cfg := MakeselfConfig{
			Label:             "Test Archive Path Without Dot Slash",
			InstallScriptFile: "script.sh",
		}

		archive := NewWithConfig(f, "", cfg)
		defer archive.Close()

		require.NoError(t, archive.Add(config.File{
			Source:      installScript1,
			Destination: "script.sh",
		}))

		require.NoError(t, archive.Close())
		require.NoError(t, f.Close())

		// Check captured args
		capturedArgsFile := filepath.Join(tmp, "captured_args.txt")
		capturedArgs, err := os.ReadFile(capturedArgsFile)
		require.NoError(t, err)
		require.Contains(t, string(capturedArgs), "Install script arg: ./script.sh")
	})

	// Test 2: Install script file with ./ prefix already
	t.Run("with_dot_slash", func(t *testing.T) {
		f, err := os.Create(outputFile + "_2.run")
		require.NoError(t, err)
		defer f.Close()

		cfg := MakeselfConfig{
			Label:             "Test Archive Path With Dot Slash",
			InstallScriptFile: "./script.sh",
		}

		archive := NewWithConfig(f, "", cfg)
		defer archive.Close()

		require.NoError(t, archive.Add(config.File{
			Source:      installScript1,
			Destination: "script.sh",
		}))

		require.NoError(t, archive.Close())
		require.NoError(t, f.Close())

		// Check captured args - should not have double ./
		capturedArgsFile := filepath.Join(tmp, "captured_args.txt")
		capturedArgs, err := os.ReadFile(capturedArgsFile)
		require.NoError(t, err)
		require.Contains(t, string(capturedArgs), "Install script arg: ./script.sh")
		// Ensure there's no double ./
		require.NotContains(t, string(capturedArgs), "Install script arg: .//")
	})
}
