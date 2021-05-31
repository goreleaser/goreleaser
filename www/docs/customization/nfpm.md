---
title: NFPM
---

GoReleaser can be wired to [nfpm](https://github.com/goreleaser/nfpm) to
generate and publish `.deb`, `.rpm` and `.apk` packages.

Available options:

```yaml
# .goreleaser.yml
nfpms:
  # note that this is an array of nfpm configs
  -
    # ID of the nfpm config, must be unique.
    # Defaults to "default".
    id: foo

    # Name of the package.
    # Defaults to `ProjectName`.
    package_name: foo

    # You can change the file name of the package.
    # Default: `{{ .PackageName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}`
    file_name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # Build IDs for the builds you want to create NFPM packages for.
    # Defaults to all builds.
    builds:
      - foo
      - bar

    # Replacements for GOOS and GOARCH in the package name.
    # Keys should be valid GOOSs or GOARCHs.
    # Values are the respective replacements.
    # Default is empty.
    replacements:
      amd64: 64-bit
      386: 32-bit
      darwin: macOS
      linux: Tux

    # Your app's vendor.
    # Default is empty.
    vendor: Drum Roll Inc.
    # Your app's homepage.
    # Default is empty.
    homepage: https://example.com/

    # Your app's maintainer (probably you).
    # Default is empty.
    maintainer: Drummer <drum-roll@example.com>

    # Your app's description.
    # Default is empty.
    description: Software to create fast and easy drum rolls.

    # Your app's license.
    # Default is empty.
    license: Apache 2.0

    # Formats to be generated.
    formats:
      - apk
      - deb
      - rpm

    # Packages your package depends on.
    dependencies:
      - git
      - zsh

    # Packages your package recommends installing.
    recommends:
      - bzr
      - gtk

    # Packages your package suggests installing.
    suggests:
      - cvs
      - ksh

    # Packages that conflict with your package.
    conflicts:
      - svn
      - bash

    # Packages it replaces.
    replaces:
      - fish

    # Override default /usr/local/bin destination for binaries
    bindir: /usr/bin

    # Version Epoch.
    # Default is extracted from `version` if it is semver compatible.
    epoch: 2

    # Version Prerelease.
    # Default is extracted from `version` if it is semver compatible.
    prerelease: beta1

    # Version Metadata (previously deb.metadata).
    # Default is extracted from `version` if it is semver compatible.
    # Setting metadata might interfere with version comparisons depending on the packager.
    version_metadata: git

    # Version Release.
    release: 1

    # Section.
    section: default

    # Priority.
    priority: extra

    # Makes a meta package - an empty package that contains only supporting files and dependencies.
    # When set to `true`, the `builds` option is ignored.
    # Defaults to false.
    meta: true

    # Empty folders that should be created and managed by the packager
    # implementation.
    # Default is empty.
    empty_folders:
      - /var/log/foobar

    # Contents to add to the package.
    # GoReleaser will automatically add the binaries.
    contents:
      # Basic file that applies to all packagers
      - src: path/to/local/foo
        dst: /usr/local/bin/foo

      # Simple config file
      - src: path/to/local/foo.conf
        dst: /etc/foo.conf
        type: config

      # Simple symlink.
      # Corresponds to `ln -s /sbin/foo /usr/local/bin/foo`
      - src: /sbin/foo
        dst: /usr/local/bin/foo
        type: "symlink"

      # Corresponds to `%config(noreplace)` if the packager is rpm, otherwise it is just a config file
      - src: path/to/local/bar.conf
        dst: /etc/bar.conf
        type: "config|noreplace"

      # The src and dst attributes also supports name templates
      - src: path/{{ .Os }}-{{ .Arch }}/bar.conf
        dst: /etc/foo/bar-{{ .ProjectName }}.conf

      # These files are not actually present in the package, but the file names
      # are added to the package header. From the RPM directives documentation:
      #
      # "There are times when a file should be owned by the package but not
      # installed - log files and state files are good examples of cases you might
      # desire this to happen."
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

      # You can user the packager field to add files that are unique to a specific packager
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

      # Sometimes it is important to be able to set the mtime, mode, owner, or group for a file
      # that differs from what is on the local build system at build time.
      - src: path/to/foo
        dst: /usr/local/foo
        file_info:
          mode: 0644
          mtime: 2008-01-02T15:04:05Z
          owner: notRoot
          group: notRoot

    # Scripts to execute during the installation of the package.
    # Keys are the possible targets during the installation process
    # Values are the paths to the scripts which will be executed
    scripts:
      preinstall: "scripts/preinstall.sh"
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"
      postremove: "scripts/postremove.sh"

    # Some attributes can be overridden per package format.
    overrides:
      deb:
        conflicts:
          - subversion
        dependencies:
          - git
        suggests:
          - gitk
        recommends:
          - tig
        replaces:
          - bash
        empty_folders:
          - /var/log/bar
      rpm:
        replacements:
          amd64: x86_64
        file_name_template: "{{ .ProjectName }}-{{ .Version }}-{{ .Arch }}"
        files:
          "tmp/man.gz": "/usr/share/man/man8/app.8.gz"
        config_files:
          "tmp/app_generated.conf": "/etc/app-rpm.conf"
        scripts:
          preinstall: "scripts/preinstall-rpm.sh"

    # Custom configuration applied only to the RPM packager.
    rpm:
      # RPM specific scripts.
      scripts:
        # The pretrans script runs before all RPM package transactions / stages.
        pretrans: ./scripts/pretrans.sh
        # The posttrans script runs after all RPM package transactions / stages.
        posttrans: ./scripts/posttrans.sh

      # The package summary.
      # Defaults to the first line of the description.
      summary: Explicit Summary for Sample Package

      # The package group. This option is deprecated by most distros
      # but required by old distros like CentOS 5 / EL 5 and earlier.
      group: Unspecified

      # Compression algorithm.
      compression: lzma

      # These config files will not be replaced by new versions if they were
      # changed by the user. Corresponds to %config(noreplace).
      config_noreplace_files:
        path/to/local/bar.con: /etc/bar.conf

      # These files are not actually present in the package, but the file names
      # are added to the package header. From the RPM directives documentation:
      #
      # "There are times when a file should be owned by the package but not
      # installed - log files and state files are good examples of cases you might
      # desire this to happen."
      #
      # "The way to achieve this, is to use the %ghost directive. By adding this
      # directive to the line containing a file, RPM will know about the ghosted
      # file, but will not add it to the package."
      ghost_files:
        - /etc/casper.conf
        - /var/log/boo.log

      # The package is signed if a key_file is set
      signature:
        # PGP secret key (can also be ASCII-armored). The passphrase is taken
        # from the environment variable $NFPM_ID_RPM_PASSPHRASE with a fallback
        # to $NFPM_ID_PASSPHRASE, where ID is the id of the current nfpm config.
        # The id will be transformed to uppercase.
        # E.g. If your nfpm id is 'default' then the rpm-specific passphrase
        # should be set as $NFPM_DEFAULT_RPM_PASSPHRASE
        key_file: key.gpg

    # Custom configuration applied only to the Deb packager.
    deb:
      # Custom deb special files.
      scripts:
        # Deb rules script.
        rules: foo.sh
        # Deb templates file, when using debconf.
        templates: templates

      # Custom deb triggers
      triggers:
        # register interrest on a trigger activated by another package
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
        # PGP secret key (can also be ASCII-armored). The passphrase is taken
        # from the environment variable $NFPM_ID_DEB_PASSPHRASE with a fallback
        # to $NFPM_ID_PASSPHRASE, where ID is the id of the current nfpm config.
        # The id will be transformed to uppercase.
        # E.g. If your nfpm id is 'default' then the deb-specific passphrase
        # should be set as $NFPM_DEFAULT_DEB_PASSPHRASE
        key_file: key.gpg
        # The type describes the signers role, possible values are "origin",
        # "maint" and "archive". If unset, the type defaults to "origin".
        type: origin

    apk:
      # The package is signed if a key_file is set
      signature:
        # RSA private key in the PEM format. The passphrase is taken
        # from the environment variable $NFPM_ID_APK_PASSPHRASE with a fallback
        # to $NFPM_ID_PASSPHRASE, where ID is the id of the current nfpm config.
        # The id will be transformed to uppercase.
        # E.g. If your nfpm id is 'default' then the deb-specific passphrase
        # should be set as $NFPM_DEFAULT_APK_PASSPHRASE
        key_file: key.gpg
        # The name of the signing key. When verifying a package, the signature
        # is matched to the public key store in /etc/apk/keys/<key_name>.rsa.pub.
        # If unset, it defaults to the maintainer email address.
        key_name: origin
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
