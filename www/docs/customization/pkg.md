# macOS Pkg

<!-- md:pro -->
<!-- md:version v2.14 -->

GoReleaser can create macOS `.pkg` installer files using `pkgbuild`.

The `pkgs` section specifies how the installers should be created:

```yaml title=".goreleaser.yaml"
pkgs:
  - # ID of the resulting installer.
    #
    # Default: the project name.
    id: foo

    # Filename of the installer (without the extension).
    #
    # Default: '{{.ProjectName}}_{{.Arch}}'.
    # Templates: allowed.
    name: 'myproject{{ if neq .Arch "all" }}-{{.Arch}}{{ end }}'

    # IDs of the builds to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # Which kind of artifact to package.
    #
    # Valid options are:
    # - 'binary':    binary files
    # - 'appbundle': app bundles
    #
    # Default: 'binary'.
    use: binary

    # Allows to further filter the artifacts.
    #
    # Artifacts that do not match this expression will be ignored.
    #
    # Templates: allowed.
    if: '{{ eq .Arch "arm64" }}'

    # The package identifier (reverse domain notation).
    #
    # Required.
    # Templates: allowed.
    identifier: com.example.myapp

    # The path where the binary will be installed.
    #
    # Default: '/usr/local/bin'.
    # Templates: allowed.
    install_location: /usr/local/bin

    # Path to a directory containing pre/postinstall scripts.
    # The directory should contain scripts named 'preinstall' and/or 'postinstall'.
    # These scripts will be executed during package installation.
    #
    # Templates: allowed.
    scripts: ./scripts

    # Whether to remove the archives from the artifact list.
    # If left as false, your end release will have both the archives and the
    # pkg files.
    replace: true

    # Set the modified timestamp on the output pkg, typically
    # you would do this to ensure a build was reproducible. Pass an
    # empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"
```

<!-- md:templates -->
