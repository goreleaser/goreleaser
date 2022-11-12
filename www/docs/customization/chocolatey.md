# Chocolatey Packages

GoReleaser can also generate `nupkg` packages.
[Chocolatey](http://chocolatey.org/) are packages based on `nupkg` format, that
will let you publish your project directly to the Chocolatey Repository. From
there it will be able to install locally or in Windows distributions.

You can read more about it in the [chocolatey docs](https://docs.chocolatey.org/).

Available options:

```yaml
# .goreleaser.yaml
chocolateys:
  -
    # Your app's package name.
    # The value may not contain spaces or character that are not valid for a URL.
    # If you want a good separator for words, use '-', not  '.'.
    #
    # Defaults to `ProjectName`.
    name: foo

    # IDs of the archives to use.
    # Defaults to empty, which includes all artifacts.
    ids:
      - foo
      - bar

    # Your app's owner.
    # It basically means your.
    # Defaults empty.
    owners: Drum Roll Inc

    # The app's title.
    # A human-friendly title of the package.
    # Defaults to `ProjectName`.
    title: Foo Bar

    # Your app's authors (probably you).
    # Defaults are shown below.
    authors: Drummer

    # Your app's project url.
    # It is a required field.
    project_url: https://example.com/

    # Template for the url which is determined by the given Token (github,
    # gitlab or gitea)
    # Default depends on the client.
    url_template: "https://github.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # App's icon.
    # Default is empty.
    icon_url: 'https://rawcdn.githack.com/foo/bar/efbdc760-395b-43f1-bf69-ba25c374d473/icon.png'

    # Your app's copyright details.
    # Default is empty.
    copyright: 2022 Drummer Roll Inc

    # App's license information url.
    license_url: https://github.com/foo/bar/blob/main/LICENSE

    # Your apps's require license acceptance:
    # Specify whether the client must prompt the consumer to accept the package
    # license before installing.
    # Default is false.
    require_license_acceptance: false

    # Your app's source url.
    # Default is empty.
    project_source_url: https://github.com/foo/bar

    # Your app's documentation url.
    # Default is empty.
    docs_url: https://github.com/foo/bar/blob/main/README.md

    # App's bugtracker url.
    # Default is empty.
    bug_tracker_url: https://github.com/foo/barr/issues

    # Your app's tag list.
    # Default is empty.
    tags: "foo bar baz"

    # Your app's summary:
    summary: Software to create fast and easy drum rolls.

    # This the description of your chocolatey package.
    # Supports markdown.
    description: |
      {{ .ProjectName }} installer package.
      Software to create fast and easy drum rolls.

    # Your app's release notes.
    # A description of the changes made in this release of the package.
    # Supports markdown. To prevent the need to continually update this field,
    # providing a URL to an external list of Release Notes is perfectly
    # acceptable.
    # Default is empty.
    release_notes: "https://github.com/foo/bar/releases/tag/v{{ .Version }}"

    # App's dependencies
    # Default is empty. Version is not required.
    dependencies:
      - id: nfpm
        version: 2.20.0

    # The api key that should be used to push to the chocolatey repository.
    #
    # WARNING: do not expose your api key in the configuration file!
    api_key: '{{ .Env.CHOCOLATEY_API_KEY }}'

    # The source repository that will push the package to.
    #
    # Defaults are shown below.
    source_repo: "https://push.chocolatey.org/"

    # Setting this will prevent goreleaser to actually try to push the package
    # to chocolatey repository, leaving the responsability of publishing it to
    # the user.
    skip_publish: false

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    # Default is v1.
    goamd64: v1
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

!!! note
    GoReleaser will not install `chocolatey` nor any of its dependencies for you.
