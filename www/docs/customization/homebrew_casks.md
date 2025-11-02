# Homebrew Casks

<!-- md:version v2.10 -->

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
    binaries:
      - myapp
      - myapp2

    # App to use instead of the binary.
    # This will then make GoReleaser use only the DMG files instead of archives.
    #
    # Pro only.
    # Templates: allowed.
    app: Foo.app

    # Path to the manpage files.
    #
    # Templates: allowed.
    manpages:
      - man/myapp.1
      - man/myapp-subcmd.1

    # Completions for different shells
    #
    # Templates: allowed.
    completions:
      bash: completions/myapp.bash
      zsh: completions/myapp.zsh
      fish: completions/myapp.fish

    # This information will be used to build the URL section of your Cask.
    #
    # You can set the template, as well as additional parameters.
    # These parameters can be used to provide extra headers, cookies, or other
    # download requirements for your application.
    # See https://docs.brew.sh/Cask-Cookbook#additional-url-parameters for more details.
    #
    # All fields are optional.
    url:
      # URL which is determined by the given Token (github, gitlab or gitea).
      #
      # Default depends on the client.
      # Templates: allowed.
      template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

      # Used when the domains of `url` and `homepage` differ.
      # Templates: allowed.
      verified: "github.com/owner/repo/"

      # Download strategy or format specification
      # See official Cask Cookbook for allowed values.
      # Templates: allowed.
      using: ":homebrew_curl"

      # HTTP cookies to send with the download request
      # Templates: allowed.
      cookies:
        license: "accept-backup"

      # HTTP referer header
      # Templates: allowed.
      referer: "https://example.com/download-page"

      # Additional HTTP headers
      # Templates: allowed.
      headers:
        - "X-Version: {{ .Version }}"

      # Custom User-Agent header
      # Templates: allowed.
      user_agent: "MyApp/1.0 (macOS)"

      # Custom body when using POST request
      # Templates: allowed.
      data:
        format: "dmg"
        platform: "mac"

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
    #
    # This block is placed at the top of the cask definition.
    # It allows you to define custom modules and helper methods
    # for advanced tasks, such as dynamic URL construction.
    # For more information, see: https://docs.brew.sh/Cask-Cookbook#arbitrary-ruby-methods
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

    # Hooks for the cask lifecycle.
    #
    # Templates: allowed (since v2.13-unreleased).
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

## Signing and Notarizing

Casks are supposed to be signed, even if they are coming from a tap.

GoReleaser can [sign and notarize both binaries and apps](./notarize.md), but,
Apple charges a yearly fee for that.

If you don't want to do it, you still have the option to tell macOS to remove
the quarantine bit from the binary on a post install hook:

```yaml title=".goreleaser.yaml"
homebrew_casks:
  - name: foo
    hooks:
      post:
        # replace foo with the actual binary name
        install: |
          if OS.mac?
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/foo"]
          end
```

!!! danger "What happens if I don't follow the steps above?"

    **Not following this might lead to your app/binary to not run.**

    In these cases, users will see the infamous "_App Name is damaged and
    cannot be opened_" alert.

    If you don't want to do any of the steps above, you may want to instruct
    your users to run the appropriate `xattr` command manually.

    You may do so in using the `caveats` property, for example.

!!! warning "xattr bypasses macOS security - use with caution"

    Use of `xattr` to bypass Gatekeeper circumvents macOS security protections
    designed to verify software authenticity. This removes Apple's verification
    layer and requires users to trust the software directly.

    Proper code signing and notarization is Apple's recommended method for
    distributing software. This approach should only be considered when code
    signing is not feasible.

    Important: Apple may disable this bypass method in future macOS versions
    without notice, potentially breaking software distribution that relies on it.

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

To publish a cask from one repository to another using GitHub Actions, you
cannot use the default action token. You must use a separate token with content
write privileges for the tap repository. You can check the
[resource not accessible by integration](https://goreleaser.com/errors/resource-not-accessible-by-integration/)
for more information.

## Private GitHub Repositories

The best way to support private repositories is to add by using a custom block,
a custom template URL, and custom headers.

Here's an example:

!!! warning

    Please note that this example uses an internal Homebrew API to retrieve the GitHub API token.

    Replace with your implementation as needed.

```yaml title=".goreleaser.yaml"
homebrew_casks:
  - name: foo
    custom_block: |
      module GitHubHelper
        def self.token
          require "utils/github"

          # Prefer environment variable if available
          github_token = ENV["HOMEBREW_GITHUB_API_TOKEN"]
          github_token ||= GitHub::API.credentials
          raise "Failed to retrieve github api token" if github_token.nil? || github_token.empty?

          github_token
        end

        def self.release_asset_url(tag, name)
          require "json"
          require "net/http"
          require "uri"

          resp = Net::HTTP.get(
            # Replace with your GitHub repository URL
            URI.parse("https://api.github.com/repos/goreleaser/example/releases/tags/#{tag}"),
            {
              "Accept" => "application/vnd.github+json",
              "Authorization" => "Bearer #{token}",
              "X-GitHub-Api-Version" => "2022-11-28"
            }
          )

          release = JSON.parse(resp)
          release["assets"].find { |asset| asset["name"] == name }["url"]
        end
      end

    url:
      template: '#{GitHubHelper.release_asset_url("{{.Tag}}", "{{.ArtifactName}}")}'
      headers:
        - "Accept: application/octet-stream"
        - "Authorization: Bearer #{GitHubHelper.token}"
        - "X-GitHub-Api-Version: 2022-11-28"
```

{% include-markdown "../includes/prs.md" comments=false start='---\n\n' %}
