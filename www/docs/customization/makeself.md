# Makeself Self-Extracting Archives

Makeself creates self-extracting archives that can be executed to automatically extract and optionally install their contents. This is particularly useful for distributing software that needs to be easily installable without requiring users to manually extract archives. Typically this supports Linux, MacOS and any other platform that makeself runs on. See the makeself
link below.

!!! note

    The `makeself` command requires the `makeself` or `makeself.sh` command to be available in your system PATH. You can install it from your system package manager or from [the makeself project](https://github.com/megastep/makeself).

Here is a commented `makeselfs` section with all fields specified:

```yaml title=".goreleaser.yaml"
makeselfs:
  - #
    # ID of this makeself package.
    #
    # Default: 'default'.
    id: my-installer

    # IDs of the builds which should be packaged in this makeself archive.
    #
    # Default: empty (include all).
    ids:
      - my-binary

    # Archive name template.
    #
    # Default: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}'
    # Templates: allowed.
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # Custom file extension for the self-extracting archive.
    #
    # Default: '.run'
    # Templates: allowed.
    extension: ".run"

    # Custom label/description for the archive shown during execution.
    #
    # Default: 'Self-extracting archive'
    # Templates: allowed.
    label: "{{ .ProjectName }} v{{ .Version }} Installer"

    # Compression format to use.
    #
    # Valid options: gzip, bzip2, xz, lzo, compress, none
    # Default: gzip (makeself default)
    # Templates: allowed.
    # note: none translates to makeslef's --nocomp flag
    compression: "gzip"

    # Inline install script content.
    # This script will be executed after extraction.
    # Templates: allowed.
    install_script: |
      #!/bin/bash
      echo "Installing {{ .ProjectName }}..."
      chmod +x {{ .Binary }}
      cp {{ .Binary }} /usr/local/bin/
      echo "Installation complete!"

    # Path to install script file within the archive.
    # Alternative to install_script for external script files.
    # Templates: allowed.
    install_script_file: "install.sh"

    # Additional command-line arguments to pass to makeself.
    # Templates: allowed.
    extra_args:
      - "--notemp"     # Don't use a temporary directory for extraction
      - "--needroot"   # Requires root privileges to run
      - "--keep"       # Don't remove extracted files after execution
      - "--copy"       # Copy files to temporary directory instead of extracting in place

    # Linux Software Map (LSM) template content.
    # Templates: allowed.
    lsm_template: |
      Begin4
      Title: {{ .ProjectName }}
      Version: {{ .Version }}
      Description: {{ .ProjectName }} self-extracting installer
      Author: Your Name
      Maintained-by: your-email@example.com
      Primary-site: https://github.com/youruser/yourproject
      Platforms: Linux
      Copying-policy: MIT
      End

    # Path to external LSM file.
    # Alternative to lsm_template for external LSM files.
    # Templates: allowed.
    lsm_file: "project.lsm"

    # Additional files/globs you want to add to the makeself package.
    # These files will be available to the install script.
    #
    # Default: [ 'LICENSE*', 'README*', 'CHANGELOG', 'license*', 'readme*', 'changelog'].
    # Templates: allowed.
    files:
      - LICENSE.txt
      - README_{{.Os}}.md
      - CHANGELOG.md
      - configs/*
      - scripts/*.sh
      # a more complete example
      - src: "*.md"
        dst: docs

        # Strip parent directories when adding files to the archive.
        strip_parent: true

        # File info.
        info:
          # Templates: allowed.
          owner: root

          # Templates: allowed.
          group: root

          # Must be in time.RFC3339Nano format.
          # Templates: allowed.
          mtime: "{{ .CommitDate }}"

          # File mode.
          mode: 0644

    # This will create a package without any binaries, only the files are there.
    # Useful for configuration packages or system setup packages.
    # The name template must not contain any references to `Os`, `Arch` etc, since the package will be meta.
    #
    # Templates: allowed.
    meta: false

    # Disable this makeself package.
    # Templates: allowed.
    disable: "{{ .Env.SKIP_MAKESELF }}"

    # Deprecated: use 'ids' instead.
    builds:
      - default
```

## Simple Makeself Example

Here's a minimal example to create a self-extracting installer:

```yaml title=".goreleaser.yaml"
makeselfs:
  - id: installer
    ids:
      - default
    label: "{{ .ProjectName }} v{{ .Version }} Installer"
    extra_args:
      - "--needroot"  # Ensures installer runs as root
    install_script: |
      #!/bin/bash
      echo "Installing {{ .ProjectName }}..."
      chmod +x {{ .Binary }}
      cp {{ .Binary }} /usr/local/bin/
      echo "{{ .ProjectName }} installed successfully!"
```

This will create a `.run` file that users can execute with `./yourproject_1.0.0_linux_amd64.run`. The `--needroot` flag ensures the installer automatically requests root privileges, so the install script can assume it's running as root (no `sudo` needed in the script).

## Advanced Examples

### Multiple Package Types

You can create multiple makeself packages with different configurations:

```yaml title=".goreleaser.yaml"
makeselfs:
  # Full installer with all components
  - id: full
    ids:
      - default
    name_template: "{{ .ProjectName }}_full_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    label: "{{ .ProjectName }} Full Installer"
    files:
      - configs/*
      - scripts/*
      - docs/*
    extra_args:
      - "--needroot"
    install_script: |
      #!/bin/bash
      echo "Installing {{ .ProjectName }} (Full)..."
      chmod +x {{ .Binary }}
      cp {{ .Binary }} /usr/local/bin/
      mkdir -p /etc/{{ .ProjectName }}
      cp configs/* /etc/{{ .ProjectName }}/
      echo "Full installation complete!"

  # Minimal installer with just the binary
  - id: minimal
    ids:
      - default
    name_template: "{{ .ProjectName }}_minimal_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    label: "{{ .ProjectName }} Minimal Installer"
    extension: ".bin"
    files:
      - none*  # Don't include default files
    install_script: |
      #!/bin/bash
      echo "Installing {{ .ProjectName }} (Minimal)..."
      chmod +x {{ .Binary }}
      cp {{ .Binary }} /usr/local/bin/
      echo "Minimal installation complete!"
```

### Meta Packages

Create configuration-only packages without binaries:

```yaml title=".goreleaser.yaml"
makeselfs:
  - id: config
    meta: true  # No binaries included
    name_template: "{{ .ProjectName }}_config_{{ .Version }}"
    label: "{{ .ProjectName }} Configuration Package"
    files:
      - configs/*
      - scripts/*
    install_script: |
      #!/bin/bash
      echo "Installing {{ .ProjectName }} configurations..."
      mkdir -p /etc/{{ .ProjectName }}
      cp configs/* /etc/{{ .ProjectName }}/
      chmod +x scripts/*.sh
      cp scripts/*.sh /usr/local/bin/
      echo "Configuration installed!"
```

### Using External Install Scripts

Instead of inline scripts, you can reference external files:

```yaml title=".goreleaser.yaml"
makeselfs:
  - id: installer
    ids:
      - default
    label: "{{ .ProjectName }} v{{ .Version }} Installer"
    install_script_file: "scripts/install.sh"
    files:
      - scripts/install.sh
      - scripts/uninstall.sh
      - configs/*
```

Create `scripts/install.sh`:
```bash
#!/bin/bash
set -e

echo "Installing {{ .ProjectName }}..."

# Install binary
chmod +x {{ .Binary }}
cp {{ .Binary }} /usr/local/bin/

# Install configurations
mkdir -p /etc/{{ .ProjectName }}
cp configs/* /etc/{{ .ProjectName }}/

# Install uninstaller
chmod +x scripts/uninstall.sh
cp scripts/uninstall.sh /usr/local/bin/{{ .ProjectName }}-uninstall

echo "{{ .ProjectName }} installed successfully!"
echo "Run '{{ .ProjectName }}' to get started"
echo "Run '{{ .ProjectName }}-uninstall' to remove"
```

## Makeself Features Supported

- **Custom Extensions**: Use templated extensions like `.{{ .Os }}.run` for platform-specific naming
- **LSM Support**: Include Linux Software Map information for software catalogs
- **Flexible Install Scripts**: Use either inline scripts or external script files from your repository
- **Compression Options**: Choose from multiple compression formats based on size vs. speed trade-offs
- **Integration with Files**: Add documentation, licenses, and other files that will be available to the install script
- **Root Privileges**: Use `--needroot` in `extra_args` to ensure the installer runs with root privileges, simplifying system-wide installations
- **Meta Packages**: Create configuration-only packages without binaries

## Available Template Variables

All standard GoReleaser template variables are available, including:

- `{{ .ProjectName }}` - Project name
- `{{ .Version }}` - Version being released
- `{{ .Tag }}` - Git tag
- `{{ .Binary }}` - Binary name (useful in install scripts)
- `{{ .Os }}` - Operating system
- `{{ .Arch }}` - Architecture
- `{{ .Env.VARIABLE_NAME }}` - Environment variables

## Makeself Arguments

Common `extra_args` options:

- `--needroot` - Requires root privileges to run the installer
- `--keep` - Keep extracted files after execution (useful for debugging)
- `--notemp` - Don't use a temporary directory for extraction
- `--copy` - Copy files to temp directory instead of extracting in place
- `--current` - Files will be extracted to the current directory
- `--nooverwrite` - Don't overwrite existing files

!!! tip

    The install script has access to all files included in the package, so you can reference documentation, configuration files, or other assets in your installation logic.

!!! tip "Root Privileges"

    When using `--needroot` in `extra_args`, the makeself installer will automatically prompt for root privileges when executed. This allows your install script to assume root access without using `sudo` commands, making the script simpler and more reliable. Users will be prompted like: `"This installer requires root privileges. Please enter your password when prompted."`

!!! warning

    Makeself packages are platform-specific (typically Linux and macos) and create executable files. Make sure your target users can execute them on their systems.

<!-- md:templates -->
