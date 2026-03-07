# Arch User Repositories (Sources)

<!-- md:version v2.5 -->

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a `PKGBUILD` to an _Arch User Repository_ based on sources.

!!! warning

    Before going further on this, make sure to read
    [AUR's Submission Guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines).

This page describes the available options.

```yaml title=".goreleaser.yaml"
aur_sources:
  - # The package name.
    #
    # Note that since this integration creates a PKGBUILD to build from
    # source, per Arch's guidelines.
    # That said, GoReleaser will remove `-bin` suffix if its present.
    #
    # Default: ProjectName.
    name: package

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
    git_url: "ssh://aur@aur.archlinux.org/mypackage.git"

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
      - myapp

    # List of packages that conflict with, or cause problems with the package.
    #
    # Default: the project name.
    conflicts:
      - myapp

    # List of packages that must be installed to install this.
    depends:
      - curl

    # List of packages that are not needed for the software to function,
    # but provide additional features.
    #
    # Must be in the format `package: short description of the extra functionality`.
    optdepends:
      - "wget: for downloading things"

    # List of packages that must be installed to build this.
    # Default: ["go", "git"]
    makedepends:
      - make

    # List of files that can contain user-made changes and should be preserved
    # during package upgrades and removals.
    backup:
      - /etc/foo.conf

    # Custom prepare instructions.
    prepare: |-
      cd "${pkgname}_${pkgver}"
      go mod download

    # Custom build instructions.
    build: |-
      cd "${pkgname}_${pkgver}"
      export CGO_CPPFLAGS="${CPPFLAGS}"
      export CGO_CFLAGS="${CFLAGS}"
      export CGO_CXXFLAGS="${CXXFLAGS}"
      export CGO_LDFLAGS="${LDFLAGS}"
      export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
      go build -ldflags="-w -s -buildid='' -linkmode=external -X main.version=${pkgver}" .
      chmod +x ./goreleaser

    # Custom package instructions.
    package: |-
      cd "${pkgname}_${pkgver}"
      install -Dsm755 ./myapp "${pkgdir}/usr/bin/myapp"

    # This will be added into the package as 'name.install'.
    # In this file, you may define functions like `pre_install`, `post_install`,
    # and so on.
    #
    # <!-- md:inline_version v2.8 -->.
    install: ./scripts/install.sh

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

    # Whether to disable this particular AUR configuration.
    #
    # Templates: allowed.
    # <!-- md:inline_version v2.8 -->.
    disable: "{{ .IsSnapshot }}"

{% include-markdown "../includes/commit_author.md" comments=false start='---\n\n' %}
```

<!-- md:templates -->

!!! tip

    For more info about what each field does, please refer to
    [Arch's PKGBUILD reference](https://wiki.archlinux.org/title/PKGBUILD).
