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
    # Name template of the recipe
    # Default to project name
    name: myproject

    # IDs of the archives to use.
    # Empty means all IDs.
    #
    # Default: []
    ids:
    - foo
    - bar

    # GOARM to specify which 32-bit arm version to use if there are multiple
    # versions from the build section. Brew formulas support only one 32-bit
    # version.
    # Default is 6 for all artifacts or each id if there a multiple versions.
    goarm: 6

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    # Default is v1.
    goamd64: v3

    # NOTE: make sure the url_template, the token and given repo (github or
    # gitlab) owner and name are from the same kind.
    # We will probably unify this in the next major version like it is
    # done with scoop.

    # GitHub/GitLab repository to push the formula to
    tap:
      # Repository owner template. (templateable)
      owner: user

      # Repository name. (templateable)
      name: homebrew-tap

      # Optionally a branch can be provided. (templateable)
      #
      # Default: default repository branch.
      branch: main

      # Optionally a token can be provided, if it differs from the token
      # provided to GoReleaser
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"

    # Template for the url which is determined by the given Token (github,
    # gitlab or gitea)
    #
    # Default depends on the client.
    url_template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Allows you to set a custom download strategy. Note that you'll need
    # to implement the strategy and add it to your tap repository.
    # Example: https://docs.brew.sh/Formula-Cookbook#specifying-the-download-strategy-explicitly
    # Default is empty.
    download_strategy: CurlDownloadStrategy

    # Allows you to add a custom require_relative at the top of the formula
    # template.
    # Default is empty
    custom_require: custom_download_strategy

    # Git author used to commit to the repository.
    # Defaults are shown.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    commit_msg_template: "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"

    # Folder inside the repository to put the formula.
    # Default is the root folder.
    folder: Formula

    # Caveats for the user of your binary.
    # Default is empty.
    caveats: "How to use this binary"

    # Your app's homepage.
    # Default is empty.
    homepage: "https://example.com/"

    # Template of your app's description.
    # Default is empty.
    description: "Software to create fast and easy drum rolls."

    # SPDX identifier of your app's license.
    # Default is empty.
    license: "MIT"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # formula - instead, the formula file will be stored on the dist folder only,
    # leaving the responsibility of publishing it to the user.
    # If set to auto, the release will not be uploaded to the homebrew tap
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    # Default is false.
    skip_upload: true

    # Custom block for brew.
    # Can be used to specify alternate downloads for devel or head releases.
    # Default is empty.
    custom_block: |
      head "https://github.com/some/package.git"
      ...

    # Packages your package depends on.
    dependencies:
      - name: git
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
    # Default is empty.
    plist: |
      <?xml version="1.0" encoding="UTF-8"?>
      # ...

    # Service block.
    #
    # Since: v1.7
    service: |
      run: foo/bar
      # ...

    # So you can `brew test` your formula.
    # Default is empty.
    test: |
      system "#{bin}/foo --version"
      # ...

    # Custom install script for brew.
    # Default is 'bin.install "the binary name"'.
    install: |
      bin.install "some_other_name"
      bash_completion.install "completions/foo.bash" => "foo"
      # ...

    # Custom post_install script for brew.
    # Could be used to do any additional work after the "install" script
    # Default is empty.
    post_install: |
    	etc.install "app-config.conf"
    	...
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

By defining the `brew` section, GoReleaser will take care of publishing the
Homebrew tap.
Assuming that the current tag is `v1.2.3`, the above configuration will generate a
`program.rb` formula in the `Formula` folder of `user/homebrew-tap` repository:

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
    if Hardware::CPU.intel?
      url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_Linux_64bit.zip"
      sha256 "b41bebd25fd7bb1a67dc2cd5ee12c9f67073094567fdf7b3871f05fd74a45fdd"
    end
    if Hardware::CPU.arm? && !Hardware::CPU.is_64_bit?
      url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_Linux_armv7.zip"
      sha256 "78f31239430eaaec01df783e2a3443753a8126c325292ed8ddb1658ddd2b401d"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_Linux_arm64.zip"
      sha256 "97cadca3c3c3f36388a4a601acf878dd356d6275a976bee516798b72bfdbeecf"
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

## Limitations

- Only one `GOARM` build is allowed;
