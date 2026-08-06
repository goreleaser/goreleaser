---
title: "Gentoo Ebuilds"
linkTitle: Gentoo
weight: 150
---

{{< g_version "v2.17" >}}

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _Gentoo Ebuild_ to an overlay repository.

The `gentoo_overlay` section specifies how the ebuilds should be created:

```yaml {filename=".goreleaser.yaml"}
gentoo_overlay:
  - # ID of the gentoo configuration, must be unique.
    #
    # Default: "default".
    id: myproject

    # Name of the package/ebuild.
    #
    # Default: the project name.
    # Templates: allowed.
    name: myproject

    # IDs of the archives to use.
    # Empty means all archives.
    ids:
      - foo
      - bar

    # The ebuild's relative path within the repository.
    # Must include category and package name (e.g. app-admin/myproject/myproject-{{ .Version }}.ebuild).
    #
    # Default: "app-admin/{{ .Name }}/{{ .Name }}-{{ .Version }}.ebuild" (or "app-misc/..." for non-bin).
    # Templates: allowed.
    path: "app-admin/myproject/myproject-{{ .Version }}.ebuild"

    # Commit message template.
    #
    # Default: '{{ .ProjectName }}: bump to {{ .Tag }}'.
    # Templates: allowed.
    commit_msg_template: "{{ .ProjectName }}: bump to {{ .Tag }}"

    # Keywords to be applied to the package.
    #
    # Default: ["~amd64"].
    keywords:
      - "~amd64"
      - "~arm64"

    # Extra files to add to the ebuild's files/ directory.
    files:
      - src: "init.d/myproject"
        dst: "myproject.init"

    # Skip size (< 20KB) and binary file checks for files in the files/ directory.
    #
    # Default: false.
    skip_files_validation: false

    # Whether the ebuild installs pre-compiled binary packages (unpacking binary release artifacts).
    #
    # Note: GoReleaser currently generates unpack-based binary ebuilds.
    # Required: must be true.
    bin: true

    # The ebuild type variant.
    #
    # When set to "bin" (default), GoReleaser automatically appends a `-bin` suffix
    # to the default package name and ebuild path (e.g. `app-misc/myproject-bin/myproject-bin-{{ .Version }}.ebuild`),
    # and sets the default binary installation directory `bindir` to `/opt/bin`.
    #
    # Default: "bin".
    type: "bin"

    # Keep past versions that were created by GoReleaser.
    # Requires an active SCM integration (e.g. GitHub/GitLab token) to work properly.
    #
    # Default: 0.
    keep_versions: 3

    # Retention strategy for old ebuild versions.
    #
    # Valid options: keep_latest, keep_prereleases.
    # Required if keep_versions > 0.
    version_retention_strategy: keep_latest

    # Whether to skip uploading the ebuild to the repository.
    #
    # Valid options: true, false, auto.
    # Default: false.
    # Templates: allowed.
    skip_upload: false

    # Destination binary installation directory.
    #
    # Default: "/opt/bin" if type is "bin", otherwise "/usr/bin".
    bindir: "/usr/bin"

    # The Gentoo maintainers of the ebuild.
    maintainers:
      - name: "John Doe"
        email: "john@example.com"

    # Upstream bug tracker URL.
    #
    # Templates: allowed.
    bugs_to: "https://github.com/myorg/myproject/issues"

    # Project homepage.
    #
    # Default: inferred from global metadata.
    # Templates: allowed.
    homepage: "https://myproject.com"

    # The ebuild's description.
    #
    # Default: inferred from global metadata.
    # Templates: allowed.
    description: "Software to create fast and easy drum rolls."

    # The ebuild's license.
    #
    # Required.
    # Default: inferred from global metadata.
    # Templates: allowed.
    license: "MIT"

    # Extra ebuild installation instructions.
    #
    # Templates: allowed.
    extra_install: |
      doins extra/stuff

    # USE flags defined for the ebuild.
    useflags:
      - flag: systemd
        description: "Enable systemd support"

    # Files to install with dobin.
    dobin:
      - src: "path/to/bin"
        dst: "mybin"
        use:
          - systemd

    # Files to install with doconfd.
    doconfd:
      - src: "path/to/conf"
        dst: "myconf"

    # Directories to create with dodir.
    dodir:
      - "/var/lib/myproject"

    # Documentation files to install with dodoc.
    dodoc:
      - "README.md"

    # Files to install with doenvd.
    doenvd:
      - src: "path/to/env"
        dst: "50myproject"

    # Executable files to install with doexe.
    doexe:
      - src: "path/to/script.sh"
        dst: "script.sh"

    # Header files to install with doheader.
    doheader:
      - src: "path/to/header.h"
        dst: "header.h"

    # Init scripts to install with doinitd.
    doinitd:
      - src: "path/to/init"
        dst: "myproject"

    # General files to install with doins.
    doins:
      - src: "path/to/file"
        dst: "file"

    # Man pages to install with doman.
    doman:
      - "man/myproject.1"

    # System binaries to install with dosbin.
    dosbin:
      - src: "path/to/sbin"
        dst: "mysbin"

    # Symlinks to create with dosym.
    dosym:
      - src: "/usr/bin/myproject"
        dst: "/usr/bin/myproject-alias"

    # Systemd service files to install.
    systemd:
      - src: "path/to/service.service"
        dst: "service.service"

{{% g_include file="includes/commit_author.md" %}}
{{% g_include file="includes/repository.md" %}}
```

{{< g_templates >}}

## Limitations

- The target repository needs to have standard Gentoo repo files like `metadata/layout.conf` or they will be generated implicitly.

{{% g_include file="includes/prs.md" %}}

