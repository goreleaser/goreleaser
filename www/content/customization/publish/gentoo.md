---
title: "Gentoo Ebuilds"
linkTitle: Gentoo
weight: 150
---

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _Gentoo Ebuild_ to an overlay or repository.

The `gentoo_overlay` section specifies how the ebuilds should be created:

```yaml {filename=".goreleaser.yaml"}
gentoo_overlay:
  -
    # Name of the ebuild.
    # Default: the project name.
    # Templates: allowed.
    name: myproject

    # IDs of the archives to use.
    ids:
    - foo
    - bar

    # The ebuild's path within the repository.
    # Templates: allowed.
    path: "app-admin/myproject/myproject-{{ .Version }}.ebuild"

    # Commit message template.
    # Default: "Update to {{ .Tag }}".
    # Templates: allowed.
    commit_msg_template: "pkgbuild updates"

    # Optional keywords to be applied to the package.
    # Default: derived from artifacts architectures
    keywords:
      - "~amd64"
      - "~arm64"

    # Extra files to add to the ebuild files directory.
    files:
      - src: "init.d/myproject"
        dst: "myproject.init"

    # Optional type of the package.
    # Default: ""
    type: "bin"

    # The Gentoo maintainers of the ebuild.
    maintainers:
      - name: "John Doe"
        email: "john@example.com"

    # Upstream bugs tracker URL.
    bugs_to: "https://github.com/myorg/myproject/issues"

    # Project Homepage
    homepage: "https://myproject.com"

    # The ebuild's description.
    # Default: inferred from global metadata.
    # Templates: allowed.
    description: "Software to create fast and easy drum rolls."

    # The ebuild's license.
    # Default: inferred from global metadata.
    license: "MIT"

    # Whether the binaries should be extracted from the archive.
    # This generates an unpack-based ebuild.
    bin: true

    # Keep past versions that were created by goreleaser.
    # Requires an active SCM integration (e.g. GitHub/GitLab token) to work properly.
    keep_versions: 3

    # Disable ignoring size to binary files
    disable_ignore_size_to_binary_files: false

    # Retention strategy, `keep` vs `drop`. Drop will remove old ones.
    version_retention_strategy: keep

    # Whether to skip uploading
    # Templates: allowed.
    skip_upload: true

    # Destination binary directory
    bindir: "/usr/bin"

    # Extra install commands
    extra_install: |
      doins extra/stuff

    # Use flags
    useflags:
      - flag: systemd
        description: "Enable systemd support"

    # Files to install with dobin
    dobin:
      - source: "path/to/bin"
        destination: "mybin"

    # Similarly we have doconfd, dodir, dodoc, doenvd, doexe, doheader, doinitd, doins, doman, dosbin, dosym, systemd
    # e.g.:
    systemd:
      - source: "path/to/service.service"
        destination: "service.service"

{{% g_include file="includes/commit_author.md" %}}
{{% g_include file="includes/repository.md" %}}
```

{{< g_templates >}}

## Limitations

- The target repository needs to have standard Gentoo repo files like `metadata/layout.conf` or they will be generated implicitly.

{{% g_include file="includes/prs.md" %}}
