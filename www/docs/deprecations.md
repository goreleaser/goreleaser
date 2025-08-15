# Deprecation notices

This page is used to list deprecation notices across GoReleaser.

Deprecated options are only removed on major versions of GoReleaser.

Nevertheless, it's a good thing to keep your configuration up-to-date to prevent
any issues.

You can check your use of deprecated configurations by running:

```sh
goreleaser check
```

## Active deprecation notices

<!--

Template for new deprecations:

### property

> since v2.xx

Description.

PS: Don't forget to add it to cmd/mcp.go as well!

=== "Before"

    ```yaml
    foo: bar
    ```

=== "After"

    ```yaml
    foo: bar
    ```

-->

### homebrew_casks.manpage

> since v2.11

You may now define multiple man pages, which was not possible in v2.10.

=== "Before"

    ```yaml
    homebrew_casks:
      manpage: foo.1.gz
    ```

=== "After"

    ```yaml
    homebrew_casks:
      manpages:
        - foo.1.gz
    ```

### brews

> since v2.10

Historically, GoReleaser would generate _hackyish_ formulas that would install
the pre-compiled binaries.
This was the only way to do it for Linuxbrew at the time, but this is no longer
true, and _Casks_ should be used instead.

That said, we now have a `homebrew_casks` section!

For simple cases, simply replacing one with the other will be good enough.
More complex settings might require further change.
Check the [new documentation](./customization/homebrew_casks.md) for more
details.

Once you do the first release this way, you might also want to delete the old
_Formulas_ from your _Tap_.
You may also want to make the _Cask_ conflict with the previous _Formula_.

=== "Before"

    ```yaml
    brews:
    - name: foo
      directory: Formulas
    ```

=== "After"

    ```yaml
    homebrew_casks:
    - name: foo
      # Optional: either set it to Casks, or remove it:
      directory: Casks

      # Optional: make the old formula conflict with the cask:
      conflicts:
        - formula: foo

      # Optional: helps pass `homebrew audit` if homepage is different from download domain:
      url:
        verified: github.com/myorg/myrepo

      # Optional: if your app/binary isn't signed and notarized, you'll need this:
      hooks:
        post:
          # replace foo with the actual binary name
          install: |
            if OS.mac?
              system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/foo"]
            end
    ```

!!! warning

    Don't forget to remove the `directory: Formula` from your configuration.
    Casks **need** to be in the `Casks` directory - which is the default.

I would also recommend manually editing your Formula to disable it, e.g.:

```ruby
class Foo < Formula
  # ...
  # make sure to bump the version:
  version "1.2.3"
  # ...
  disable! date: "2025-06-10", because: "the cask should be used now instead", replacement_cask: "foo"
  # ...
end
```

With this, when the user tries to upgrade, they should see and error like so:

```
==> Upgrading 1 outdated package:
goreleaser/tap/goreleaser 2.9.0 -> 2.9.1
Error: goreleaser/tap/goreleaser has been disabled because it the cask should be used now instead! It will be disabled on 2025-06-14.
Replacement:
  brew install --cask goreleaser
```

### archives.builds

> since v2.8

The `builds` field has been replaced with the `ids`, which is the nomenclature
used everywhere else.

=== "Before"

    ```yaml
    archives:
      builds: [a, b]
    ```

=== "After"

    ```yaml
    archives:
      ids: [a, b]
    ```

### snaps.builds

> since v2.8

The `builds` field has been replaced with the `ids`, which is the nomenclature
used everywhere else.

=== "Before"

    ```yaml
    snaps:
      builds: [a, b]
    ```

=== "After"

    ```yaml
    snaps:
      ids: [a, b]
    ```

### nfpms.builds

> since v2.8

The `builds` field has been replaced with the `ids`, which is the nomenclature
used everywhere else.

=== "Before"

    ```yaml
    nfpms:
      builds: [a, b]
    ```

=== "After"

    ```yaml
    nfpms:
      ids: [a, b]
    ```

### archives.format

> since v2.6

Format was renamed to `formats`, and now accepts a list of formats.

=== "Before"

    ```yaml
    archives:
      - format: zip
    ```

=== "After"

    ```yaml
    archives:
      - formats: [ 'zip' ]
    ```

!!! tip

    It will still accept a single string, e.g.: `formats: zip`.
    In most cases you can simply rename the property to formats.

### archives.format_overrides.format

> since v2.6

Format was renamed to `formats`, and now accepts a list of formats.

!!! tip

    It will still accept a single string, e.g.: `formats: zip`.
    In most cases you can simply rename the property to formats.

=== "Before"

    ```yaml
    archives:
      - format_overrides:
        - format: zip
    ```

=== "After"

    ```yaml
    archives:
      - format_overrides:
        - formats: [ 'zip' ]
    ```

!!! tip

    It will still accept a single string, e.g.: `formats: zip`.
    In most cases you can simply rename the property to formats.

### kos.repository

> since v2.5

Use `repositories` instead. It allows to create multiple images with Ko, without
having to rebuild each of them.

=== "Before"

    ```yaml
    kos:
      - repository: foo/bar
    ```

=== "After"

    ```yaml
    kos:
      - repositories:
          - foo/bar
    ```

### builds.gobinary

> since v2.5

The property was renamed to `tool`, as to better accommodate multiple languages.

=== "Before"

    ```yaml
    builds:
      - gobinary: 'go1.2.3'
    ```

=== "After"

    ```yaml
    builds:
      - tool: 'go1.2.3'
    ```

### kos.sbom

> since v2.2

Ko removed support for `cyclonedx` and `go.version-m` SBOMs from upstream.
You can now either use `spdx` or `none`.
From now on, these two options will be replaced by `none`.
We recommend you change it to `spdx`.

### nightly.name_template

> since v2.2

Property renamed so its easier to reason about.

=== "Before"

    ```yaml
    nightly:
      name_template: 'foo'
    ```

=== "After"

    ```yaml
    nightly:
      version_template: 'foo'
    ```

### snapshot.name_template

> since v2.2

Property renamed so its easier to reason about.

=== "Before"

    ```yaml
    snapshot:
      name_template: 'foo'
    ```

=== "After"

    ```yaml
    snapshot:
      version_template: 'foo'
    ```

## Removed in v2

### archives.strip_parent_binary_folder

> since 2024-03-29 (v1.25), removed 2024-05-26 (v2.0)

Property was renamed to be consistent across all configurations.

=== "Before"

    ```yaml
    archives:
      -
        strip_parent_binary_folder: true
    ```

=== "After"

    ```yaml
    archives:
      -
        strip_binary_directory: true
    ```

### blobs.folder

> since 2024-03-29 (v1.25), removed 2024-05-26 (v2.0)

Property was renamed to be consistent across all configurations.

=== "Before"

    ```yaml
    blobs:
      -
        folder: foo
    ```

=== "After"

    ```yaml
    blobs:
      -
        directory: foo
    ```

### brews.folder

> since 2024-03-29 (v1.25), removed 2024-05-26 (v2.0)

Property was renamed to be consistent across all configurations.

=== "Before"

    ```yaml
    brews:
      -
        folder: foo
    ```

=== "After"

    ```yaml
    brews:
      -
        directory: foo
    ```

### scoops.folder

> since 2024-03-29 (v1.25), removed 2024-05-26 (v2.0)

Property was renamed to be consistent across all configurations.

=== "Before"

    ```yaml
    scoops:
      -
        folder: foo
    ```

=== "After"

    ```yaml
    scoops:
      -
        directory: foo
    ```

### furies.skip

> since 2024-03-03 (v1.25), removed 2024-05-26 (v2.0)

Changed to `disable` to conform with all other pipes.

=== "Before"

    ```yaml
    furies:
      - skip: true
    ```

=== "After"

    ```yaml
    furies:
      - disable: true
    ```

### changelog.skip

> since 2024-01-14 (v1.24), removed 2024-05-26 (v2.0)

Changed to `disable` to conform with all other pipes.

=== "Before"

    ```yaml
    changelog:
      skip: true
    ```

=== "After"

    ```yaml
    changelog:
      disable: true
    ```

### blobs.kmskey

> since 2024-01-07 (v1.24), removed 2024-05-26 (v2.0)

Changed to `kms_key` to conform with all other options.

=== "Before"

    ```yaml
    blobs:
      - kmskey: foo
    ```

=== "After"

    ```yaml
    blobs:
      - kms_key: foo
    ```

### blobs.disableSSL

> since 2024-01-07 (v1.24), removed 2024-05-26 (v2.0)

Changed to `disable_ssl` to conform with all other options.

=== "Before"

    ```yaml
    blobs:
      - disableSSL: true
    ```

=== "After"

    ```yaml
    blobs:
      - disable_ssl: true
    ```

### `--skip`

> since 2023-09-14 (v1.21), removed 2024-05-26 (v2.0)

The following `goreleaser release` flags were deprecated:

- `--skip-announce`
- `--skip-before`
- `--skip-docker`
- `--skip-ko`
- `--skip-publish`
- `--skip-sbom`
- `--skip-sign`
- `--skip-validate`

By the same token, the following `goreleaser build` flags were deprecated:

- `--skip-before`
- `--skip-post-hooks`
- `--skip-validate`

All these flags are now under a single `--skip` flag, that accepts multiple
values.

=== "Before"

    ```sh
    goreleaser build --skip-before --skip-validate
    goreleaser release --skip-validate --skip-publish
    ```

=== "After"

    ```sh
    goreleaser build --skip=before,validate
    goreleaser release --skip=validate,publish

    # or

    goreleaser build --skip=before --skip=validate
    goreleaser release --skip=validate --skip=publish
    ```

You can check `goreleaser build --help` and `goreleaser release --help` to see
the valid options, and shell autocompletion should work properly as well.

### scoops.bucket

> since 2023-06-13 (v1.19.0), removed 2024-05-26 (v2.0)

Replace `bucket` with `repository`.

=== "Before"

    ```yaml
    scoops:
      -
        bucket:
          - name: foo
            owner: bar
    ```

=== "After"

    ```yaml
    scoops:
      -
        repository:
          - name: foo
            owner: bar
    ```

### krews.index

> since 2023-06-13 (v1.19.0), removed 2024-05-26 (v2.0)

Replace `index` with `repository`.

=== "Before"

    ```yaml
    krews:
      -
        index:
          - name: foo
            owner: bar
    ```

=== "After"

    ```yaml
    krews:
      -
        repository:
          - name: foo
            owner: bar
    ```

### brews.tap

> since 2023-06-13 (v1.19.0), removed 2024-05-26 (v2.0)

Replace `tap` with `repository`.

=== "Before"

    ```yaml
    brews:
      -
        tap:
          - name: foo
            owner: bar
    ```

=== "After"

    ```yaml
    brews:
      -
        repository:
          - name: foo
            owner: bar
    ```

### archives.rlcp

> since 2023-06-06 (v1.19.0), removed 2024-05-26 (v2.0)

This option is now default and can't be changed. You can remove it from your
configuration files.

See [this](./old-deprecations.md#archivesrlcp) for more info.

### source.rlcp

> since 2023-06-06 (v1.19.0), removed 2024-05-26 (v2.0)

This option is now default and can't be changed. You can remove it from your
configuration files.

See [this](./old-deprecations.md#sourcerlcp) for more info.

### brews.plist

> since 2023-06-06 (v1.19.0), removed 2024-05-26 (v2.0)

`plist` is deprecated by Homebrew, and now on GoReleaser too. Use `service`
instead.

=== "Before"

    ```yaml
    brews:
    -
      plist: |
        <?xml version="1.0" encoding="UTF-8"?>
        <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
        <plist version="1.0">
        <dict>
        # etc ...
    ```

=== "After"

    ```yaml
    brews:
    -
      service: |
        run [opt_bin/"mybin"]
        keep_alive true
        # etc ...
    ```

### --debug

> since 2023-05-16 (v1.19.0), removed 2024-05-26 (v2.0)

`--debug` has been deprecated in favor of `--verbose`.

=== "Before"

    ```bash
    goreleaser release --debug
    ```

=== "After"

    ```bash
    goreleaser release --verbose
    ```

### scoop

> since 2023-04-30 (v1.18.0), removed 2024-05-26 (v2.0)

GoReleaser now allows many `scoop` configurations, so it should be pluralized
[accordingly](customization/scoop.md).

=== "Before"

    ```yaml
    scoop:
      # ...
    ```

=== "After"

    ```yaml
    scoops:
    - # ...
    ```

### build

> since 2023-02-09 (v1.16.0), removed 2024-05-26 (v2.0)

This option was still being supported, even though undocumented, for a couple
of years now. It's finally time to sunset it.

Simply use the pluralized form, `builds`, according to the
[documentation](./customization/builds/index.md).

=== "Before"

    ```yaml
    build:
      # ...
    ```

=== "After"

    ```yaml
    builds:
    - # ...
    ```

### --rm-dist

> since 2023-01-17 (v1.15.0), removed 2024-05-26 (v2.0)

`--rm-dist` has been deprecated in favor of `--clean`.

=== "Before"

    ```bash
    goreleaser release --rm-dist
    ```

=== "After"

    ```bash
    goreleaser release --clean
    ```

### nfpms.maintainer

> since 2022-05-07 (v1.9.0), removed 2024-05-26 (v2.0)

nFPM will soon make mandatory setting the maintainer field.

=== "Before"

    ```yaml
    nfpms:
    - maintainer: ''
    ```

=== "After"

    ```yaml
    nfpms:
    - maintainer: 'Name <email>'
    ```

## Previous versions

Deprecations that were removed in v1.x or earlier have been moved into its [own page](./old-deprecations.md).
