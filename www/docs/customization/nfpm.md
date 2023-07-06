# Linux packages (via nFPM)

GoReleaser can be wired to [nfpm](https://github.com/goreleaser/nfpm) to
generate and publish `.deb`, `.rpm`, `.apk`, and Archlinux packages.

Available options:

```yaml
# .goreleaser.yaml
nfpms:
  # note that this is an array of nfpm configs
  - #
    # ID of the nfpm config, must be unique.
    #
    # Default: 'default'
    id: foo

    # Name of the package.
    # Default: ProjectName
    # Templates: allowed (since v1.18)
    package_name: foo

    # You can change the file name of the package.
    #
    # Default: '{{ .PackageName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}'
    # Templates: allowed
    file_name_template: "{{ .ConventionalFileName }}"

    # Build IDs for the builds you want to create NFPM packages for.
    # Defaults empty, which means no filtering.
    builds:
      - foo
      - bar

    # Your app's vendor.
    vendor: Drum Roll Inc.

    # Your app's homepage.
    homepage: https://example.com/

    # Your app's maintainer (probably you).
    maintainer: Drummer <drum-roll@example.com>

    # Your app's description.
    description: |-
      Drum rolls installer package.
      Software to create fast and easy drum rolls.

    # Your app's license.
    license: Apache 2.0

    # Formats to be generated.
    formats:
      - apk
      - deb
      - rpm
      - termux.deb # Since: v1.11
      - archlinux # Since: v1.13

    # Umask to be used on files without explicit mode set. (overridable)
    #
    # Default: 0o002 (will remove world-writable permissions)
    # Since: v1.19
    umask: 0o002

    # Packages your package depends on. (overridable)
    dependencies:
      - git
      - zsh

    # Packages it provides. (overridable)
    #
    # Since: v1.11
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
    # Default: '/usr/bin'
    bindir: /usr/bin

    # Version Epoch.
    # Default: extracted from `version` if it is semver compatible
    epoch: 2

    # Version Prerelease.
    # Default: extracted from `version` if it is semver compatible
    prerelease: beta1

    # Version Metadata (previously deb.metadata).
    # Setting metadata might interfere with version comparisons depending on the
    # packager.
    #
    # Default: extracted from `version` if it is semver compatible
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
    #
    # Default: false
    meta: true

    # Changelog YAML file, see: https://github.com/goreleaser/chglog
    #
    # You can use goreleaser/chglog to create the changelog for your project,
    # pass that changelog yaml file to GoReleaser,
    # and it should in turn setup it accordingly for the given available
    # formats (deb and rpm at the moment).
    #
    # Experimental.
    # Since: v1.11
    changelog: ./foo.yml

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
      # Since: v1.17
      # Templates: allowed
      - src: some/directory/
        dst: /etc
        type: tree

      # Simple config file
      - src: path/to/foo.conf
        dst: /etc/foo.conf
        type: config

      # Simple symlink.
      # Corresponds to `ln -s /sbin/foo /usr/local/bin/foo`
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
    # Since: v1.17 (pro)
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed
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
          mode: 0644
          mtime: 2008-01-02T15:04:05Z
          owner: notRoot
          group: notRoot

      # If `dst` ends with a `/`, it'll create the given path and copy the given
      # `src` into it, the same way `cp` works with and without trailing `/`.
      - src: ./foo/bar/*
        dst: /usr/local/myapp/

      # Using the type 'dir', empty directories can be created. When building
      # RPMs, however, this type has another important purpose: Claiming
      # ownership of that folder. This is important because when upgrading or
      # removing an RPM package, only the directories for which it has claimed
      # ownership are removed. However, you should not claim ownership of a
      # folder that is created by the OS or a dependency of your package.
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
    scripts:
      preinstall: "scripts/preinstall.sh"
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"
      postremove: "scripts/postremove.sh"

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
      # Default: first line of the description
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

      # Compression algorithm (gzip (default), lzma or xz).
      compression: lzma

      # The package is signed if a key_file is set
      signature:
        # PGP secret key file path (can also be ASCII-armored).
        # The passphrase is taken from the environment variable
        # `$NFPM_ID_RPM_PASSPHRASE` with a fallback to `$NFPM_ID_PASSPHRASE`,
        # where ID is the id of the current nfpm config.
        # The id will be transformed to uppercase.
        # E.g. If your nfpm id is 'default' then the rpm-specific passphrase
        # should be set as `$NFPM_DEFAULT_RPM_PASSPHRASE`
        #
        # Templates: allowed
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

      # The package is signed if a key_file is set
      signature:
        # PGP secret key file path (can also be ASCII-armored).
        # The passphrase is taken from the environment variable
        # `$NFPM_ID_DEB_PASSPHRASE` with a fallback to `$NFPM_ID_PASSPHRASE`,
        # where ID is the id of the current nfpm config.
        # The id will be transformed to uppercase.
        # E.g. If your nfpm id is 'default' then the deb-specific passphrase
        # should be set as `$NFPM_DEFAULT_DEB_PASSPHRASE`
        #
        # Templates: allowed
        key_file: "{{ .Env.GPG_KEY_PATH }}"

        # The type describes the signers role, possible values are "origin",
        # "maint" and "archive".
        #
        # Default: 'origin'
        type: origin

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
        # The passphrase is taken from the environment variable
        # `$NFPM_ID_APK_PASSPHRASE` with a fallback to `$NFPM_ID_PASSPHRASE`,
        # where ID is the id of the current nfpm config.
        # The id will be transformed to uppercase.
        # E.g. If your nfpm id is 'default' then the apk-specific passphrase
        # should be set as `$NFPM_DEFAULT_APK_PASSPHRASE`
        #
        # Templates: allowed
        key_file: "{{ .Env.GPG_KEY_PATH }}"

        # The name of the signing key. When verifying a package, the signature
        # is matched to the public key store in /etc/apk/keys/<key_name>.rsa.pub.
        #
        # Default: maintainer's email address
        # Templates: allowed (since v1.15)
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
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).

!!! info

    Fields marked with "overridable" can be overridden for any format.

## A note about Termux

Termux is the same format as `deb`, the differences are:

- it uses a different `bindir` (prefixed with `/data/data/com.termux/files/`)
- it uses slightly different architecture names than Debian
