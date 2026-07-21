---
title: "Alpine Linux APKBUILDs"
linkTitle: APKBUILD
weight: 125
---

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and
publish an [`APKBUILD`][apkbuild] that packages prebuilt Linux binaries.

This integration does not create `.apk` packages. To create those directly,
use [nFPM][nfpm] instead.

> [!WARNING]
> Alpine's official `aports` repository has its own package policies and uses
> forks and merge requests for contributions. This integration commits an
> `APKBUILD` to a configured Git repository; it does not create a merge request.

```yaml {filename=".goreleaser.yaml"}
apkbuilds:
  - # Package name.
    #
    # Default: ProjectName.
    name: package

    # Artifact IDs to filter for.
    # Empty means all IDs (no filter).
    # There must be at most one artifact per Alpine architecture.
    ids:
      - foo
      - bar

    # Your app's homepage.
    homepage: "https://example.com/"

    # Your app's description.
    # Templates: allowed.
    description: "Software to create fast and easy drum rolls."

    # Package maintainers.
    maintainers:
      - "Foo Bar <foo@example.com>"

    # Package contributors.
    contributors:
      - "Foo Zaz <zaz@example.com>"

    # SPDX identifier of your app's license.
    license: "MIT"

    # SSH private key used to commit to the Git repository.
    # This can be either a path or the key contents.
    #
    # IMPORTANT: the key must not be password-protected.
    # WARNING: do not expose your private key in the configuration file.
    private_key: "{{ .Env.APKBUILD_KEY }}"

    # Git repository to which the APKBUILD will be published.
    git_url: "ssh://git@example.com/packages/aports.git"

    # Prevent GoReleaser from committing the generated APKBUILD.
    # The file will still be available in the dist directory.
    #
    # If set to auto, prereleases will not be published.
    skip_upload: true

    # Packages for which this package provides an alternative.
    provides:
      - package-cli

    # Runtime dependencies.
    depends:
      - ca-certificates

    # Build-time dependencies.
    makedepends:
      - tar

    # Packages replaced by this package.
    replaces:
      - old-package

    # abuild options.
    #
    # Default: ['!check'], because this integration packages prebuilt binaries.
    options:
      - "!check"

    # Alpine package release number.
    #
    # Default: '0'.
    rel: "0"

    # Custom package instructions.
    #
    # The $srcdir, $pkgdir, and $_source variables are available.
    # By default, GoReleaser installs the binaries into $pkgdir/usr/bin.
    package: |-
      install -Dm755 "$srcdir/mybin" "$pkgdir/usr/bin/mybin"
      install -Dm644 "$srcdir/LICENSE" "$pkgdir/usr/share/licenses/mybin/LICENSE"

    # Commit message.
    #
    # Default: 'Update to {{ .Tag }}'.
    # Templates: allowed.
    commit_msg_template: "apkbuild updates"

    # GOAMD64 version to package when multiple versions are built.
    #
    # Default: 'v1'.
    goamd64: v2

    # Value passed to GIT_SSH_COMMAND.
    #
    # Default: 'ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null'.
    git_ssh_command: "ssh -i {{ .Env.KEY }} -o SomeOption=yes"

    # Download URL for each release artifact.
    #
    # Default: depends on the release client.
    # Templates: allowed.
    url_template: "https://example.com/releases/{{ .Tag }}/{{ .ArtifactName }}"

    # Directory in which APKBUILD will be created inside the Git repository.
    # This is useful for aports-style repositories containing multiple packages.
    #
    # Default: '.'.
    # Templates: allowed.
    directory: "testing/package"

    # Disable this particular APKBUILD configuration.
    # Templates: allowed.
    disable: "{{ .IsSnapshot }}"

{{% g_include file="includes/commit_author.md" %}}
```

Supported architecture mappings are:

| Go          | Alpine        |
|-------------|---------------|
| `amd64`     | `x86_64`      |
| `386`       | `x86`         |
| `arm64`     | `aarch64`     |
| `arm/v6`    | `armhf`       |
| `arm/v7`    | `armv7`       |
| `ppc64le`   | `ppc64le`     |
| `s390x`     | `s390x`       |
| `riscv64`   | `riscv64`     |

{{< g_templates >}}

[apkbuild]: https://wiki.alpinelinux.org/wiki/APKBUILD_Reference
[nfpm]: /customization/package/nfpm/
