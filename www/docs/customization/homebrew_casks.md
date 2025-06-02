# Homebrew Casks

<!-- md:version v2.10-unreleased -->

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _Homebrew Cask_ into a repository (_Tap_) that you have access to.

The `homebrew_casks` section specifies how the cask should be created.
You can check the
[Homebrew Cask documentation](https://docs.brew.sh/Cask-Cookbook),
for more details.

```yaml title=".goreleaser.yaml"
homebrew_casks:
  -
    # Name of the cask
    #
    # Default: the project name.
    # Templates: allowed.
    name: myproject

    # Alternative names for the current cask.
    #
    # Useful if you want to publish a versioned cask as well, so users can
    # more easily downgrade.
    #
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed.
    alternative_names:
      - myproject@{{ .Version }}
      - myproject@{{ .Major }}

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # Binary name inside the cask
    #
    # Default: the cask name.
    # Templates: allowed.
    binary: myapp

    # Path to the manpage file
    #
    # Templates: allowed.
    manpage: man/myapp.1

    # Completions for different shells
    #
    # Templates: allowed.
    completions:
      bash: completions/myapp.bash
      zsh: completions/myapp.zsh
      fish: completions/myapp.fish


    # NOTE: make sure the url_template, the token and given repo (github or
    # gitlab) owner and name are from the same kind.
    # We will probably unify this in the next major version like it is
    # done with scoop.

    # URL which is determined by the given Token (github, gitlab or gitea).
    #
    # Default depends on the client.
    # Templates: allowed.
    url_template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Additional URL parameters for Homebrew Cask downloads.
    # These parameters can be used to provide extra headers, cookies, or other
    # download requirements for your application.
    # See https://docs.brew.sh/Cask-Cookbook#additional-url-parameters for more details.
    #
    # All parameters are optional and will only be included in the generated cask
    # if explicitly configured. No default values are set.
    # Templates: allowed.
    url_additional:
      # Used when the domains of `url` and `homepage` differ.
      verified: "github.com/owner/repo/"

      # Download strategy or format specification
      # See official Cask Cookbook for allowed values.
      using: ":homebrew_curl"

      # HTTP cookies to send with the download request
      cookies:
        license: "accept-backup"

      # HTTP referer header
      referer: "https://example.com/download-page"

      # Additional HTTP headers
      headers:
        - "X-Version: {{ .Version }}"

      # Custom User-Agent header
      user_agent: "MyApp/1.0 (macOS)"

      # Custom body when using POST request
      data:
        format: "dmg"
        platform: "mac"

    # Git author used to commit to the repository.
    # Templates: allowed.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    #
    # Templates: allowed.
    commit_msg_template: "Brew cask update for {{ .ProjectName }} version {{ .Tag }}"

    # Directory inside the repository to put the cask.
    # Default: Casks
    directory: Casks

    # Caveats for the user of your binary.
    caveats: "How to use this binary"

    # Your app's homepage.
    #
    # Default: inferred from global metadata.
    homepage: "https://example.com/"

    # Your app's description.
    #
    # Templates: allowed.
    # Default: inferred from global metadata.
    description: "Software to create fast and easy drum rolls."

    # SPDX identifier of your app's license.
    #
    # Default: inferred from global metadata.
    license: "MIT"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # cask - instead, the cask file will be stored on the dist directory
    # only, leaving the responsibility of publishing it to the user.
    # If set to auto, the release will not be uploaded to the homebrew tap
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    #
    # Templates: allowed.
    skip_upload: true

    # Custom block for brew.
    # Can be used to specify alternate downloads for devel or head releases.
    custom_block: |
      head "https://github.com/some/package.git"
      ...

    # Dependencies for the cask.
    dependencies:
      - cask: some-cask
      - formula: some-formula

    # Packages that conflict with your cask.
    conflicts:
      - cask: some-cask
      - formula: some-formula

    # Hooks for the cask lifecycle.
    hooks:
      pre:
        install: |
          system_command "/usr/bin/defaults", args: ["write", "com.example.app", "key", "value"]
        uninstall: |
          system_command "/usr/bin/defaults", args: ["delete", "com.example.app"]
      post:
        install: |
          system_command "/usr/bin/open", args: ["#{appdir}/MyApp.app"]
        uninstall: |
          system_command "/usr/bin/rm", args: ["-rf", "~/.myapp"]

    # Relative path to a Service that should be moved into the
    # ~/Library/Services folder on installation.
    service: "myapp.service"

    # Additional procedures for a more complete uninstall, including user files
    # and shared resources.
    zap:
      launchctl:
        - "my.fancy.package.service"
      quit:
        - "my.fancy.package"
      login_item:
        - "my.fancy.package"
      trash:
        - "~/.foo/bar"
        - "~/otherfile"
      delete:
        - "~/.foo/bar"
        - "~/otherfile"

    # Procedures to uninstall a cask.
    # Optional unless a pkg or installer artifact stanza is used.
    uninstall:
      launchctl:
        - "my.fancy.package.service"
      quit:
        - "my.fancy.package"
      login_item:
        - "my.fancy.package"
      trash:
        - "~/.foo/bar"
        - "~/otherfile"
      delete:
        - "~/.foo/bar"
        - "~/otherfile"

{% include-markdown "../includes/repository.md" comments=false start='---\n\n' %}
```

<!-- md:templates -->

## Versioned Casks

<!-- md:pro -->

GoReleaser can also create a versioned Cask.
For instance, you might want to make keep previous minor versions available to
your users, so they easily downgrade and/or keep using an older version.

To do that, use `alternative_names`:

```yaml title=".goreleaser.yaml"
homebrew_casks:
  - name: foo
    alternative_names:
      - "foo@{{ .Major }}.{{ .Minor }}"
    # other fields
```

So, if you tag `v1.2.3`, GoReleaser will create and push `foo.rb` and
`foo@1.2.rb`.

Later on, you can tag `v1.3.0`, and then GoReleaser will create and push both
`foo.rb` (thus overriding the previous version) and `foo@1.3.rb`.
Your users can then `brew install foo@1.2` to keep using the previous version.

## GitHub Actions

To publish a cask from one repository to another using GitHub Actions, you cannot use the default action token.
You must use a separate token with content write privileges for the tap repository.
You can check the [resource not accessible by integration](https://goreleaser.com/errors/resource-not-accessible-by-integration/) for more information.

## Limitations

- Only one `GOARM` build is allowed;

{% include-markdown "../includes/prs.md" comments=false start='---\n\n' %}
