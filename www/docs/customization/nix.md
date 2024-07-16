# Nixpkgs

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _nixpkg_ to a [Nix User Repository][nur].

The `nix` section specifies how the pkgs should be created:

```yaml
# .goreleaser.yaml
nix:
  - #
    # Name of the recipe
    #
    # Default: the project name.
    # Templates: allowed.
    name: myproject

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1.
    goamd64: v1

    # URL which is determined by the given Token (github, gitlab or gitea).
    #
    # Default depends on the client.
    # Templates: allowed.
    url_template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    #
    # Templates: allowed.
    commit_msg_template: "{{ .ProjectName }}: {{ .Tag }}"

    # Path for the file inside the repository.
    #
    # Default: pkgs/<name>/default.nix.
    # Templates: allowed.
    path: pkgs/foo.nix

    # Your app's homepage.
    #
    # Templates: allowed.
    # Default: inferred from global metadata.
    homepage: "https://example.com/"

    # Your app's description.
    #
    # Templates: allowed.
    # Default: inferred from global metadata.
    description: "Software to create fast and easy drum rolls."

    # License name.
    #
    # Default: inferred from global metadata.
    license: "mit"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # package - instead, it will be stored on the dist directory only,
    # leaving the responsibility of publishing it to the user.
    #
    # If set to auto, the release will not be uploaded to the repository
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    #
    # Templates: allowed.
    skip_upload: true

    # Runtime dependencies of the package.
    dependencies:
    - zsh
    - chromium
    - name: bash
      os: linux
    - name: fish
      os: darwin

    # Custom install script.
    #
    # Default: 'mkdir -p $out/bin; cp -vr $binary $out/bin/$binary', and
    #   `makeWrapper` if `dependencies` were provided.
    # Templates: allowed.
    install: |
      mkdir -p $out/bin
      cp -vr ./foo $out/bin/foo

    # Custom additional install instructions.
    # This has the advantage of preventing you to rewrite the `install` script
    # if the defaults work for you.
    #
    # Templates: allowed.
    extra_install: |
      installManPage ./manpages/foo.1.gz

    # Custom post_install script.
    # Could be used to do any additional work after the "install" script
    #
    # Templates: allowed.
    post_install: |
      installShellCompletion ./completions/*

{% include-markdown "../includes/repository.md" comments=false %}
```

{% include-markdown "../includes/templates.md" comments=false %}

## Things not supported

- Generating packages that compile from source (using `buildGoModule`)
- Generating packages when `archives.format` is `binary`

## Dependencies

### `nix-prefetch-url`

The `nix-prefetch-url` binary must be available in the `$PATH` for the
publishing to work.

[iss4034]: https://github.com/goreleaser/goreleaser/issues/4034

### GitHub Actions

To publish a package from one repository to another using GitHub Actions, you
cannot use the default action token.
You must use a separate token with content write privileges for the tap
repository.
You can check the
[resource not accessible by integration](../errors/resource-not-accessible-by-integration.md)
for more information.

## Setting up a NUR

To set up a Nix User Repository, follow the instructions in their
[repository][nur].

Then, you'll need to:

- publish a release with GoReleaser: it should create the package at
  `./pkgs/{name}/default.nix` or whatever path you set it up to
- make sure `./flake.nix` is correct with what you want, especially the
  `systems` bit
- add your package to `./default.nix`
- edit your `README.md` removing the template stuff

That's it!

[nur]: https://github.com/nix-community/NUR

{% include-markdown "../includes/prs.md" comments=false %}
