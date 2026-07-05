---
title: "nFPM - Linux and Windows packages"
linkTitle: nFPM
weight: 30
---

GoReleaser can be wired to [nfpm](https://github.com/goreleaser/nfpm) to
generate and publish `.deb`, `.rpm`, `.apk`, `.ipk`, Archlinux, and Windows
`.msix` packages.

Most of the options below are Linux oriented. The `msix` format packages
**Windows** binaries instead — see [its section below](#a-note-about-msix) for
how it differs.

Available options:

```yaml {filename=".goreleaser.yaml"}
nfpms:
  # note that this is an array of nfpm configs
  - #
    # ID of the nfpm config, must be unique.
    #
    # Default: 'default'.
    id: foo

    # Name of the package.
    #
    # Default: ProjectName.
    # Templates: allowed.
    package_name: foo

    # You can change the file name of the package.
    #
    # Default: '{{ .PackageName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}'.
    # Templates: allowed.
    file_name_template: "{{ .ConventionalFileName }}"

    # IDs of the builds which should be archived in this package.
    #
    # {{< g_inline_version "v2.8" >}} (use 'builds' in previous versions)
    # Default: empty (include all).
    ids:
      - foo
      - bar

    # Allows to further filter the artifacts.
    #
    # Artifacts that do not match this expression will be ignored.
    #
    # {{< g_inline_pro >}}
    # {{< g_inline_version "v2.4" >}}
    # Templates: allowed.
    if: '{{ eq .Os "linux" }}'

    # Your app's vendor.
    vendor: Drum Roll Inc.

    # Your app's homepage.
    #
    # Default: inferred from global metadata.
    homepage: https://example.com/

    # Your app's maintainer (probably you).
    #
    # Default: inferred from global metadata.
    maintainer: Drummer <drum-roll@example.com>

    # Your app's description.
    #
    # Default: inferred from global metadata.
    description: |-
      Drum rolls installer package.
      Software to create fast and easy drum rolls.

    # Your app's license.
    #
    # Default: inferred from global metadata.
    license: Apache 2.0

    # Formats to be generated.
    formats:
      - apk
      - deb
      - rpm
      - termux.deb
      - archlinux
      - ipk
      # msix packages Windows binaries instead of Linux ones.
      - msix

    # Umask to be used on files without explicit mode set. (overridable)
    #
    # Default: 0o002 (will remove world-writable permissions).
    umask: 0o002

    # Packages your package depends on. (overridable)
    dependencies:
      - git
      - zsh

    # Packages it provides. (overridable)
    provides:
      - bar

    # Packages your package recommends installing. (overridable)
    recommends:
      - bzr
      - gtk

    # Packages your package suggests installing. (overridable)
    suggests:
      - cvs
      - ksh

    # Packages that conflict with your package. (overridable)
    conflicts:
      - svn
      - bash

    # Packages it replaces. (overridable)
    replaces:
      - fish

    # Path that the binaries should be installed.
    #
    # Ignored by the `msix` format, which always places binaries at the
    # package root.
    #
    # Default: '/usr/bin'.
    bindir: /usr/bin

    # Paths to the directories where to put specific types of libraries that
    # GoReleaser built.
    #
    # This should be used together with `builds.buildmode`
    #
    # Templates: allowed.
    libdirs:
      # Default: '/usr/include'.
      header: /usr/include/something

      # Default: '/usr/lib'.
      cshared: /usr/lib/foo

      # Default: '/usr/lib'.
      carchive: /usr/lib/foobar

    # Version Epoch.
    #
    # Default: extracted from `version` if it is semver compatible.
    epoch: 2

    # Version Prerelease.
    #
    # Default: extracted from `version` if it is semver compatible.
    prerelease: beta1

    # Version Metadata (previously deb.metadata).
    # Setting metadata might interfere with version comparisons depending on the
    # packager.
    #
    # Default: extracted from `version` if it is semver compatible.
    version_metadata: git

    # Version Release.
    release: 1

    # Section.
    section: default

    # Priority.
    priority: extra

    # Makes a meta package - an empty package that contains only supporting
    # files and dependencies.
    # When set to `true`, the `builds` option is ignored.
    meta: true

    # Changelog YAML file, see: https://github.com/goreleaser/chglog
    #
    # You can use goreleaser/chglog to create the changelog for your project,
    # pass that changelog yaml file to GoReleaser,
    # and it should in turn setup it accordingly for the given available
    # formats (deb and rpm at the moment).
    #
    # Experimental.
    changelog: ./foo.yml

    # The GOAMD64 variants to package.
    #
    # Note that albeit GoReleaser will build the package, it might not be
    # supported by the underlying distribution.
    # If you want to be safe, build only for `v1`.
    # Generally, most people don't build for more than one GOAMD64, so probably
    # you don't need to worry about this.
    #
    # {{< g_inline_version "v2.14" >}}
    goamd64:
      - v1
      - v3

    # Contents to add to the package.
    # GoReleaser will automatically add the binaries.
    contents:
      # Basic file that applies to all packagers
      - src: path/to/foo
        dst: /usr/bin/foo

      # This will add all files in some/directory or in subdirectories at the
      # same level under the directory /etc. This means the tree structure in
      # some/directory will not be replicated.
      - src: some/directory/
        dst: /etc

      # This will replicate the directory structure under some/directory at
      # /etc, using the "tree" type.
      #
      # Templates: allowed.
      - src: some/directory/
        dst: /etc
        type: tree
        file_info:
          # File mode.
          mode: 0644
          # Modification time.
          #
          # Templates: allowed. {{< g_inline_version "v2.6" >}}
          mtime: "{{.CommitDate}}"

          # Owner name.
          #
          # Templates: allowed. {{< g_inline_version "v2.6" >}}
          owner: notRoot

          # Group name.
          #
          # Templates: allowed. {{< g_inline_version "v2.6" >}}
          group: notRoot

      # Simple config file
      - src: path/to/foo.conf
        dst: /etc/foo.conf
        type: config

      # Simple symlink.
      # Corresponds to `ln -s /sbin/foo /usr/bin/foo`
      - src: /sbin/foo
        dst: /usr/bin/foo
        type: "symlink"

      # Corresponds to `%config(noreplace)` if the packager is rpm, otherwise it
      # is just a config file
      - src: path/to/local/bar.conf
        dst: /etc/bar.conf
        type: "config|noreplace"

      # The src and dst attributes also supports name templates
      - src: path/{{ .Os }}-{{ .Arch }}/bar.conf
        dst: /etc/foo/bar-{{ .ProjectName }}.conf

    # Additional templated contents to add to the archive.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the package.
    #
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed.
    templated_contents:
      # a more complete example, check the globbing deep dive below
      - src: "LICENSE.md.tpl"
        dst: LICENSE.md

      # These files are not actually present in the package, but the file names
      # are added to the package header. From the RPM directives documentation:
      #
      # "There are times when a file should be owned by the package but not
      # installed - log files and state files are good examples of cases you
      # might desire this to happen."
      #
      # "The way to achieve this, is to use the %ghost directive. By adding this
      # directive to the line containing a file, RPM will know about the ghosted
      # file, but will not add it to the package."
      #
      # For non rpm packages ghost files are ignored at this time.
      - dst: /etc/casper.conf
        type: ghost
      - dst: /var/log/boo.log
        type: ghost

      # You can use the packager field to add files that are unique to a
      # specific packager
      - src: path/to/rpm/file.conf
        dst: /etc/file.conf
        type: "config|noreplace"
        packager: rpm
      - src: path/to/deb/file.conf
        dst: /etc/file.conf
        type: "config|noreplace"
        packager: deb
      - src: path/to/apk/file.conf
        dst: /etc/file.conf
        type: "config|noreplace"
        packager: apk

      # Sometimes it is important to be able to set the mtime, mode, owner, or
      # group for a file that differs from what is on the local build system at
      # build time.
      - src: path/to/foo
        dst: /usr/local/foo
        file_info:
          # File mode.
          mode: 0644
          # Modification time.
          #
          # Templates: allowed. {{< g_inline_version "v2.6" >}}
          mtime: "{{.CommitDate}}"

          # Owner name.
          #
          # Templates: allowed. {{< g_inline_version "v2.6" >}}
          owner: notRoot

          # Group name.
          # Templates: allowed. {{< g_inline_version "v2.6" >}}
          #
          group: notRoot

      # If `dst` ends with a `/`, it'll create the given path and copy the given
      # `src` into it, the same way `cp` works with and without trailing `/`.
      - src: ./foo/bar/*
        dst: /usr/local/myapp/

      # Using the type 'dir', empty directories can be created. When building
      # RPMs, however, this type has another important purpose: Claiming
      # ownership of that directory. This is important because when upgrading or
      # removing an RPM package, only the directories for which it has claimed
      # ownership are removed. However, you should not claim ownership of a
      # directory that is created by the OS or a dependency of your package.
      #
      # A directory in the build environment can optionally be provided in the
      # 'src' field in order copy mtime and mode from that directory without
      # having to specify it manually.
      - dst: /some/dir
        type: dir
        file_info:
          mode: 0700

    # Scripts to execute during the installation of the package. (overridable)
    #
    # Keys are the possible targets during the installation process
    # Values are the paths to the scripts which will be executed.
    #
    # Templates: allowed.
    scripts:
      preinstall: "scripts/preinstall.sh"
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"
      postremove: "scripts/postremove.sh"

    # Templated scripts to execute during the installation of the package. (overridable)
    #
    # Keys are the possible targets during the installation process
    # Values are the paths to the scripts which will be executed.
    #
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed.
    templated_scripts:
      preinstall: "scripts/preinstall.sh"
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"
      postremove: "scripts/postremove.sh"

    # Date to be used as mtime for the package itself, and its internal files.
    # You may also want to set the mtime on its contents.
    #
    # {{< g_inline_version "v2.6" >}}
    # Templates: allowed.
    mtime: "{{ .CommitDate }}"

    # All fields above marked as `overridable` can be overridden for a given
    # package format in this section.
    overrides:
      # The dependencies override can for example be used to provide version
      # constraints for dependencies where  different package formats use
      # different versions or for dependencies that are named differently.
      deb:
        dependencies:
          - baz (>= 1.2.3-0)
          - some-lib-dev
        # ...
      rpm:
        dependencies:
          - baz >= 1.2.3-0
          - some-lib-devel
        # ...
      apk:
        # ...

    # Custom configuration applied only to the RPM packager.
    rpm:
      # RPM specific scripts.
      scripts:
        # The pretrans script runs before all RPM package transactions / stages.
        pretrans: ./scripts/pretrans.sh
        # The posttrans script runs after all RPM package transactions / stages.
        posttrans: ./scripts/posttrans.sh

      # The package summary.
      #
      # Default: first line of the description.
      summary: Explicit Summary for Sample Package

      # The package group.
      # This option is deprecated by most distros but required by old distros
      # like CentOS 5 / EL 5 and earlier.
      group: Unspecified

      # The packager is used to identify the organization that actually packaged
      # the software, as opposed to the author of the software.
      # `maintainer` will be used as fallback if not specified.
      # This will expand any env var you set in the field, eg packager: ${PACKAGER}
      packager: GoReleaser <staff@goreleaser.com>

      # The hostname of the machine the rpm was built with.
      #
      # Default: os.Hostname()
      # {{< g_inline_version "v2.10" >}}
      buildhost: foo.bar

      # Compression algorithm (gzip (default), lzma or xz).
      compression: lzma

      # Prefixes for relocatable packages.
      prefixes:
        - /usr/bin

      # The package is signed if a key_file is set
      signature:
        # PGP secret key file path (can also be ASCII-armored).
        #
        # See "Signing key passphrases" below for more information.
        #
        # Templates: allowed.
        key_file: "{{ .Env.GPG_KEY_PATH }}"

    # Custom configuration applied only to the Deb packager.
    deb:
      # Lintian overrides
      lintian_overrides:
        - statically-linked-binary
        - changelog-file-missing-in-native-package

      # Custom deb special files.
      scripts:
        # Deb rules script.
        rules: foo.sh

        # Deb templates file, when using debconf.
        templates: templates

      # Custom deb triggers
      triggers:
        # register interest on a trigger activated by another package
        # (also available: interest_await, interest_noawait)
        interest:
          - some-trigger-name

        # activate a trigger for another package
        # (also available: activate_await, activate_noawait)
        activate:
          - another-trigger-name

      # Packages which would break if this package would be installed.
      # The installation of this package is blocked if `some-package`
      # is already installed.
      breaks:
        - some-package

      # Data compression algorithm (gzip (default), xz, zstd or none).
      #
      # {{< g_inline_version "v2.14" >}}
      compression: zstd

      # The package is signed if a key_file is set
      signature:
        # PGP secret key file path (can also be ASCII-armored).
        #
        # See "Signing key passphrases" below for more information.
        #
        # Templates: allowed.
        key_file: "{{ .Env.GPG_KEY_PATH }}"

        # The type describes the signers role, possible values are "origin",
        # "maint" and "archive".
        #
        # Default: 'origin'.
        type: origin

      # Additional fields for the control file. Empty fields are ignored.
      # This will expand any env vars you set in the field values, e.g. Vcs-Browser: ${CI_PROJECT_URL}
      fields:
        Bugs: https://github.com/goreleaser/nfpm/issues

      # The Debian-specific "predepends" field can be used to ensure the complete installation of a list of
      # packages (including unpacking, pre- and post installation scripts) prior to the installation of the
      # built package.
      predepends:
        - baz (>= 1.2.3-0)

    apk:
      # APK specific scripts.
      scripts:
        # The preupgrade script runs before APK upgrade.
        preupgrade: ./scripts/preupgrade.sh

        # The postupgrade script runs after APK.
        postupgrade: ./scripts/postupgrade.sh

      # The package is signed if a key_file is set
      signature:
        # PGP secret key file path (can also be ASCII-armored).
        #
        # See "Signing key passphrases" below for more information.
        #
        # Templates: allowed.
        key_file: "{{ .Env.GPG_KEY_PATH }}"

        # The name of the signing key. When verifying a package, the signature
        # is matched to the public key store in /etc/apk/keys/<key_name>.rsa.pub.
        #
        # Default: maintainer's email address.
        # Templates: allowed.
        key_name: origin

    archlinux:
      # Archlinux-specific scripts
      scripts:
        # The preupgrade script runs before pacman upgrades the package.
        preupgrade: ./scripts/preupgrade.sh

        # The postupgrade script runs after pacman upgrades the package.
        postupgrade: ./scripts/postupgrade.sh

      # The pkgbase can be used to explicitly specify the name to be used to refer
      # to a group of packages. See: https://wiki.archlinux.org/title/PKGBUILD#pkgbase.
      pkgbase: foo

      # The packager refers to the organization packaging the software, not to be confused
      # with the maintainer, which is the person who maintains the software.
      packager: GoReleaser <staff@goreleaser.com>

    # Custom configuration applied only to the IPK packager.
    #
    # {{< g_inline_version "v2.1" >}}
    ipk:
      # The ABI version to specify.
      #
      # Default: none
      abi_version:

      # Alternate names for files created using symlinks
      #
      # Default: none
      alternatives:
        - #
          # The IPK priority used when creating the alternative link.
          priority: 4

          # The target path and file the alternative is linked to.
          target: /usr/bin/ls

          # The alternative path and file created.
          link_name: /usr/bin/alternate_ls

      # Mark the package to be auto installed.
      #
      # Default: false
      auto_install: false

      # Mark the package as essential.
      #
      # Default: false
      essential: false

      # Additional fields for the control file. Empty fields are ignored.
      # This will expand any env vars you set in the field values, e.g. Vcs-Browser: ${CI_PROJECT_URL}
      #
      # Default: none
      fields:
        Bugs: https://github.com/goreleaser/nfpm/issues

      # The IPK-specific "predepends" field can be used to ensure the complete installation of a list of
      # packages (including unpacking, pre- and post installation scripts) prior to the installation of the
      # built package.
      #
      # Default: none
      predepends:
        - baz

      # A list of tags to associate with the package.
      #
      # Default: none
      tags:
        - foo

    # Custom configuration applied only to the MSIX packager.
    #
    # Experimental.
    #
    # MSIX packages Windows binaries. Note that, unlike the Linux formats,
    # `bindir` does not apply: binaries are always placed at the root of the
    # package, so the `executable` of each application below is simply the
    # binary's file name.
    #
    # {{< g_inline_version "v2.17" >}}
    msix:
      # The publisher identity. Must match the subject of the signing
      # certificate. Required.
      #
      # Templates: allowed.
      publisher: "CN=MyCompany, O=MyCompany, C=US"

      # Architecture in MSIX nomenclature (x64, x86, arm64, arm, neutral).
      #
      # Default: derived from the binary's GOARCH.
      arch: x64

      # Package identity.
      identity:
        # Resource identifier.
        resource_id: MyApp

      # Package properties.
      properties:
        # Display name shown to users.
        #
        # Default: the package name.
        # Templates: allowed.
        display_name: My App

        # Publisher display name shown to users.
        #
        # Default: the package name.
        # Templates: allowed.
        publisher_display_name: My Company

        # Path to the package logo. Required.
        #
        # Templates: allowed.
        logo: ./assets/logo.png

      # Applications declared in the package. At least one is required.
      applications:
        - # Application ID. Required.
          id: MyApp

          # Path to the executable inside the package. Required.
          executable: myapp.exe

          # Entry point.
          #
          # Default: 'Windows.FullTrustApplication'.
          entry_point: Windows.FullTrustApplication

          # Visual elements for this application.
          visual_elements:
            # Default: the package name.
            display_name: My App
            # Default: the package description.
            description: Does great things.
            # Default: 'transparent'.
            background_color: transparent
            # Default: the package logo.
            square150x150_logo: ./assets/logo150.png
            # Default: the package logo.
            square44x44_logo: ./assets/logo44.png

      # Target device families.
      #
      # Default: Windows.Desktop, 10.0.17763.0 to 10.0.22621.0.
      dependencies:
        target_device_families:
          - name: Windows.Desktop
            min_version: 10.0.17763.0
            max_version_tested: 10.0.22621.0

      # Declared capabilities.
      capabilities:
        capabilities:
          - internetClient
        device_capabilities:
          - location
        # 'runFullTrust' is added automatically for FullTrust applications.
        restricted:
          - runFullTrust

      # Signing configuration.
      signature:
        # Path to a PFX certificate file used to sign the package.
        #
        # The passphrase is taken from the environment, see "Signing key
        # passphrases" below (use the 'MSIX' format).
        #
        # Templates: allowed.
        pfx_file: ./certs/signing.pfx
```

> [!WARNING]
> If you use `ConventionalFileName`, make sure to replace '~' with some
> other character. GitHub will replace tilde with a dot, which will render
> the checksums invalid as filenames will not match.

{{< g_templates >}}

> [!NOTE]
> Fields marked with "overridable" can be overridden for any format.

## Signing key passphrases

GoReleaser will try to get the password from the following environment
variables, in the following order of preference:

1. `$NFPM_[ID]_[FORMAT]_PASSPHRASE`
1. `$NFPM_[ID]_PASSPHRASE`
1. `$NFPM_PASSPHRASE`

Basically, it'll start from the most specific to the most generic.
Also, `[ID]` is the uppercase `id` value, and `[FORMAT]` is the uppercase format
(`deb`, `rpm`, `msix`, etc).

So, if your `nfpms.id` is `default`, then the deb-specific passphrase
will be set `$NFPM_DEFAULT_DEB_PASSPHRASE`. GoReleaser will try that, then
`$NFPM_DEFAULT_PASSPHRASE`, and finally, `$NFPM_PASSPHRASE`.

## A note about MSIX

{{< g_version "v2.17" >}}

{{< g_experimental "https://github.com/goreleaser/goreleaser/issues/6519" >}}

Unlike the other formats, `msix` packages **Windows** binaries, not Linux ones.
When `msix` is one of the `formats`, GoReleaser feeds Windows binaries to the
nfpm pipe; the Linux formats ignore those binaries, and `msix` ignores the
non-Windows ones. This means a single `nfpms` entry can list both Linux formats
and `msix` — each binary ends up only in the package that matches its platform.

A few things differ from the Linux formats:

- `msix.publisher` and `msix.properties.logo` are **required**, and at least one
  `msix.applications` entry (with `id` and `executable`) must be provided.
- `bindir` does not apply to `msix`: binaries are always placed at the root of
  the package, so each application's `executable` is simply the binary's file
  name (e.g. `myapp.exe`).
- Symlinks are not supported and are skipped.
- The version is converted to MSIX's four-part `Major.Minor.Build.Revision`
  format; non-numeric parts default to `0`.
- To sign the package, set `msix.signature.pfx_file` and provide the passphrase
  via the environment (see [Signing key passphrases](#signing-key-passphrases),
  using `MSIX` as the format).

**Example mixing Linux and Windows packages:**

```yaml {filename=".goreleaser.yaml"}
nfpms:
  - formats: [deb, rpm, msix]
    bindir: /usr/bin
    msix:
      publisher: "CN=MyCompany"
      properties:
        logo: ./assets/logo.png
      applications:
        - id: MyApp
          executable: myapp.exe
```

## A note about Termux

Termux is the same format as `deb`, the differences are:

- it uses a different file structure (`/data/data/com.termux/files/`)
- `bindir` is automatically adjusted, but other files might require extra
  configuration (see bellow)
- it uses slightly different architecture names than Debian
- it will only package binaries built for Android

**Example prefixing other files:**

```yaml {filename=".goreleaser.yaml"}
nfpms:
  - formats: [deb termux.deb rpm]
    contents:
      - src: ./foo.conf
        dst: '{{ if eq .Format "termux.deb" }}/data/data/com.termux/files{{ end }}/usr/share/foo.conf'
```

## Conventional file names, Debian, and ARMv6

On Debian, both ARMv6 and ARMv7 have the same architecture name: `armhf`.

If you use `{{.ConventionalFileName}}`, and build for both architectures, you'll
get duplicated file names.

You can go around that with something like this:

```yaml {filename=".goreleaser.yaml"}
nfpms:
  - # ...
    file_name_template: >-
      {{- trimsuffix .ConventionalFileName .ConventionalExtension -}}
      {{- if and (eq .Arm "6") (eq .ConventionalExtension ".deb") }}6{{ end -}}
      {{- if not (eq .Amd64 "v1")}}{{ .Amd64 }}{{ end -}}
      {{- .ConventionalExtension -}}

    # ...
```
