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

None so far!

<!--

Template for new deprecations:

### property

> since yyyy-mm-dd (v1.xx)

Description.

=== "Before"

    ```yaml
    foo: bar
    ```

=== "After"

    ```yaml
    foo: bar
    ```

-->

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

See [this](#archivesrlcp_1) for more info.

### source.rlcp

> since 2023-06-06 (v1.19.0), removed 2024-05-26 (v2.0)

This option is now default and can't be changed. You can remove it from your
configuration files.

See [this](#sourcerlcp_1) for more info.

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
[documentation](customization/builds.md).

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

The following options were deprecated in the past and were already removed.

## Removed in v1

### archives.rlcp

> since 2022-12-23 (v1.14.0), removed 2023-06-06 (v1.19.0)

This is not so much a deprecation property (yet), as it is a default behavior
change.

The usage of relative longest common path (`rlcp`) on the destination side of
archive files will be enabled by default by June 2023. Then, this option will be
deprecated, and you will have another 6 months (until December 2023) to remove
it.

For now, if you want to keep the old behavior, no action is required, but it
would be nice to have your opinion [here][rlcp-discuss].

[rlcp-discuss]: https://github.com/goreleaser/goreleaser/discussions/3659

If you want to make sure your releases will keep working properly, you can
enable this option and test it out with
`goreleaser release --snapshot --clean`.

=== "After"

    ```yaml
    archives:
    -
      rlcp: true
    ```

### source.rlcp

> since 2022-12-23 (v1.14.0), removed 2023-06-06 (v1.19.0)

Same as [`archives.rlcp`](#archivesrlcp).

=== "After"

    ```yaml
    source:
      rlcp: true
    ```

### nfpms.maintainer

> since 2022-05-07 (v1.9.0)

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

### archives.replacements

> since 2022-11-24 (v1.14.0), removed 2023-06-06 (v1.19.0)

The `replacements` will be removed soon from the archives section, as it was
never handled correctly when multiple archives were being used, and it also
causes confusion in other places.

You can still get the same features by abusing the `name_template` property.

=== "Before"

    ```yaml
    archives:
      - id: foo
        name_template: '{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}'
        replacements:
          darwin: Darwin
          linux: Linux
          windows: Windows
          386: i386
          amd64: x86_64
    ```

=== "After"

    ```yaml
    archives:
      - id: foo
        name_template: >-
          {{- .ProjectName }}_
          {{- title .Os }}_
          {{- if eq .Arch "amd64" }}x86_64
          {{- else if eq .Arch "386" }}i386
          {{- else }}{{ .Arch }}{{ end }}
          {{- if .Arm }}v{{ .Arm }}{{ end -}}
    ```

Those two configurations will yield the same results.

Notice that if you are using the `archives.name_template`, notice it also has a
`{{.Version}}` in it. Adjust the new `name_template` accordingly.

### nfpms.replacements

> since 2022-11-24 (v1.14.0), removed 2023-06-06 (v1.19.0)

The `replacements` will be removed soon from the nFPMs section.

You can still get the same features by abusing the `file_name_template` property.

=== "Before"

    ```yaml
    nfpms:
      - id: foo
        file_name_template: '{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}'
        replacements:
          darwin: Darwin
          linux: Linux
          windows: Windows
          386: i386
          amd64: x86_64
    ```

=== "After"

    ```yaml
    nfpms:
      - id: foo
        file_name_template: >-
          {{- .ProjectName }}_
          {{- title .Os }}_
          {{- if eq .Arch "amd64" }}x86_64
          {{- else if eq .Arch "386" }}i386
          {{- else }}{{ .Arch }}{{ end }}
          {{- if .Arm }}v{{ .Arm }}{{ end -}}
    ```

Those two configurations will yield the same results.

Generally speaking, is probably best to use `{{ .ConventionalFileName }}`
instead of custom templates.

### snapcrafts.replacements

> since 2022-11-24 (v1.14.0), removed 2023-06-06 (v1.19.0)

The `replacements` will be removed soon from the Snapcrafts section.

You can still get the same features by abusing the `name_template` property.

=== "Before"

    ```yaml
    snapcrafts:
      - id: foo
        name_template: '{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}'
        replacements:
          darwin: Darwin
          linux: Linux
          windows: Windows
          386: i386
          amd64: x86_64
    ```

=== "After"

    ```yaml
    snapcrafts:
      - id: foo
        name_template: >-
          {{ .ProjectName }}_
          {{- title .Os }}_
          {{- if eq .Arch "amd64" }}x86_64
          {{- else if eq .Arch "386" }}i386
          {{- else }}{{ .Arch }}{{ end }}
    ```

Those two configurations will yield the same results.

Generally speaking, is probably best to use `{{ .ConventionalFileName }}`
instead of custom templates.

### variables

> since 2022-01-20 (v1.4.0), removed 2023-05-01 (v1.18.0)

In [GoReleaser PRO](pro.md) custom variables should now be prefixed with `.Var`.

=== "Before"

    ```yaml
    variables:
      foo: bar
    some_template: 'lala-{{ .foo }}'
    ```

=== "After"

    ```yaml
    variables:
      foo: bar
    some_template: 'lala-{{ .Var.foo }}'
    ```

### dockers.use: buildpacks

> since 2022-03-16 (v1.7.0), removed 2022-09-28 (v1.12.0)

This was removed due to some issues:

- The binary gets rebuild again during the buildpacks build;
- There is no ARM support.

### rigs

> since 2022-03-21 (v1.8.0), removed 2022-08-16 (v1.11.0)

GoFish was deprecated by their authors, therefore, we're removing its
support from GoReleaser too.

### nfpms.empty_folders

> since 2021-11-14 (v1.0.0), removed 2022-06-14 (v1.10.0)

nFPM empty folders is now deprecated in favor of a `dir` content type:

=== "Before"

    ```yaml
    nfpms:
    - empty_folders:
      - /foo/bar
    ```

=== "After"

    ```yaml
    nfpms:
    - contents:
      - dst: /foo/bar
        type: dir
    ```

### builds for windows/arm64

> since 2021-08-16 (v0.175.0), removed 2022-06-12 (v1.10.0)

Since Go 1.17, `windows/arm64` is a valid target.

Prior to v0.175.0, GoReleaser would just ignore this target.
Since in Go 1.17 it is now a valid target, GoReleaser will build it if the Go version being used is 1.17 or later.

If you want to make sure it is ignored in the future, you need to add this to your build config:

```yaml
ignore:
  - goos: windows
    goarch: arm64
```

If you try to use new versions of GoReleaser with Go 1.16 or older, it will warn
about it until this deprecation warning expires, after that your build will
likely fail.

### godownloader

> since 2021-10-13 (all), removed 2022-05-18

GoDownloader, the installation script generator, wasn't updated for a long time
and is now officially deprecated.
The website and all install scripts will be taken out in 6 months.
You can still use any of the other install methods.

This also includes `install.goreleaser.com`.

Most common tools installed via that website were probably
[GoReleaser](install.md) itself and
[golangci-lint](https://golangci-lint.run/welcome/install/).

Please follow to the check their documentation for alternative install methods.

### dockers.use_buildx

> since 2021-06-26 (v0.172.0), removed 2022-03-16 (v1.7.0)

`use_buildx` is deprecated in favor of the more generalist `use`, since now it also allow other options in the future:

Change this:

=== "Before"

    ```yaml
    dockers:
      -
        use_buildx: true
    ```

=== "After"

    ```yaml
    dockers:
      -
        use: buildx
    ```

### builds for darwin/arm64

> since 2021-02-17 (v0.157.0), removed 2022-03-16 (v1.7.0)

Since Go 1.16, `darwin/arm64` is macOS on Apple Silicon instead of `iOS`.

Prior to v0.156.0, GoReleaser would just ignore this target.
Since in Go 1.16 and later it is a valid target, GoReleaser will now build it if the Go version being used is 1.16 or later.

If you want to make sure it is ignored in the future, you need to add this to your build config:

```yaml
ignore:
  - goos: darwin
    goarch: arm64
```

If you try to use new versions of GoReleaser with Go 1.15 or older, it will warn about it until this deprecation warning expires, after that your build will likely fail.

## Removed in v0.\*

### Skipping SemVer Validations

> since 2021-02-28 (v0.158.0), removed 2021-09-22 (v0.180.0)

GoReleaser skips SemVer validations when run with `--skip-validation` or `--snapshot`.
This causes other problems later, such as [invalid Linux packages](https://github.com/goreleaser/goreleaser/issues/2081).
Because of that, once this deprecation expires, GoReleaser will hard fail on non-semver versions, as stated on our [limitations page](https://goreleaser.com/limitations/semver/).

### docker.builds

> since 2021-01-07 (v0.154.0), removed 2021-08-13 (v0.175.0)

`builds` is deprecated in favor of `ids`, since now it also allows to copy nfpm packages:

Change this:

=== "Before"

    ```yaml
    dockers:
      -
        builds: ['a', 'b']
    ```

=== "After"

    ```yaml
    dockers:
      -
        ids: ['a', 'b']
    ```

### docker.binaries

> since 2021-01-07 (v0.154.0), removed 2021-08-13 (v0.175.0)

`binaries` is deprecated and now does nothing.
If you want to filter something out, use the `ids` property.

Change this:

=== "Before"

    ```yaml
    dockers:
      -
        binaries: ['foo']
    ```

=== "After"

    ```yaml
    dockers:
      -
        ids: ['foo']
    ```

### nfpms.files

> since 2020-12-21 (v0.149.0), removed 2021-07-26 (v0.172.0)

`files` is deprecated in favor of `contents` (check [this page](https://goreleaser.com/customization/nfpm/) for more details):

Change this:

=== "Before"

    ```yaml
    nfpms:
      -
        files:
          foo: bar
    ```

=== "After"

    ```yaml
    nfpms:
      -
        contents:
          - src: foo
            dst: bar
    ```

### nfpms.config_files

> since 2020-12-21 (v0.149.0), removed 2021-07-26 (v0.172.0)

`config_files` is deprecated in favor of `contents` (check [this page](https://goreleaser.com/customization/nfpm/) for more details):

Change this:

=== "Before"

    ```yaml
    nfpms:
      -
        config_files:
          foo: bar
    ```

=== "After"

    ```yaml
    nfpms:
      -
        contents:
          - src: foo
            dst: bar
            type: config
    ```

### nfpms.symlinks

> since 2020-12-21 (v0.149.0), removed 2021-07-26 (v0.172.0)

`symlinks` is deprecated in favor of `contents` (check [this page](https://goreleaser.com/customization/nfpm/) for more details):

Change this:

=== "Before"

    ```yaml
    nfpms:
      -
        symlinks:
          foo: bar
    ```

=== "After"

    ```yaml
    nfpms:
      -
        contents:
          - src: foo
            dst: bar
            type: symlink
    ```

### nfpms.rpm.ghost_files

> since 2020-12-21 (v0.149.0), removed 2021-07-26 (v0.172.0)

`rpm.ghost_files` is deprecated in favor of `contents` (check [this page](https://goreleaser.com/customization/nfpm/) for more details):

Change this:

=== "Before"

    ```yaml
    nfpms:
      -
        rpm:
          ghost_files:
            - foo
    ```

=== "After"

    ```yaml
    nfpms:
      -
        contents:
          - dst: bar
            type: ghost
            packager: rpm # optional
    ```

### nfpms.rpm.config_noreplace_files

> since 2020-12-21 (v0.149.0), removed 2021-07-26 (v0.172.0)

`rpm.config_noreplace_files` is deprecated in favor of `contents` (check [this page](https://goreleaser.com/customization/nfpm/) for more details):

Change this:

=== "Before"

    ```yaml
    nfpms:
      -
        rpm:
          config_noreplace_files:
            foo: bar
    ```

=== "After"

    ```yaml
    nfpms:
      -
        contents:
          - src: foo
            dst: bar
            type: config|noreplace
            packager: rpm # optional
    ```

### nfpms.deb.version_metadata

> since 2020-12-21 (v0.149.0), removed 2021-07-26 (v0.172.0)

`deb.version_metadata` is deprecated in favor of `version_metadata` (check [this page](https://goreleaser.com/customization/nfpm/) for more details):

Change this:

=== "Before"

    ```yaml
    nfpms:
      -
        deb:
          version_metadata: beta1
    ```

=== "After"

    ```yaml
    nfpms:
      -
        version_metadata: beta1
    ```

### brews.github

> since 2020-07-06 (v0.139.0), removed 2021-01-04 (v0.152.0)

GitHub section was deprecated in favour of `tap` which
reflects Homebrew's naming convention. GitHub will be picked
automatically when GitHub token is passed.

Change this:

=== "Before"

    ```yaml
    brews:
      -
        github:
          owner: goreleaser
          name: homebrew-tap
    ```

=== "After"

    ```yaml
    brews:
      -
        tap:
          owner: goreleaser
          name: homebrew-tap
    ```

### brews.gitlab

> since 2020-07-06 (v0.139.0), removed 2021-01-04 (v0.152.0)

GitLab section was deprecated in favour of `tap` which
reflects Homebrew's naming convention. GitLab will be picked
automatically when GitLab token is passed.

Change this:

=== "Before"

    ```yaml
    brews:
      -
        gitlab:
          owner: goreleaser
          name: homebrew-tap
    ```

=== "After"

    ```yaml
    brews:
      -
        tap:
          owner: goreleaser
          name: homebrew-tap
    ```

### puts

> since 2019-11-15, removed 2020-04-14 (v0.132.0)

The HTTP upload support was extended to also accept `POST` as a method,
so the name `puts` kind of lost its meaning.

=== "Before"

    ```yaml
    puts:
    - ...
    ```

=== "After"

    ```yaml
    uploads:
    - ...
    ```

Also note that secrets environment variable name prefixes have changed from
`PUT_` to `UPLOAD_`.

### nfpms.name_template

> since 2019-11-15, removed 2020-04-14 (v0.132.0)

The `name_template` field was deprecated in favor of a more clear one,
`file_name_template`.

=== "Before"

    ```yaml
    nfpms:
    - name_template: foo
    ```

=== "After"

    ```yaml
    nfpms:
    - file_name_template: foo
    ```

### blob

> since 2019-08-02, removed 2020-03-22 (v0.130.0)

Blob was deprecated in favor of its plural form.
It was already accepting multiple inputs, but its pluralized now so its more
clear.

=== "Before"

    ```yaml
    blob:
      # etc
    ```

=== "After"

    ```yaml
    blobs:
      # etc
    ```

### sign

> since 2019-07-20, removed 2020-03-22 (v0.130.0)

Sign was deprecated in favor of its plural form.

=== "Before"

    ```yaml
    sign:
      # etc
    ```

=== "After"

    ```yaml
    signs:
      -
        # etc
    ```

### brew

> since 2019-06-09, removed 2020-01-26 (v0.125.0)

Brew was deprecated in favor of its plural form.

Change this:

=== "Before"

    ```yaml
    brew:
      # etc
    ```

=== "After"

    ```yaml
    brews:
      -
        # etc
    ```

### s3

> since 2019-06-09, removed 2020-01-07 (v0.125.0)

S3 was deprecated in favor of the new `blob`, which supports S3, Azure Blob and
GCS.

=== "Before"

    ```yaml
    s3:
    -
      # etc
    ```

=== "After"

    ```yaml
    blobs:
    -
      provider: s3
      # etc
    ```

ACLs should be set on the bucket, the `acl` option does not exist anymore.

### archive

> since 2019-04-16, removed 2019-12-27 (v0.124.0)

We now allow multiple archives, so the `archive` statement will be removed.

=== "Before"

    ```yaml
    archive:
      format: zip
    ```

=== "After"

    ```yaml
    archives:
      - id: foo
        format: zip
    ```

### snapcraft

> since 2019-05-27, removed 2019-12-27 (v0.124.0)

We now allow multiple Snapcraft configs, so the `snapcraft` statement will be removed.

=== "Before"

    ```yaml
    snapcraft:
      publish: true
      # ...
    ```

=== "After"

    ```yaml
    snapcrafts:
      -
        publish: true
        # ...
    ```

### nfpm

> since 2019-05-07, removed 2019-12-27 (v0.124.0)

We now allow multiple NFPM config, so the `nfpm` statement will be removed.

=== "Before"

    ```yaml
    nfpm:
      formats:
        - deb
    ```

=== "After"

    ```yaml
    nfpms:
      -
        formats:
          - deb
    ```

### docker.binary

> since 2018-10-01, removed 2019-08-02 (v0.114.0)

You can now create a Docker image with multiple binaries.

=== "Before"

    ```yaml
    dockers:
    - image: foo/bar
      binary: foo
    ```

=== "After"

    ```yaml
    dockers:
    - image: foo/bar
      binaries:
      - foo
    ```

### docker.image

> since 2018-10-20, removed 2019-08-02 (v0.114.0)

This property was deprecated in favor of more flexible `image_templates`.
The idea is to be able to define several images and tags using templates instead of just one image with tag templates.
This flexibility allows images to be pushed to multiple registries.

=== "Before"

    ```yaml
    dockers:
    - image: foo/bar
      tag_templates:
        - '{{ .Tag }}'
    ```

=== "After"

    ```yaml
    dockers:
    - image_templates:
        - 'foo/bar:{{ .Tag }}'
    ```

### docker.tag_templates

> since 2018-10-20, removed 2019-08-02 (v0.114.0)

This property was deprecated in favor of more flexible `image_templates`.
The idea is to be able to define several images and tags using templates instead of just one image with tag templates.

=== "Before"

    ```yaml
    dockers:
    - image: foo/bar
      tag_templates:
        - '{{ .Tag }}'
    ```

=== "After"

    ```yaml
    dockers:
    - image_templates:
        - 'foo/bar:{{ .Tag }}'
    ```

### git.short_hash

> since 2018-10-03, removed 2019-01-19 (v0.98.0)

This property was being used to tell GoReleaser to use short git hashes
instead of the full ones. This has been removed in favor of specific
template variables (`.FullCommit` and `.ShortCommit`).

=== "Before"

    ```yaml
    git:
      short_hash: true

    fake:
      foo_template: 'blah {{ .Commit }}'
    ```

=== "After"

    ```yaml
    fake:
      foo_template: 'blah {{ .ShortCommit }}'
    ```

### fpm

> since 2018-02-17, removed 2017-08-15 (v0.83.0)

FPM is deprecated in favor of nfpm, which is a simpler alternative written
in Go. The objective is to remove the ruby dependency thus simplify the
CI/CD pipelines.

Just replace the `fpm` keyword by `nfpm` in your `.goreleaser.yaml` file.

=== "Before"

    ```yaml
    fpm:
      # ...
    ```

=== "After"

    ```yaml
    nfpm:
      # ...
    ```

### docker.tag_template

> since 2018-01-19, removed 2017-08-15 (v0.83.0)

This property was deprecated in favor of the pluralized `tag_templates`.
The idea is to be able to define several tags instead of just one.

=== "Before"

    ```yaml
    dockers:
    - image: foo/bar
      tag_template: '{{ .Tag }}'
    ```

=== "After"

    ```yaml
    dockers:
    - image: foo/bar
      tag_templates:
        - '{{ .Tag }}'
    ```

### docker.latest

> since 2018-01-19, removed 2017-08-15 (v0.83.0)

The `latest` field in Docker config is deprecated in favor of the newer
`tag_templates` field.

=== "Before"

    ```yaml
    dockers:
    - image: foo/bar
      latest: true
    ```

=== "After"

    ```yaml
    dockers:
    - image: foo/bar
      tag_templates:
        - '{{ .Tag }}'
        - latest
    ```
