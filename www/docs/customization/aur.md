# Arch User Repositories

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a `PKGBUILD` to an _Arch User Repository_.

!!! warning

    Before going further on this, make sure to read
    [AUR's Submission Guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines).

This page describes the available options.

```yaml
# .goreleaser.yaml
aurs:
  - # The package name.
    #
    # Note that since this integration does not create a PKGBUILD to build from
    # source, per Arch's guidelines.
    # That said, GoReleaser will enforce a `-bin` suffix if its not present.
    #
    # Default: ProjectName with a -bin suffix.
    name: package-bin

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
    # Default: inferred from global metadata.
    description: "Software to create fast and easy drum rolls."

    # The maintainers of the package.
    #
    # Default: inferred from global metadata.
    maintainers:
      - "Foo Bar <foo at bar dot com>"

    # The contributors of the package.
    contributors:
      - "Foo Zaz <foo at zaz dot com>"

    # SPDX identifier of your app's license.
    #
    # Default: inferred from global metadata.
    license: "MIT"

    # The SSH private key that should be used to commit to the Git repository.
    # This can either be a path or the key contents.
    #
    # IMPORTANT: the key must not be password-protected.
    #
    # WARNING: do not expose your private key in the configuration file!
    private_key: "{{ .Env.AUR_KEY }}"

    # The AUR Git URL for this package.
    # Publish is skipped if empty.
    git_url: "ssh://aur@aur.archlinux.org/mypackage-bin.git"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # formula - instead, the formula file will be stored on the dist directory
    # only, leaving the responsibility of publishing it to the user.
    #
    # If set to auto, the release will not be uploaded to the AUR repo
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1.
    skip_upload: true

    # List of additional packages that the software provides the features of.
    #
    # Default: the project name.
    provides:
      - mybin

    # List of packages that conflict with, or cause problems with the package.
    #
    # Default: the project name.
    conflicts:
      - mybin

    # List of packages that must be installed to install this.
    depends:
      - curl

    # List of packages that are not needed for the software to function,
    # but provide additional features.
    #
    # Must be in the format `package: short description of the extra functionality`.
    optdepends:
      - "wget: for downloading things"

    # List of files that can contain user-made changes and should be preserved
    # during package upgrades and removals.
    backup:
      - /etc/foo.conf

    # Custom package instructions.
    # which is not always correct.
    #
    # We recommend you override this, installing the binary, license and
    # everything else your package needs.
    #
    # Default: 'install -Dm755 "./PROJECT_NAME" "${pkgdir}/usr/bin/PROJECT_NAME"'.
    package: |-
      # bin
      install -Dm755 "./mybin" "${pkgdir}/usr/bin/mybin"

      # license
      install -Dm644 "./LICENSE.md" "${pkgdir}/usr/share/licenses/mybin/LICENSE"

      # completions
      mkdir -p "${pkgdir}/usr/share/bash-completion/completions/"
      mkdir -p "${pkgdir}/usr/share/zsh/site-functions/"
      mkdir -p "${pkgdir}/usr/share/fish/vendor_completions.d/"
      install -Dm644 "./completions/mybin.bash" "${pkgdir}/usr/share/bash-completion/completions/mybin"
      install -Dm644 "./completions/mybin.zsh" "${pkgdir}/usr/share/zsh/site-functions/_mybin"
      install -Dm644 "./completions/mybin.fish" "${pkgdir}/usr/share/fish/vendor_completions.d/mybin.fish"

      # man pages
      install -Dm644 "./manpages/mybin.1.gz" "${pkgdir}/usr/share/man/man1/mybin.1.gz"

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # Commit message.
    #
    # Default: 'Update to {{ .Tag }}'.
    # Templates: allowed.
    commit_msg_template: "pkgbuild updates"

    # If you build for multiple GOAMD64 versions, you may use this to choose which one to use.
    #
    # Default: 'v1'.
    goamd64: v2

    # The value to be passed to `GIT_SSH_COMMAND`.
    # This is mainly used to specify the SSH private key used to pull/push to
    # the Git URL.
    #
    # Default: 'ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null'.
    git_ssh_command: "ssh -i {{ .Env.KEY }} -o SomeOption=yes"

    # URL which is determined by the given Token
    # (github, gitlab or gitea).
    #
    # Default: depends on the client.
    # Templates: allowed.
    url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

    # Directory in which the files will be created inside the repository.
    # Only useful if you're creating your own AUR with multiple packages in a
    # single repository.
    #
    # Default: '.'.
    # Templates: allowed.
    directory: "."
```

{% include-markdown "../includes/templates.md" comments=false %}

!!! tip

    For more info about what each field does, please refer to
    [Arch's PKGBUILD reference](https://wiki.archlinux.org/title/PKGBUILD).
