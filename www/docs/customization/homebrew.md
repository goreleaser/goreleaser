# Homebrew Taps

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _homebrew-tap_ recipe into a repository that you have access to.

The `brews` section specifies how the formula should be created.
You can check the
[Homebrew documentation](https://github.com/Homebrew/brew/blob/master/docs/How-to-Create-and-Maintain-a-Tap.md),
and the
[formula cookbook](https://github.com/Homebrew/brew/blob/master/docs/Formula-Cookbook.md)
for more details.

```yaml
# .goreleaser.yaml
brews:
  -
    # Name of the recipe
    #
    # Default: the project name.
    # Templates: allowed.
    name: myproject

    # Alternative names for the current recipe.
    #
    # Useful if you want to publish a versioned formula as well, so users can
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

    # Sets the app file within a DMG.
    #
    # This feature is only available in GoReleaser Pro.
    app: MyApp.app

    # GOARM to specify which 32-bit arm version to use if there are multiple
    # versions from the build section. Brew formulas support only one 32-bit
    # version.
    #
    # Default: 6.
    goarm: 6

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1.
    goamd64: v1

    # NOTE: make sure the url_template, the token and given repo (github or
    # gitlab) owner and name are from the same kind.
    # We will probably unify this in the next major version like it is
    # done with scoop.

    # URL which is determined by the given Token (github, gitlab or gitea).
    #
    # Default depends on the client.
    # Templates: allowed.
    url_template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Headers to include in the `url` stanza.
    # This can be a more modern alternative to `download_strategy` in some
    # cases.
    url_headers:
      - "Accept: application/octet-stream"
      - 'Authorization: bearer #{ENV["HOMEBREW_GITHUB_API_TOKEN"]}'

    # Allows you to set a custom download strategy. Note that you'll need
    # to implement the strategy and add it to your tap repository.
    # Example: https://docs.brew.sh/Formula-Cookbook#specifying-the-download-strategy-explicitly
    download_strategy: CurlDownloadStrategy

    # Allows you to add a custom require_relative at the top of the formula
    # template.
    custom_require: custom_download_strategy

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    #
    # Templates: allowed.
    commit_msg_template: "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"

    # Directory inside the repository to put the formula.
    directory: Formula

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
    # formula - instead, the formula file will be stored on the dist directory
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

    # Packages your package depends on.
    dependencies:
      - name: git
        # Allow to specify the OS in which the dependency is required.
        # Valid options are `mac` and `linux`.
        os: mac
      - name: zsh
        type: optional
      - name: fish
        version: v1.2.3
      # if providing both version and type, only the type will be taken into
      # account.
      - name: elvish
        type: optional
        version: v1.2.3


    # Packages that conflict with your package.
    conflicts:
      - svn
      - bash

    # Specify for packages that run as a service.
    plist: |
      <?xml version="1.0" encoding="UTF-8"?>
      # ...

    # Service block.
    service: |
      run: foo/bar
      # ...

    # So you can `brew test` your formula.
    #
    # Template: allowed
    test: |
      system "#{bin}/foo --version"
      # ...

    # Custom install script for brew.
    #
    # Template: allowed
    # Default: 'bin.install "BinaryName"'.
    install: |
      bin.install "some_other_name"
      bash_completion.install "completions/foo.bash" => "foo"
      # ...

    # Additional install instructions so you don't need to override `install`.
    #
    # Template: allowed
    extra_install: |
      bash_completion.install "completions/foo.bash" => "foo"
      man1.install "man/foo.1.gz"
      # ...

    # Custom post_install script for brew.
    # Could be used to do any additional work after the "install" script
    post_install: |
    	etc.install "app-config.conf"
      # ...

{% include-markdown "../includes/repository.md" comments=false %}
```

{% include-markdown "../includes/templates.md" comments=false %}

By defining the `brew` section, GoReleaser will take care of publishing the
Homebrew tap.
Assuming that the current tag is `v1.2.3`, the above configuration will generate a
`program.rb` formula in the `Formula` directory of `user/homebrew-tap`
repository:

```rb
class Program < Formula
  desc "How to use this binary"
  homepage "https://github.com/user/repo"
  version "v1.2.3"

  on_macos do
    url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_macOs_64bit.zip"
    sha256 "9ee30fc358fae8d248a2d7538957089885da321dca3f09e3296fe2058e7fff74"
  end

  on_linux
    on_intel do
      url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_Linux_64bit.zip"
      sha256 "b41bebd25fd7bb1a67dc2cd5ee12c9f67073094567fdf7b3871f05fd74a45fdd"
    end
    on_arm do
      if !Hardware::CPU.is_64_bit?
        url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_Linux_armv7.zip"
        sha256 "78f31239430eaaec01df783e2a3443753a8126c325292ed8ddb1658ddd2b401d"
      end
    end
    on_arm do
      if Hardware::CPU.is_64_bit?
        url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_Linux_arm64.zip"
        sha256 "97cadca3c3c3f36388a4a601acf878dd356d6275a976bee516798b72bfdbeecf"
      end
    end
  end

  depends_on "git"
  depends_on "zsh" => :optional

  def install
    bin.install "program"
  end

  def post_install
  	etc.install "app-config.conf"
  end
end
```

!!! info

    Note that GoReleaser does not generate a valid homebrew-core formula.
    The generated formulas are meant to be published as
    [homebrew taps](https://docs.brew.sh/Taps.html), and in their current
    form will not be accepted in any of the official homebrew repositories.

## Head Formulas

GoReleaser does not generate `head` formulas for you, as it may be very different
from one software to another.

Our suggestion is to create a `my-app-head.rb` file on your tap following
[homebrew's documentation](https://docs.brew.sh/Formula-Cookbook#unstable-versions-head).

## Versioned formulas

{% include-markdown "../includes/pro.md" comments=false %}

GoReleaser can also create a versioned formula.
For instance, you might want to make keep previous minor versions available to
your users, so they easily downgrade and/or keep using an older version.

To do that, use `alternative_names`:

```yaml
# .goreleaser.yaml
brews:
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

To publish a formula from one repository to another using GitHub Actions, you cannot use the default action token.
You must use a separate token with content write privileges for the tap repository.
You can check the [resource not accessible by integration](https://goreleaser.com/errors/resource-not-accessible-by-integration/) for more information.

## Limitations

- Only one `GOARM` build is allowed;

{% include-markdown "../includes/prs.md" comments=false %}
