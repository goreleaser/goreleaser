---
title: "Alpine Linux packages"
linkTitle: APKBUILD
weight: 125
---

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
an `APKBUILD` file to an _Alpine Linux_ repository.

> [!WARNING]
> Before going further on this, make sure to read
> [Alpine's APKBUILD Reference](https://wiki.alpinelinux.org/wiki/APKBUILD_Reference)
> and [Creating an Alpine package](https://wiki.alpinelinux.org/wiki/Creating_an_Alpine_package).

This page describes the available options.

```yaml {filename=".goreleaser.yaml"}
apkbuilds:
  - # The package name.
    #
    # Default: ProjectName.
    name: my-app

    # Artifact IDs to filter for.
    # Empty means all IDs (no filter).
    ids:
      - foo
      - bar

    # Your app's homepage.
    #
    # Default: inferred from global metadata.
    homepage: "https://example.com/"

    # Your app's description.
    #
    # Templates: allowed.
    description: "A CLI tool."

    # Your app's license.
    #
    # Default: inferred from global metadata.
    license: "MIT"

    # The APKBUILD `pkgrel`.
    #
    # Default: `0`.
    rel: "0"

    # Template for the URL to the downloadable artifacts.
    #
    # Default: inferred from global metadata.
    url_template: "https://github.com/owner/repo/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Maintainer lines rendered as `# Maintainer: ...` comments.
    maintainers:
      - "Foo Bar <foo@example.com>"

    # Contributor lines rendered as `# Contributor: ...` comments.
    contributors:
      - "Baz Qux <baz@example.com>"

    # Runtime dependencies (`depends`).
    depends:
      - "ca-certificates"

    # Build dependencies (`makedepends`).
    makedepends:
      - "build-base"

    # The body of the `package()` function.
    #
    # Templates: allowed.
    package: |
      install -Dm755 ./my-app "$pkgdir"/usr/bin/my-app

    # The `goamd64` value to filter artifacts by (only relevant when building
    # with `goamd64`).
    #
    # Default: v1.
    goamd64: v1

    # Git repository to push the APKBUILD to.
    #
    # This can be a full URL or an SCP-like string.
    # Default: inferred from global metadata.
    git_url: "git@git.alpinelinux.org:aports/community/my-app.git"

    # SSH command to use.
    #
    # Default: inferred from global metadata.
    git_ssh_command: "ssh -o StrictHostKeyChecking=no"

    # Private key to use when pushing to the Git repository.
    private_key: "{{ .Env.PRIVATE_KEY }}"

    # Directory where to write the APKBUILD locally before pushing.
    #
    # Default: `apkbuild` inside the dist folder.
    directory: "dist/apkbuild"

    # Commit message template for the generated commit.
    #
    # Default: `Update to {{ .Tag }}`.
    commit_msg_template: "Update to {{ .Tag }}"

    # The commit author.
    commit_author:
      name: "GoReleaser Bot"
      email: "bot@goreleaser.com"

    # Whether to skip pushing the APKBUILD.
    #
    # Can be a boolean or a template.
    # Default: false.
    skip_upload: true

    # Whether to disable this pipe.
    #
    # Can be a boolean or a template.
    disable: false
```

> [!NOTE]
> APKBUILD scripts must be pure POSIX `sh` (not Bash), which is the key
> difference from AUR's PKGBUILD. Make sure your `package()` body follows
> Alpine's conventions.
