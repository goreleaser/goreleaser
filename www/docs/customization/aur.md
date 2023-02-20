# Arch User Repositories

Since: v1.4.

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a `PKGBUILD` to an _Arch User Repository_.

!!! warning
    Before going further on this, make sure to read
    [AUR's Submission Guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines).

This page describes the available options.

```yaml
# .goreleaser.yaml
aurs:
  -
    # The package name.
    #
    # Defaults to the Project Name with a -bin suffix.
    #
    # Note that since this integration does not create a PKGBUILD to build from
    # source, per Arch's guidelines.
    # That said, GoReleaser will enforce a `-bin` suffix if its not present.
    name: package-bin

    # Artifact IDs to filter for.
    #
    # Defaults to empty, which includes all artifacts.
    ids:
      - foo
      - bar

    # Your app's homepage.
    # Default is empty.
    homepage: "https://example.com/"

    # Template of your app's description.
    # Default is empty.
    description: "Software to create fast and easy drum rolls."

    # The maintainers of the package.
    # Defaults to empty.
    maintainers:
      - 'Foo Bar <foo at bar dot com>'

    # The contributors of the package.
    # Defaults to empty.
    contributors:
      - 'Foo Zaz <foo at zaz dot com>'

    # SPDX identifier of your app's license.
    # Default is empty.
    license: "MIT"

    # The SSH private key that should be used to commit to the Git repository.
    # This can either be a path or the key contents.
    #
    # IMPORTANT: the key must not be password-protected.
    #
    # WARNING: do not expose your private key in the configuration file!
    private_key: '{{ .Env.AUR_KEY }}'

    # The AUR Git URL for this package.
    # Defaults to empty
    # Publish is skipped if empty.
    git_url: 'ssh://aur@aur.archlinux.org/mypackage-bin.git'

    # Setting this will prevent goreleaser to actually try to commit the updated
    # formula - instead, the formula file will be stored on the dist folder only,
    # leaving the responsibility of publishing it to the user.
    #
    # If set to auto, the release will not be uploaded to the AUR repo
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1.
    #
    # Default is false.
    skip_upload: true

    # List of additional packages that the software provides the features of.
    #
    # Defaults to the project name.
    provides:
      - mybin

    # List of packages that conflict with, or cause problems with the package.
    #
    # Defaults to the project name.
    conflicts:
      - mybin

    # List of packages that must be installed to install this.
    #
    # Defaults to empty.
    depends:
      - curl

    # List of packages that are not needed for the software to function,
    # but provide additional features.
    #
    # Must be in the format `package: short description of the extra functionality`.
    #
    # Defaults to empty.
    optdepends:
      - 'wget: for downloading things'

    # List of files that can contain user-made changes and should be preserved
    # during package upgrades and removals.
    #
    # Default: empty.
    # Since: v1.12
    backup:
      - /etc/foo.conf

    # Custom package instructions.
    #
    # Defaults to `install -Dm755 "./PROJECT_NAME" "${pkgdir}/usr/bin/PROJECT_NAME",
    # which is not always correct.
    #
    # We recommend you override this, installing the binary, license and
    # everything else your package needs.
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
    # Defaults are shown below.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # Commit message template.
    # Defaults to `Update to {{ .Tag }}`.
    commit_msg_template: "pkgbuild updates"

    # If you build for multiple GOAMD64 versions, you may use this to choose which one to use.
    # Defaults to `v1`.
    goamd64: v2

    # The value to be passed to `GIT_SSH_COMMAND`.
    # This is mainly used to specify the SSH private key used to pull/push to
    # the Git URL.
    #
    # Defaults to `ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null`.
    git_ssh_command: 'ssh -i {{ .Env.KEY }} -o SomeOption=yes'

    # Template for the url which is determined by the given Token
    # (github, gitlab or gitea).
    #
    # Default depends on the client.
    url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

!!! tip
    For more info about what each field does, please refer to
    [Arch's PKGBUILD reference](https://wiki.archlinux.org/title/PKGBUILD).
